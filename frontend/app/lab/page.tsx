"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";

type LabEvent = { client_epoch?: number; session_id?: string; type: string; data: Record<string, unknown>; server_ms?: number };
type Metric = { at: number; type: string; data: Record<string, unknown> };
const id = () => crypto.randomUUID().replaceAll("-", "");

export default function VoiceLab() {
  const [stt, setSTT] = useState("base");
  const [tts, setTTS] = useState("kokoro");
  const [voice, setVoice] = useState("pf_dora");
  const [turn, setTurn] = useState("silence");
  const [provider, setProvider] = useState("scripted");
  const [model, setModel] = useState("");
  const [status, setStatus] = useState("Desconectado");
  const [connected, setConnected] = useState(false);
  const [text, setText] = useState("");
  const [transcript, setTranscript] = useState("");
  const [answer, setAnswer] = useState("");
  const [history, setHistory] = useState<Array<Record<string, unknown>>>([]);
  const [events, setEvents] = useState<Metric[]>([]);
  const state = useRef<{
    pc?: RTCPeerConnection; dc?: RTCDataChannel; mic?: MediaStream; ctx?: AudioContext;
    gain?: GainNode; raf?: number; ping?: ReturnType<typeof setInterval>;
    session?: string; conversation?: string; turn?: string; segment?: number;
    epoch?: number; blocked?: Set<string>; segmentAt?: number; sentMS?: number; firstPCM?: boolean; stopped?: boolean; attempt?: number;
  }>({});
  const log = (type: string, data: Record<string, unknown> = {}) => {
    setEvents(old => [...old.slice(-1999), { at: performance.now(), type, data }]);
  };
  const send = (data: Record<string, unknown>) => {
    if (state.current.dc?.readyState === "open") state.current.dc.send(JSON.stringify({ ...data, client_epoch: state.current.epoch ?? 0 }));
  };
  const ack = () => {
    const s = state.current;
    if (s.turn && s.segment !== undefined && s.segmentAt !== undefined)
      send({ type: "playback", turn_id: s.turn, segment: s.segment, ms: Math.max(0, Math.min(s.sentMS ?? 0, Math.floor(performance.now() - s.segmentAt))) });
  };
  const interrupt = () => {
    ack();
    const s = state.current;
    if (s.gain) s.gain.gain.value = 0;
    s.firstPCM = false;
    s.epoch = (s.epoch ?? 0) + 1;
    s.blocked ??= new Set(); if (s.turn) s.blocked.add(s.turn);
    send({ type: "interrupt" });
    s.turn = undefined; s.segment = undefined; s.segmentAt = undefined; s.sentMS = 0;
    log("client.interrupt", { turn_id: s.turn });
  };
  const teardown = () => {
    const s = state.current;
    if (s.raf) cancelAnimationFrame(s.raf);
    if (s.ping) clearInterval(s.ping);
    s.pc?.close(); s.mic?.getTracks().forEach(t => t.stop());
    void s.ctx?.close();
    s.pc = undefined; s.ctx = undefined; s.dc = undefined;
    s.segment = undefined; s.segmentAt = undefined;
  };
  useEffect(() => {
    void fetch("/api/lab-profile").then(r=>r.json()).then(p=>{
      if(p.stt)setSTT(p.stt);if(p.tts)setTTS(p.tts);if(p.voice)setVoice(p.voice);
      if(p.turn)setTurn(p.turn);if(p.provider)setProvider(p.provider);if(p.model)setModel(p.model);
    }).catch(()=>{});
  }, []);
  useEffect(() => () => {
    state.current.stopped = true;
    const s = state.current;
    if (s.raf) cancelAnimationFrame(s.raf);
    if (s.ping) clearInterval(s.ping);
    s.pc?.close(); s.mic?.getTracks().forEach(t => t.stop()); void s.ctx?.close();
  }, []);

  async function loadHistory() {
    const conversation = state.current.conversation;
    if (!conversation) return;
    const r = await fetch(`/api/v2/conversations/${conversation}`);
    if (r.ok) setHistory((await r.json()).turns);
  }

  async function connect(reconnect = false) {
    if (!reconnect) state.current.stopped = false;
    teardown(); setConnected(false); setStatus("Preparando modelos e conexão…");
    try {
      const s = state.current;
      s.conversation ??= localStorage.getItem("jarvis.lab.conversation") ?? undefined;
      s.session ??= sessionStorage.getItem("jarvis.lab.session") ?? undefined;
      s.ctx = new AudioContext(); await s.ctx.resume();
      s.gain = s.ctx.createGain(); s.gain.gain.value = 0; s.gain.connect(s.ctx.destination);
      s.mic = await navigator.mediaDevices.getUserMedia({ audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true }, video: false });
      const input = s.ctx.createAnalyser(); input.fftSize = 512;
      s.ctx.createMediaStreamSource(s.mic).connect(input);
      const output = s.ctx.createAnalyser(); output.fftSize = 512;
      output.connect(s.gain);
      let candidate = 0, localSpeaking = false, quiet = 0;
      const inputData = new Float32Array(512), outputData = new Float32Array(512);
      const meter = () => {
        input.getFloatTimeDomainData(inputData); output.getFloatTimeDomainData(outputData);
        const rms = Math.sqrt(inputData.reduce((sum, v) => sum + v*v, 0) / inputData.length);
        const now = performance.now();
        if (rms > .035) {
          quiet = 0; candidate ||= now;
          if (!localSpeaking && now-candidate >= 100) { localSpeaking = true; interrupt(); log("client.speech_started"); }
        } else {
          candidate = 0; quiet ||= now;
          if (now-quiet > 250 && localSpeaking) { localSpeaking = false; log("client.speech_ended", { estimated_end_at: quiet }); }
        }
        const outputRMS = Math.sqrt(outputData.reduce((sum,v) => sum+v*v,0)/outputData.length);
        if (s.gain && s.gain.gain.value > 0 && outputRMS > .002 && !s.firstPCM) {
          s.firstPCM = true; s.segmentAt = now; log("client.first_audio", { turn_id: s.turn, segment: s.segment, estimated: true }); ack();
        }
        s.raf = requestAnimationFrame(meter);
      };
      const pc = new RTCPeerConnection({ iceServers: [] }); s.pc = pc;
      s.mic.getTracks().forEach(track => pc.addTrack(track, s.mic!));
      pc.ontrack = event => {
        const stream = new MediaStream([event.track]);
        s.ctx!.createMediaStreamSource(stream).connect(output);
      };
      const dc = pc.createDataChannel("jarvis"); s.dc = dc;
      dc.onopen = () => {
        s.attempt = 0; setConnected(true); setStatus("Conectado — microfone contínuo");
        s.ping = setInterval(() => { if (dc.readyState === "open") { dc.send("ping"); ack(); } }, 1000);
      };
      dc.onmessage = event => {
        let v: LabEvent;
        try { v = JSON.parse(event.data); } catch { return; }
        log(v.type, { ...v.data, server_ms: v.server_ms, client_epoch: v.client_epoch });
        if ((v.client_epoch ?? 0) < (s.epoch ?? 0)) return;
        if (v.session_id && s.session && v.session_id !== s.session) return;
        if (typeof v.data.turn_id === "string" && s.blocked?.has(v.data.turn_id)) return;
        if (v.type === "turn.interrupted" && typeof v.data.turn_id === "string") {s.blocked ??= new Set();s.blocked.add(v.data.turn_id);}
        if (v.type === "stt.final") setTranscript(String(v.data.text));
        if (v.type === "turn.started") { s.turn = String(v.data.turn_id); setAnswer(""); s.firstPCM = false; }
        if (v.type === "text.delta" && v.data.turn_id === s.turn) setAnswer(old => old + String(v.data.delta));
        if (v.type === "speech.started" || v.type === "turn.interrupted") { if (s.gain) s.gain.gain.value = 0; }
        if (v.type === "audio.segment" && v.data.turn_id === s.turn) {
          ack(); s.segment = Number(v.data.segment); s.sentMS = 0; s.segmentAt = performance.now();
          if (s.gain && !localSpeaking) s.gain.gain.value = 1;
        }
        if (v.type === "audio.progress" && v.data.turn_id === s.turn && Number(v.data.segment) === s.segment) s.sentMS = Number(v.data.ms);
        if (v.type === "audio.finished" && v.data.turn_id === s.turn) {
          ack();send({type:"playback",turn_id:s.turn,segment:s.segment??0,ms:s.sentMS??0,finished:true});
        }
        if (v.type === "playback.saved") void loadHistory();
        if (v.type === "error") setStatus(`Erro: ${v.data.code}`);
        if (v.type === "turn.finished") void loadHistory();
      };
      pc.onconnectionstatechange = () => {
        if (["failed", "disconnected"].includes(pc.connectionState) && !s.stopped) {
          setConnected(false); setStatus("Conexão perdida. Retomando sem repetir o turno…");
          const attempt = (s.attempt ?? 0)+1; s.attempt = attempt;
          if (attempt <= 3) setTimeout(() => { if (!s.stopped && s.pc === pc) void connect(true); }, attempt*1500);
          else setStatus("Reconexão esgotada. Clique em conectar.");
        }
      };
      await pc.setLocalDescription(await pc.createOffer());
      if (pc.iceGatheringState !== "complete") await new Promise<void>((resolve, reject) => {
        const timer = setTimeout(() => reject(new Error("ice_gathering_timeout")), 10000);
        pc.addEventListener("icegatheringstatechange", () => { if (pc.iceGatheringState === "complete") { clearTimeout(timer); resolve(); } });
      });
      const r = await fetch("/voice/offer", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ sdp: pc.localDescription?.sdp, type: "offer", profile: { stt, tts, voice, turn }, provider, model, conversation_id: s.conversation, resume_session: s.session, client_epoch: s.epoch ?? 0 }) });
      const data = await r.json(); if (!r.ok) throw new Error(`${data.error}${data.detail ? ` (${data.detail})` : ""}`);
      s.session = data.session_id; s.conversation = data.conversation_id;
      localStorage.setItem("jarvis.lab.conversation", s.conversation!); sessionStorage.setItem("jarvis.lab.session", s.session!);
      await pc.setRemoteDescription({ sdp: data.sdp, type: data.type });
      setStatus("Sinalização pronta; aguardando mídia WebRTC…");
      setTimeout(()=>{if(!s.stopped && s.pc===pc && pc.connectionState!=="connected")setStatus("Mídia não conectou: verifique ICE, firewall e política de UDP do navegador.");},15000);
      log("session.profile", { stt, tts, voice, turn, provider, model });
      void loadHistory();
      meter();
      setTimeout(() => { void pc.getStats().then(stats => stats.forEach(report => { if (report.type === "candidate-pair" && report.state === "succeeded") log("ice.selected_pair", { localCandidateId: report.localCandidateId, remoteCandidateId: report.remoteCandidateId, currentRoundTripTime: report.currentRoundTripTime }); })); }, 3000);
    } catch (error) { teardown(); setStatus(error instanceof Error ? error.message : "Falha na conexão"); }
  }

  const exportResults = () => {
    const blob = new Blob([JSON.stringify({ version: 1, profile: { stt, tts, voice, turn, provider, model }, browser: navigator.userAgent, events }, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = url; a.download = `jarvis-${id()}.json`; a.click(); URL.revokeObjectURL(url);
  };
  const field = "rounded border border-slate-600 bg-slate-900 p-2";
  return <main className="min-h-screen bg-slate-950 p-6 text-slate-100">
    <div className="mx-auto flex max-w-4xl flex-col gap-5">
      <Link href="/" className="text-cyan-300">← Jarvis atual</Link>
      <h1 className="text-2xl font-semibold">Laboratório de voz contínua</h1>
      <p>Compare uma variável por vez. Os modelos carregam ao conectar; somente uma sessão é permitida. Use fones no primeiro teste.</p>
      <fieldset disabled={connected} className="grid gap-3 sm:grid-cols-3">
        <label>Reconhecimento<select aria-label="Reconhecimento" className={`${field} block w-full`} value={stt} onChange={e => setSTT(e.target.value)}>{["base","small","fixture"].map(v=><option key={v}>{v}</option>)}</select></label>
        <label>Síntese<select aria-label="Síntese" className={`${field} block w-full`} value={tts} onChange={e => {setTTS(e.target.value);setVoice(e.target.value === "pocket" ? "rafael" : "pf_dora");}}>{["kokoro","pocket","tone"].map(v=><option key={v}>{v}</option>)}</select></label>
        <label>Voz<select aria-label="Voz" className={`${field} block w-full`} value={voice} onChange={e => setVoice(e.target.value)}>{(tts === "pocket" ? ["rafael"] : ["pf_dora","pm_alex","pm_santa"]).map(v=><option key={v}>{v}</option>)}</select></label>
        <label>Fim de turno<select aria-label="Fim de turno" className={`${field} block w-full`} value={turn} onChange={e=>setTurn(e.target.value)}><option value="silence">Silêncio fixo</option><option value="smart">Smart Turn v3.2</option></select></label>
        <label>Motor de texto<select aria-label="Motor de texto" className={`${field} block w-full`} value={provider} onChange={e=>setProvider(e.target.value)}>{["scripted","groq","codex","ollama"].map(v=><option key={v}>{v}</option>)}</select></label>
        <label>Modelo<input aria-label="Modelo" className={`${field} block w-full`} value={model} onChange={e=>setModel(e.target.value)} placeholder="ID elegível ou padrão" /></label>
      </fieldset>
      <p role="status">{status}</p>
      <div className="flex flex-wrap gap-3">
        <button className={field} disabled={connected} onClick={()=>void connect()}>Conectar</button>
        <button className={field} disabled={!connected} onClick={interrupt}>Interromper</button>
        <button className={field} onClick={()=>{state.current.stopped=true; interrupt();teardown();setConnected(false);setStatus("Desconectado");}}>Desconectar</button>
        <button className={field} onClick={exportResults}>Exportar medições</button>
        <button className={field} disabled={connected} onClick={()=>{state.current.conversation=undefined;localStorage.removeItem("jarvis.lab.conversation");setHistory([]);}}>Nova conversa</button>
      </div>
      <form className="flex gap-3" onSubmit={e=>{e.preventDefault();interrupt();send({type:"text",text});setTranscript(text);setText("");}}>
        <input aria-label="Mensagem de teste" className={`${field} flex-1`} value={text} onChange={e=>setText(e.target.value)} placeholder="Texto para testar sem STT" />
        <button className={field} disabled={!connected || !text.trim()}>Enviar</button>
      </form>
      <p><strong>Transcrição:</strong> {transcript}</p>
      <p className="whitespace-pre-wrap"><strong>Resposta gerada:</strong> {answer}</p>
      <details><summary>Histórico persistido ({history.length} turnos)</summary>{[...history].reverse().map(v=><p key={String(v.turn_id)} className="my-3 border-b border-slate-700 pb-3">{String(v.input)}<br/>{String(v.generated)}<br/><small>{String(v.status)} / áudio: {String(v.delivery_status)} · segmento reproduzido estimado: {String(v.played_segment)}</small></p>)}</details>
      <details><summary>Eventos e métricas ({events.length})</summary><pre className="max-h-96 overflow-auto text-xs">{JSON.stringify(events.slice(-40),null,2)}</pre></details>
      <p className="text-sm text-slate-400">fixture/tone são recursos sintéticos de teste e exigem habilitação no servidor. Codex exige ativação e login; Groq exige conta Free confirmada. Nenhum provedor pago é usado como fallback.</p>
    </div>
  </main>;
}
