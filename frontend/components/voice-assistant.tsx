"use client";

import { useEffect, useRef, useState } from "react";

import {
  audioBase64ToBlob,
  getVoiceStatus,
  sendVoiceConversation,
} from "@/services/voice";
import {
  connectRealtimeVoiceRoom,
  createRealtimeVoiceSession,
} from "@/services/realtime-voice";
import type {
  RealtimeVoiceConnection,
  RealtimeVoiceSession,
} from "@/types/realtime-voice";
import type { AssistantState, VoiceStatus } from "@/types/voice";

const barCount = 28;
const particleCount = 150;
const vadSpeechThreshold = 0.035;
const vadSilenceMs = 900;
const vadSpeechStartMs = 240;
const vadMinSpeechMs = 650;
const vadMaxSpeechMs = 12000;
const idleLevels = Array.from({ length: barCount }, (_, index) =>
  0.18 + (index % 5) * 0.035,
);
const particles = Array.from({ length: particleCount }, (_, index) => {
  const angle = index * 137.508;
  const wave = Math.sin(index * 1.91) * 5 + Math.cos(index * 0.37) * 4;
  const radius = 9 + Math.sqrt(index / particleCount) * 39 + wave;
  const radians = (angle * Math.PI) / 180;
  const left = 50 + Math.cos(radians) * radius * (0.92 + (index % 7) * 0.015);
  const top = 50 + Math.sin(radians) * radius * (0.78 + (index % 9) * 0.018);
  const warm = index % 9 === 0 || index % 19 === 0;
  const violet = index % 4 === 0 || index % 10 === 0;

  return {
    left,
    top,
    size: 3.2 + (index % 7) * 0.9,
    driftX: Math.cos(radians * 1.7) * (5 + (index % 5) * 1.4),
    driftY: Math.sin(radians * 1.3) * (5 + (index % 6) * 1.2),
    delay: -(index % 30) * 0.13,
    duration: 4.8 + (index % 9) * 0.5,
    levelIndex: index % barCount,
    tint: warm ? "#ff365f" : violet ? "#a536ff" : index % 3 === 0 ? "#2377ff" : "#00d8ff",
  };
});

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
  const [realtimeSession, setRealtimeSession] =
    useState<RealtimeVoiceSession | null>(null);
  const [realtimeConnection, setRealtimeConnection] =
    useState<RealtimeVoiceConnection | null>(null);
  const [isRealtimeConnecting, setIsRealtimeConnecting] = useState(false);
  const [now, setNow] = useState<Date | null>(null);

  const recorderRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  const audioContextRef = useRef<AudioContext | null>(null);
  const animationFrameRef = useRef<number | null>(null);
  const vadFrameRef = useRef<number | null>(null);
  const objectUrlRef = useRef<string | null>(null);
  const realtimeConnectionRef = useRef<RealtimeVoiceConnection | null>(null);
  const standbyEnabledRef = useRef(false);
  const speechDetectedRef = useRef(false);
  const speechCandidateStartedAtRef = useRef(0);
  const speechStartedAtRef = useRef(0);
  const silenceStartedAtRef = useRef(0);
  const shouldSubmitRecordingRef = useRef(false);
  const restartingStandbyRef = useRef(false);

  async function toggleListening() {
    if (voiceStatus && (!voiceStatus.sttAvailable || !voiceStatus.ttsAvailable)) {
      setError(
        `Voz local indisponivel. STT: ${voiceStatus.sttCommand || "nao configurado"}, TTS: ${voiceStatus.ttsCommand || "nao configurado"}.`,
      );
      return;
    }

    if (assistantState === "listening") {
      standbyEnabledRef.current = false;
      shouldSubmitRecordingRef.current = false;
      restartingStandbyRef.current = false;
      recorderRef.current?.stop();
      return;
    }

    if (assistantState === "thinking" || assistantState === "speaking") {
      return;
    }

    await startListening();
  }

  async function prepareRealtimeSession() {
    if (isRealtimeConnecting) {
      return;
    }

    if (realtimeConnection) {
      await realtimeConnection.disconnect();
      realtimeConnectionRef.current = null;
      setRealtimeConnection(null);
      setRealtimeSession(null);
      setResponse("Realtime desconectado.");
      return;
    }

    setError("");
    setIsRealtimeConnecting(true);
    try {
      const session = await createRealtimeVoiceSession({
        room: "jarvis-main",
        identity: "desktop",
      });
      const connection = await connectRealtimeVoiceRoom(session);
      realtimeConnectionRef.current = connection;
      setRealtimeSession(session);
      setRealtimeConnection(connection);
      setResponse(`Realtime conectado em ${connection.roomName}.`);
      setTranscript("");
    } catch {
      setRealtimeSession(null);
      realtimeConnectionRef.current = null;
      setRealtimeConnection(null);
      setError("Nao foi possivel conectar o canal realtime.");
    } finally {
      setIsRealtimeConnecting(false);
    }
  }

  async function startListening() {
    setError("");
    setTranscript("");
    setResponse("");
    standbyEnabledRef.current = true;
    restartingStandbyRef.current = false;
    speechDetectedRef.current = false;
    speechCandidateStartedAtRef.current = 0;
    speechStartedAtRef.current = 0;
    silenceStartedAtRef.current = 0;
    shouldSubmitRecordingRef.current = true;

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;
      chunksRef.current = [];

      const recorder = new MediaRecorder(stream, {
        mimeType: pickMimeType(),
      });
      recorderRef.current = recorder;

      recorder.ondataavailable = (event) => {
        if (event.data.size === 0) {
          return;
        }
        chunksRef.current.push(event.data);
      };
      recorder.onstop = () => {
        void submitRecording();
      };

      recorder.start(250);
      setAssistantState("listening");
      startInputMeter(stream);
    } catch {
      setError("Nao foi possivel acessar o microfone.");
      standbyEnabledRef.current = false;
      setAssistantState("idle");
      stopTracks();
    }
  }

  async function submitRecording() {
    stopMeter();
    stopVad();
    stopTracks();

    if (!shouldSubmitRecordingRef.current) {
      setAssistantState("idle");
      setLevels(idleLevels);
      if (restartingStandbyRef.current) {
        restartingStandbyRef.current = false;
        restartStandby();
      }
      return;
    }

    const audio = new Blob(chunksRef.current, { type: pickMimeType() });
    if (audio.size === 0) {
      restartStandby();
      return;
    }

    setAssistantState("thinking");
    setLevels(idleLevels.map((level) => level + 0.12));

    try {
      const result = await sendVoiceConversation(audio, { wakeRequired: true });
      setTranscript(result.transcript);
      if (result.ignored) {
        setResponse("");
        restartStandby();
        return;
      }
      setResponse(result.response);
      await playResponse(result.audioBase64, result.audioContentType);
    } catch {
      setError("Nao foi possivel completar a conversa por voz.");
      restartStandby();
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
    if (standbyEnabledRef.current) {
      restartStandby();
      return;
    }
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
    startVad(analyser);
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

  function startVad(analyser: AnalyserNode) {
    const data = new Uint8Array(analyser.fftSize);

    function frame() {
      analyser.getByteTimeDomainData(data);
      let sum = 0;
      for (const value of data) {
        const centered = (value - 128) / 128;
        sum += centered * centered;
      }
      const rms = Math.sqrt(sum / data.length);
      const nowMs = performance.now();

      if (rms >= vadSpeechThreshold) {
        if (!speechDetectedRef.current) {
          if (speechCandidateStartedAtRef.current === 0) {
            speechCandidateStartedAtRef.current = nowMs;
          }
          if (nowMs - speechCandidateStartedAtRef.current >= vadSpeechStartMs) {
            speechDetectedRef.current = true;
            speechStartedAtRef.current = nowMs;
          }
        }
        silenceStartedAtRef.current = 0;
      } else if (!speechDetectedRef.current) {
        speechCandidateStartedAtRef.current = 0;
      } else {
        if (silenceStartedAtRef.current === 0) {
          silenceStartedAtRef.current = nowMs;
        }
        const speechDuration = nowMs - speechStartedAtRef.current;
        const silenceDuration = nowMs - silenceStartedAtRef.current;
        if (
          speechDuration >= vadMinSpeechMs &&
          (silenceDuration >= vadSilenceMs || speechDuration >= vadMaxSpeechMs)
        ) {
          recorderRef.current?.stop();
          return;
        }
      }

      vadFrameRef.current = window.requestAnimationFrame(frame);
    }

    frame();
  }

  function stopVad() {
    if (vadFrameRef.current) {
      window.cancelAnimationFrame(vadFrameRef.current);
      vadFrameRef.current = null;
    }
  }

  function restartStandby() {
    setAssistantState("idle");
    setLevels(idleLevels);
    if (!standbyEnabledRef.current) {
      return;
    }
    restartingStandbyRef.current = true;
    window.setTimeout(() => {
      if (standbyEnabledRef.current) {
        void startListening();
      } else {
        restartingStandbyRef.current = false;
      }
    }, 350);
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
      stopVad();
      stopTracks();
      if (objectUrlRef.current) {
        URL.revokeObjectURL(objectUrlRef.current);
      }
      if (realtimeConnectionRef.current) {
        void realtimeConnectionRef.current.disconnect();
        realtimeConnectionRef.current = null;
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

      <div className="voice-stage" data-state={assistantState}>
        <div className="particle-field" aria-hidden="true">
          {particles.map((particle, index) => (
            <span
              key={index}
              style={{
                left: `${particle.left}%`,
                top: `${particle.top}%`,
                width: `${particle.size}px`,
                height: `${particle.size}px`,
                backgroundColor: particle.tint,
                opacity: 0.34 + levels[particle.levelIndex] * 0.62,
                transform: `translate(calc(-50% + ${particle.driftX * levels[particle.levelIndex]}px), calc(-50% + ${particle.driftY * levels[particle.levelIndex]}px)) scale(${0.78 + levels[particle.levelIndex] * 1.35})`,
                animationDelay: `${particle.delay}s`,
                animationDuration: `${particle.duration}s`,
              }}
            />
          ))}
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
          className="hud-pill"
          onClick={prepareRealtimeSession}
          disabled={isRealtimeConnecting}
        >
          {realtimeConnection ? "Sair" : realtimeSession ? "Realtime" : "Conectar"}
        </button>
        <div className="voice-control-cluster">
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
              assistantState === "listening" ? "Desarmar escuta" : "Armar escuta"
            }
          >
            <span />
          </button>
        </div>
        <button type="button" className="hud-pill" onClick={onOpenSystem}>
          Configuracoes
        </button>
      </div>

      <p className="voice-status">
        {statusText(
          assistantState,
          isSupported,
          voiceStatus,
          realtimeSession,
          realtimeConnection,
          isRealtimeConnecting,
        )}
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
  realtimeSession: RealtimeVoiceSession | null,
  realtimeConnection: RealtimeVoiceConnection | null,
  isRealtimeConnecting: boolean,
) {
  if (isRealtimeConnecting) {
    return "Conectando realtime...";
  }
  if (realtimeConnection) {
    return `Realtime conectado: ${realtimeConnection.roomName}`;
  }
  if (realtimeSession) {
    return `Realtime preparado: ${realtimeSession.room}`;
  }
  if (isSupported === null) {
    return "Inicializando";
  }
  if (isSupported === false) {
    return "Microfone indisponivel";
  }
  if (voiceStatus && (!voiceStatus.sttAvailable || !voiceStatus.ttsAvailable)) {
    return "Voz local indisponivel";
  }
  if (state === "listening") {
    return "Ouvindo...";
  }
  if (state === "thinking") {
    return "Processando...";
  }
  if (state === "speaking") {
    return "Respondendo...";
  }
  return "Em espera";
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
