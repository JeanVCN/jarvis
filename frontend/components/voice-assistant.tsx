"use client";

import { useEffect, useRef, useState } from "react";

import {
  audioBase64ToBlob,
  getVoiceStatus,
  sendVoiceConversation,
} from "@/services/voice";
import type { AssistantState, VoiceStatus } from "@/types/voice";

const barCount = 28;
const particleCount = 54;
const idleLevels = Array.from({ length: barCount }, (_, index) =>
  0.18 + (index % 5) * 0.035,
);
const particles = Array.from({ length: particleCount }, (_, index) => ({
  angle: (360 / particleCount) * index,
  radius: 35 + (index % 9) * 3.2,
  size: 2 + (index % 4) * 0.6,
  delay: -(index % 12) * 0.18,
}));

type VoiceAssistantProps = {
  onOpenText: () => void;
  onOpenSystem: () => void;
};

export function VoiceAssistant({
  onOpenText,
  onOpenSystem,
}: VoiceAssistantProps) {
  const [assistantState, setAssistantState] = useState<AssistantState>("idle");
  const [levels, setLevels] = useState(idleLevels);
  const [transcript, setTranscript] = useState("");
  const [response, setResponse] = useState("");
  const [error, setError] = useState("");
  const [isSupported, setIsSupported] = useState<boolean | null>(null);
  const [voiceStatus, setVoiceStatus] = useState<VoiceStatus | null>(null);
  const [now, setNow] = useState<Date | null>(null);

  const recorderRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  const audioContextRef = useRef<AudioContext | null>(null);
  const animationFrameRef = useRef<number | null>(null);
  const objectUrlRef = useRef<string | null>(null);

  async function toggleListening() {
    if (voiceStatus && (!voiceStatus.sttAvailable || !voiceStatus.ttsAvailable)) {
      setError(
        `Voz local indisponivel. STT: ${voiceStatus.sttCommand || "nao configurado"}, TTS: ${voiceStatus.ttsCommand || "nao configurado"}.`,
      );
      return;
    }

    if (assistantState === "listening") {
      recorderRef.current?.stop();
      return;
    }

    if (assistantState === "thinking" || assistantState === "speaking") {
      return;
    }

    await startListening();
  }

  async function startListening() {
    setError("");
    setTranscript("");
    setResponse("");

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;
      chunksRef.current = [];

      const recorder = new MediaRecorder(stream, {
        mimeType: pickMimeType(),
      });
      recorderRef.current = recorder;

      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          chunksRef.current.push(event.data);
        }
      };
      recorder.onstop = () => {
        void submitRecording();
      };

      recorder.start();
      setAssistantState("listening");
      startInputMeter(stream);
    } catch {
      setError("Nao foi possivel acessar o microfone.");
      setAssistantState("idle");
      stopTracks();
    }
  }

  async function submitRecording() {
    stopMeter();
    stopTracks();

    const audio = new Blob(chunksRef.current, { type: pickMimeType() });
    if (audio.size === 0) {
      setAssistantState("idle");
      setError("Nenhum audio foi capturado.");
      return;
    }

    setAssistantState("thinking");
    setLevels(idleLevels.map((level) => level + 0.12));

    try {
      const result = await sendVoiceConversation(audio);
      setTranscript(result.transcript);
      setResponse(result.response);
      await playResponse(result.audioBase64, result.audioContentType);
    } catch {
      setError("Nao foi possivel completar a conversa por voz.");
      setAssistantState("idle");
      setLevels(idleLevels);
    }
  }

  async function playResponse(base64: string, contentType: string) {
    const blob = audioBase64ToBlob(base64, contentType);
    if (objectUrlRef.current) {
      URL.revokeObjectURL(objectUrlRef.current);
    }
    objectUrlRef.current = URL.createObjectURL(blob);

    const audio = new Audio(objectUrlRef.current);
    setAssistantState("speaking");
    startOutputMeter(audio);

    await audio.play();
    await new Promise<void>((resolve) => {
      audio.onended = () => resolve();
      audio.onerror = () => resolve();
    });

    stopMeter();
    setAssistantState("idle");
    setLevels(idleLevels);
  }

  function startInputMeter(stream: MediaStream) {
    const audioContext = new AudioContext();
    const source = audioContext.createMediaStreamSource(stream);
    const analyser = audioContext.createAnalyser();
    analyser.fftSize = 128;
    source.connect(analyser);
    audioContextRef.current = audioContext;
    animateAnalyser(analyser);
  }

  function startOutputMeter(audio: HTMLAudioElement) {
    const audioContext = new AudioContext();
    const source = audioContext.createMediaElementSource(audio);
    const analyser = audioContext.createAnalyser();
    analyser.fftSize = 128;
    source.connect(analyser);
    analyser.connect(audioContext.destination);
    audioContextRef.current = audioContext;
    animateAnalyser(analyser);
  }

  function animateAnalyser(analyser: AnalyserNode) {
    const data = new Uint8Array(analyser.frequencyBinCount);

    function frame() {
      analyser.getByteFrequencyData(data);
      const nextLevels = Array.from({ length: barCount }, (_, index) => {
        const value = data[index % data.length] / 255;
        return Math.max(0.12, Math.min(1, value));
      });
      setLevels(nextLevels);
      animationFrameRef.current = window.requestAnimationFrame(frame);
    }

    frame();
  }

  function stopMeter() {
    if (animationFrameRef.current) {
      window.cancelAnimationFrame(animationFrameRef.current);
      animationFrameRef.current = null;
    }
    if (audioContextRef.current) {
      void audioContextRef.current.close();
      audioContextRef.current = null;
    }
  }

  function stopTracks() {
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
  }

  useEffect(() => {
    const initialization = window.setTimeout(() => {
      setIsSupported(supportsVoiceInput());
      setNow(new Date());
    }, 0);

    void getVoiceStatus()
      .then(setVoiceStatus)
      .catch(() => {
        setVoiceStatus(null);
      });

    const timer = window.setInterval(() => {
      setNow(new Date());
    }, 1000);

    return () => {
      window.clearTimeout(initialization);
      window.clearInterval(timer);
    };
  }, []);

  useEffect(() => {
    return () => {
      stopMeter();
      stopTracks();
      if (objectUrlRef.current) {
        URL.revokeObjectURL(objectUrlRef.current);
      }
    };
  }, []);

  return (
    <section className="voice-shell" data-state={assistantState}>
      <div className="hud-time">
        <strong>{now ? formatTime(now) : "--:--"}</strong>
        <span>{now ? formatDate(now) : "SINCRONIZANDO"}</span>
      </div>

      <div className="hud-state">
        <div>
          <span>ONLINE</span>
          <strong>{stateText(assistantState)}</strong>
        </div>
        <div className="hud-mini-wave" aria-hidden="true">
          {levels.slice(0, 18).map((level, index) => (
            <i
              key={index}
              style={{ transform: `scaleY(${0.28 + level * 1.5})` }}
            />
          ))}
        </div>
      </div>

      <div className="hud-side hud-side-left" aria-hidden="true">
        <span>Ferramentas</span>
        <span>Sistema</span>
        <span>Memoria</span>
      </div>

      <div className="hud-side hud-side-right" aria-hidden="true">
        <span className={assistantState === "thinking" ? "is-active" : ""}>
          Pensando
        </span>
        <span className={assistantState === "listening" ? "is-active" : ""}>
          Ouvindo
        </span>
        <span className={assistantState === "speaking" ? "is-active" : ""}>
          Respondendo
        </span>
      </div>

      <div className="voice-stage" data-state={assistantState}>
        <div className="hud-reticle" aria-hidden="true" />
        <div className="voice-rings" aria-hidden="true">
          <span />
          <span />
          <span />
          <span />
        </div>

        <div className="particle-field" aria-hidden="true">
          {particles.map((particle, index) => (
            <span
              key={index}
              style={{
                width: `${particle.size}px`,
                height: `${particle.size}px`,
                transform: `translate(-50%, -50%) rotate(${particle.angle}deg) translateX(${particle.radius * 3}px)`,
                animationDelay: `${particle.delay}s`,
              }}
            />
          ))}
        </div>

        <div className="mesh-core" aria-hidden="true">
          {levels.map((level, index) => (
            <span
              key={index}
              style={{
                transform: `rotate(${(360 / barCount) * index}deg) translateY(-50%) scaleY(${0.35 + level * 1.7})`,
                opacity: 0.25 + level * 0.75,
              }}
            />
          ))}
          <div className="energy-surface" />
          <div className="core-void" />
        </div>

        {(transcript || response || error) && (
          <div className="hud-readout">
            {error ? (
              <p>{error}</p>
            ) : (
              <p>{response || transcript}</p>
            )}
          </div>
        )}
      </div>

      <div className="hud-bottom">
        <button type="button" className="hud-pill" onClick={onOpenText}>
          Modo Texto
        </button>
        <button
          type="button"
          onClick={toggleListening}
          disabled={
            isSupported !== true ||
            assistantState === "thinking" ||
            assistantState === "speaking"
          }
          className="voice-button"
          aria-label={
            assistantState === "listening" ? "Finalizar fala" : "Iniciar escuta"
          }
        >
          <span />
        </button>
        <button type="button" className="hud-pill" onClick={onOpenSystem}>
          Sistema
        </button>
      </div>

      <p className="voice-status">
        {statusText(assistantState, isSupported, voiceStatus)}
      </p>
    </section>
  );
}

function pickMimeType() {
  if (typeof MediaRecorder === "undefined") {
    return "audio/webm";
  }
  if (MediaRecorder.isTypeSupported("audio/webm;codecs=opus")) {
    return "audio/webm;codecs=opus";
  }
  if (MediaRecorder.isTypeSupported("audio/ogg;codecs=opus")) {
    return "audio/ogg;codecs=opus";
  }
  return "audio/webm";
}

function supportsVoiceInput() {
  return (
    typeof navigator !== "undefined" &&
    Boolean(navigator.mediaDevices?.getUserMedia) &&
    typeof MediaRecorder !== "undefined"
  );
}

function statusText(
  state: AssistantState,
  isSupported: boolean | null,
  voiceStatus: VoiceStatus | null,
) {
  if (isSupported === null) {
    return "Inicializando interface de voz.";
  }
  if (isSupported === false) {
    return "Microfone indisponivel neste navegador.";
  }
  if (voiceStatus && (!voiceStatus.sttAvailable || !voiceStatus.ttsAvailable)) {
    return `Configure STT/TTS locais para ativar voz. STT: ${voiceStatus.sttCommand || "?"}, TTS: ${voiceStatus.ttsCommand || "?"}.`;
  }
  if (state === "listening") {
    return "Ouvindo em portugues.";
  }
  if (state === "thinking") {
    return "Processando voz, ferramentas e resposta.";
  }
  if (state === "speaking") {
    return "Respondendo por voz.";
  }
  return "Pronto para escutar.";
}

function stateText(state: AssistantState) {
  if (state === "listening") {
    return "Listening";
  }
  if (state === "thinking") {
    return "Thinking";
  }
  if (state === "speaking") {
    return "Speaking";
  }
  return "Online";
}

function formatTime(date: Date) {
  return new Intl.DateTimeFormat("pt-BR", {
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function formatDate(date: Date) {
  return new Intl.DateTimeFormat("pt-BR", {
    day: "2-digit",
    month: "long",
    year: "numeric",
  })
    .format(date)
    .toUpperCase();
}
