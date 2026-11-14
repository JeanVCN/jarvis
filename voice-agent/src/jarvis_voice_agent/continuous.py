"""Private continuous voice lab using Pipecat SmallWebRTC and local CPU engines."""
from __future__ import annotations

import asyncio
import concurrent.futures
import contextlib
import json
import os
import re
import threading
import time
import uuid
from collections import deque
from fractions import Fraction
from dataclasses import asdict

import aiohttp
from aiohttp import web
import av
import numpy as np
from aiortc import MediaStreamTrack
from pipecat.transports.smallwebrtc.connection import SmallWebRTCConnection
from pipecat.audio.vad.silero import SileroVADAnalyzer
from pipecat.audio.vad.vad_analyzer import VADParams, VADState
from pipecat.audio.turn.smart_turn.local_smart_turn_v3 import LocalSmartTurnAnalyzerV3
from pipecat.audio.turn.smart_turn.base_smart_turn import SmartTurnParams
from pipecat.audio.turn.base_turn_analyzer import EndOfTurnState
from .engines import Engines, Profile


def sentences(buffer: str, final: bool = False) -> tuple[list[str], str]:
    """Bounded speech chunks; do not split decimal numbers at their dots."""
    out = []
    while True:
        match = re.search(r"[.!?](?:\s|$)", buffer)
        if match:
            end = match.end()
        elif len(buffer) > 180:
            end = buffer.rfind(" ", 0, 180)
            if end < 1: end = 180
        else: break
        part, buffer = buffer[:end].strip(), buffer[end:]
        if part: out.append(part)
    if final and buffer.strip():
        out.append(buffer.strip())
        buffer = ""
    return out, buffer


class OutputTrack(MediaStreamTrack):
    kind = "audio"

    def __init__(self, notify):
        super().__init__()
        self.queue = asyncio.Queue(maxsize=12)  # <=240 ms PCM; bounded backpressure
        self.generation = 0
        self.notify = notify
        self.pts = 0
        self.started = None
        self.segment = None
        self.segment_samples = 0

    def clear(self):
        self.generation += 1
        while not self.queue.empty():
            self.queue.get_nowait()
            self.queue.task_done()
        self.segment = None

    async def put(self, audio: np.ndarray, rate: int, generation: int, meta: dict):
        # Engine output is mono float32. Resample once per chunk; WebRTC uses 48kHz.
        from scipy.signal import resample_poly
        import math
        gcd = math.gcd(rate, 48000)
        samples = resample_poly(audio, 48000 // gcd, rate // gcd) if rate != 48000 else audio
        pcm = np.clip(samples * 32767, -32768, 32767).astype(np.int16)
        for start in range(0, len(pcm), 960):
            if generation != self.generation: return
            block = np.zeros(960, np.int16)
            chunk = pcm[start:start+960]
            block[:len(chunk)] = chunk
            await self.queue.put((generation, block, meta))

    async def recv(self):
        if self.started is None: self.started = time.monotonic()
        await asyncio.sleep(max(0, self.started + self.pts / 48000 - time.monotonic()))
        block = np.zeros(960, np.int16)
        while not self.queue.empty():
            generation, candidate, meta = self.queue.get_nowait()
            self.queue.task_done()
            if generation == self.generation:
                block = candidate
                key = (meta["turn_id"], meta["segment"])
                if self.segment != key:
                    self.segment = key
                    self.segment_samples = 0
                    self.notify("audio.segment", meta)
                self.segment_samples += 960
                if self.segment_samples % 9600 == 0:
                    self.notify("audio.progress", {**meta, "ms": self.segment_samples // 48})
                break
        frame = av.AudioFrame.from_ndarray(block.reshape(1, -1), format="s16", layout="mono")
        frame.sample_rate = 48000
        frame.pts = self.pts
        frame.time_base = Fraction(1, 48000)
        self.pts += 960
        return frame


class Session:
    def __init__(self, connection, engines, conversation, provider, model, http):
        self.id = uuid.uuid4().hex
        self.client_epoch = 0
        self.connection = connection
        self.engines = engines
        self.conversation = conversation
        self.provider, self.model = provider, model
        self.http = http
        self.core = os.getenv("JARVIS_CORE_URL", "http://127.0.0.1:18080").rstrip("/")
        self.output = OutputTrack(self.event)
        self.tasks = set()
        self.turn_task = None
        self.turn_id = None
        self.stop = threading.Event()
        self.tts_executor = concurrent.futures.ThreadPoolExecutor(max_workers=1, thread_name_prefix="jarvis-tts")
        self.stt_executor = concurrent.futures.ThreadPoolExecutor(max_workers=1, thread_name_prefix="jarvis-stt")
        self.vad = SileroVADAnalyzer(sample_rate=16000, params=VADParams(start_secs=.10, stop_secs=.15, confidence=.65))
        self.vad.set_sample_rate(16000)
        self.smart = LocalSmartTurnAnalyzerV3(sample_rate=16000, cpu_count=1, params=SmartTurnParams(stop_secs=1.5)) if engines.profile.turn == "smart" else None
        if self.smart: self.smart.set_sample_rate(16000)
        self.speaking = False
        self.closed = False

    def spawn(self, coro):
        task = asyncio.create_task(coro)
        self.tasks.add(task)
        task.add_done_callback(self.tasks.discard)
        return task

    def event(self, kind, data):
        self.connection.send_app_message({"version": 1, "session_id": self.id, "client_epoch": self.client_epoch, "type": kind, "data": data, "server_ms": time.monotonic() * 1000})

    async def interrupt(self):
        old = self.turn_id
        self.stop.set()
        self.output.clear()
        if self.turn_task: self.turn_task.cancel()
        self.turn_task = None
        self.turn_id = None
        self.event("turn.interrupted", {"turn_id": old, "cancel_state": "local_discard", "provider_confirmed": False})
        if old:
            self.spawn(self.cancel_core(old))

    async def cancel_core(self, turn):
        with contextlib.suppress(aiohttp.ClientError, asyncio.TimeoutError):
            async with self.http.post(f"{self.core}/api/v2/turns/{turn}/cancel") as r:
                await r.read()

    async def close(self):
        if self.closed: return
        self.closed = True
        await self.interrupt()
        for task in list(self.tasks): task.cancel()
        await asyncio.gather(*list(self.tasks), return_exceptions=True)
        self.output.stop()
        self.tts_executor.shutdown(wait=False, cancel_futures=True)
        self.stt_executor.shutdown(wait=False, cancel_futures=True)
        await self.vad.cleanup()
        if self.smart: await self.smart.cleanup()
        await self.connection.disconnect()

    async def consume(self):
        track = self.connection.audio_input_track()
        if track is None: return
        resampler = av.AudioResampler(format="s16", layout="mono", rate=16000)
        pre = deque(maxlen=15)  # 300 ms before VAD confirms speech
        frames = []
        active = False
        silence = 0
        total = 0
        last_speech_ms = 0
        try:
            while not self.closed:
                raw = await track.recv()
                for frame in resampler.resample(raw):
                    pcm = frame.to_ndarray().tobytes()
                    state = await self.vad.analyze_audio(pcm)
                    speech = state in (VADState.SPEAKING, VADState.STOPPING)
                    if self.smart: self.smart.append_audio(pcm, speech)
                    pre.append(pcm)
                    if speech and not active:
                        await self.interrupt()
                        active = True
                        frames = list(pre)[:-1]
                        total = 0
                        self.event("speech.started", {})
                    if not active: continue
                    frames.append(pcm)
                    duration = len(pcm) / 32
                    total += duration
                    silence = 0 if speech else silence + duration
                    if speech: last_speech_ms = time.monotonic() * 1000
                    finish = silence >= 650 or total >= 15000
                    if self.smart and silence >= 200 and int(silence // 200) != int((silence-duration) // 200):
                        result, _ = await self.smart.analyze_end_of_turn()
                        finish = result == EndOfTurnState.COMPLETE or silence >= 1500 or total >= 15000
                    elif self.smart: finish = silence >= 1500 or total >= 15000
                    if finish:
                        self.event("speech.ended", {"speech_end_server_ms": last_speech_ms, "endpoint_delay_ms": max(0, time.monotonic()*1000-last_speech_ms)})
                        self.turn_task = self.spawn(self.run_turn(b"".join(frames)))
                        active, frames, total, silence = False, [], 0, 0
                        pre.clear()
                        if self.smart: self.smart.clear()
        except (asyncio.CancelledError, av.error.EOFError):
            return
        except Exception:
            if not self.closed: self.event("error", {"code": "audio_input_failed"})

    async def run_turn(self, pcm=None, text=None):
        turn = uuid.uuid4().hex
        self.turn_id = turn
        self.stop = threading.Event()
        stop = self.stop
        generation = self.output.generation
        queue = asyncio.Queue(maxsize=3)
        speech_task = None
        try:
            if text is None:
                begin = time.monotonic()
                text = await asyncio.get_running_loop().run_in_executor(self.stt_executor, self.engines.transcribe, pcm)
                if stop.is_set(): return
                self.event("stt.final", {"turn_id": turn, "text": text, "duration_ms": (time.monotonic()-begin)*1000})
            if not text: return
            payload = {"turn_id": turn, "message": text, "provider": self.provider, "model": self.model}
            # A cancelled previous turn may need a moment to commit terminal state.
            for attempt in range(10):
                async with self.http.post(f"{self.core}/api/v2/conversations/{self.conversation}/turns", json=payload) as r:
                    data = await r.json()
                    if r.status != 409: break
                await asyncio.sleep(.1)
            if r.status not in (200, 202): raise ValueError(data.get("error", "turn_start_failed"))
            speech_task = self.spawn(self.speak(queue, stop, generation, turn))
            buffer = ""
            async with self.http.get(f"{self.core}/api/v2/turns/{turn}/events") as r:
                r.raise_for_status()
                async for line in r.content:
                    if stop.is_set(): return
                    if not line.startswith(b"data: "): continue
                    event = json.loads(line[6:])
                    self.event(event["type"], {**event["data"], "turn_id": turn, "seq": event["seq"]})
                    if event["type"] == "text.delta":
                        parts, buffer = sentences(buffer + event["data"]["delta"])
                        for part in parts: await queue.put(part)
                    if event["type"] == "turn.finished": break
            parts, _ = sentences(buffer, final=True)
            for part in parts: await queue.put(part)
            await queue.put(None)
            await speech_task
            await self.output.queue.join()
            self.event("audio.finished", {"turn_id": turn})
            if self.turn_id == turn: self.turn_id = None
        except asyncio.CancelledError:
            stop.set()
            if speech_task: speech_task.cancel()
            raise
        except Exception as error:
            stop.set()
            self.event("error", {"turn_id": turn, "code": str(error) if isinstance(error, ValueError) else "voice_pipeline_failed"})
            if speech_task: speech_task.cancel()
        finally:
            if stop.is_set(): await self.cancel_core(turn)

    async def speak(self, queue, stop, generation, turn):
        segment = 0
        loop = asyncio.get_running_loop()
        while not stop.is_set():
            text = await queue.get()
            if text is None: break
            started = time.monotonic()
            self.event("tts.started", {"turn_id": turn, "segment": segment, "text": text})
            meta = {"turn_id": turn, "segment": segment, "text": text}

            def work():
                first = True
                for audio, rate in self.engines.synthesize(text, stop):
                    if stop.is_set(): return
                    if first:
                        loop.call_soon_threadsafe(self.event, "tts.first_pcm", {**meta, "duration_ms": (time.monotonic()-started)*1000})
                        first = False
                    future = asyncio.run_coroutine_threadsafe(self.output.put(audio, rate, generation, meta), loop)
                    while True:
                        try: future.result(timeout=.1); break
                        except concurrent.futures.TimeoutError:
                            if stop.is_set(): future.cancel(); return
            await loop.run_in_executor(self.tts_executor, work)
            segment += 1
        self.event("audio.generated", {"turn_id": turn, "segments": segment})


async def create_app():
    app = web.Application(client_max_size=128*1024)
    state = {"session": None, "engines": None, "profile": None, "lock": asyncio.Lock()}
    http = aiohttp.ClientSession(timeout=aiohttp.ClientTimeout(total=200))

    @web.middleware
    async def private_origin(request, handler):
        origin = request.headers.get("Origin")
        allowed = os.getenv("JARVIS_ALLOWED_ORIGINS", "http://127.0.0.1:13000,http://localhost:13000").split(",")
        if request.method != "GET" and origin and origin not in allowed:
            return web.json_response({"error": "origin_denied"}, status=403)
        return await handler(request)
    app.middlewares.append(private_origin)

    async def offer(request):
        async with state["lock"]:
            data = await request.json()
            try:
                profile = Profile(**data.get("profile", {}))
                profile.validate()
            except (ValueError, TypeError) as error: return web.json_response({"error": str(error)}, status=400)
            epoch = data.get("client_epoch", 0)
            if not isinstance(epoch, int) or not 0 <= epoch <= 2**31-1:
                return web.json_response({"error": "invalid_client_epoch"}, status=400)
            old = state["session"]
            if old and not old.closed:
                if data.get("resume_session") != old.id:
                    return web.json_response({"error": "session_in_use"}, status=409)
                await old.close()
                await asyncio.get_running_loop().run_in_executor(None, lambda: (old.stt_executor.shutdown(wait=True), old.tts_executor.shutdown(wait=True)))
                old.engines = None
            if old and old.closed:
                await asyncio.get_running_loop().run_in_executor(None, lambda: (old.stt_executor.shutdown(wait=True), old.tts_executor.shutdown(wait=True)))
                old.engines = None
            # Loading happens only after previous inference stops; no concurrent model instances.
            if state["profile"] != profile:
                state["engines"] = None
                import gc
                gc.collect()
                try: state["engines"] = await asyncio.to_thread(Engines, profile)
                except Exception as error:
                    return web.json_response({"error": "model_load_failed", "detail": type(error).__name__}, status=503)
                state["profile"] = profile
            conversation = data.get("conversation_id")
            if not conversation:
                async with http.post(os.getenv("JARVIS_CORE_URL", "http://127.0.0.1:18080")+"/api/v2/conversations") as r:
                    if r.status != 201: return web.json_response({"error": "core_unavailable"}, status=503)
                    conversation = (await r.json())["conversation_id"]
            if not re.fullmatch(r"[a-f0-9]{32}", conversation): return web.json_response({"error": "invalid_conversation"}, status=400)
            connection = SmallWebRTCConnection(ice_servers=[], connection_timeout_secs=30)
            session = Session(connection, state["engines"], conversation, data.get("provider", "scripted"), data.get("model", ""), http)
            session.client_epoch = epoch
            state["session"] = session

            @connection.event_handler("connected")
            async def connected(_):
                session.event("session.ready", {"session_id": session.id, "conversation_id": conversation, "profile": asdict(profile), "stt_partial": False})
                session.spawn(session.consume())

            @connection.event_handler("app-message")
            async def message(_, value):
                kind = value.get("type")
                if kind in ("interrupt", "text"):
                    epoch = value.get("client_epoch", session.client_epoch)
                    if not isinstance(epoch, int) or epoch < session.client_epoch: return
                    session.client_epoch = min(2**31-1, epoch)
                if kind == "interrupt": await session.interrupt()
                elif kind == "text" and isinstance(value.get("text"), str) and len(value["text"]) <= 8000:
                    await session.interrupt()
                    session.turn_task = session.spawn(session.run_turn(text=value["text"]))
                elif kind == "playback" and re.fullmatch(r"[a-f0-9]{32}", value.get("turn_id", "")):
                    async with http.post(f"{session.core}/api/v2/turns/{value['turn_id']}/playback", json={"segment": value.get("segment", 0), "ms": value.get("ms", 0), "finished": value.get("finished", False)}) as r:
                        await r.read()
                        if r.status == 204 and value.get("finished"):
                            session.event("playback.saved", {"turn_id": value["turn_id"]})

            @connection.event_handler("failed")
            async def failed(_):
                if not session.closed: await session.close()

            @connection.event_handler("closed")
            async def closed(_):
                if not session.closed: await session.close()
            try:
                await connection.initialize(data["sdp"], data["type"])
                connection.replace_audio_track(session.output)
                await connection.connect()
            except Exception:
                await session.close()
                return web.json_response({"error": "webrtc_offer_failed"}, status=400)
            answer = connection.get_answer()
            address = os.getenv("JARVIS_ICE_ADDRESS")
            if address:
                lines = answer["sdp"].split("\r\n")
                allowed = [line for line in lines if line.startswith("a=candidate:") and len(line.split()) > 4 and line.split()[4] == address]
                if not allowed:
                    await session.close()
                    return web.json_response({"error": "ice_address_unavailable"}, status=503)
                answer["sdp"] = "\r\n".join(line for line in lines if not line.startswith("a=candidate:") or line in allowed)
            return web.json_response({**answer, "session_id": session.id, "conversation_id": conversation, "profile": asdict(profile)})

    async def health(request):
        return web.json_response({"status": "ok", "transport": "smallwebrtc", "profile": asdict(state["profile"]) if state["profile"] else None, "synthetic_enabled": os.getenv("JARVIS_TEST_AUDIO") == "true"})
    async def cleanup(_):
        if state["session"]: await state["session"].close()
        await http.close()
    app.router.add_post("/offer", offer)
    app.router.add_get("/health", health)
    app.on_cleanup.append(cleanup)
    return app


def main():
    web.run_app(create_app(), host=os.getenv("JARVIS_VOICE_AGENT_HOST", "127.0.0.1"), port=int(os.getenv("JARVIS_VOICE_AGENT_PORT", "18100")))


if __name__ == "__main__": main()
