from __future__ import annotations

import asyncio
import json
import logging
import sys
from dataclasses import dataclass

from jarvis_voice_agent.audio_turn import AudioTurnDetector
from jarvis_voice_agent.config import Settings
from jarvis_voice_agent.jarvis_core import JarvisCoreClient, JarvisCoreError
from jarvis_voice_agent.livekit import LiveKitTokenIssuer
from jarvis_voice_agent.stt import STTClient, STTError
from jarvis_voice_agent.wake import strip_wake_word

class RealtimeTransportUnavailable(RuntimeError):
    pass


def require_realtime_dependencies() -> None:
    missing: list[str] = []
    for module_name in ("pipecat", "livekit"):
        try:
            __import__(module_name)
        except ImportError:
            missing.append(module_name)

    if missing:
        joined = ", ".join(missing)
        raise RealtimeTransportUnavailable(
            "Realtime transport dependencies are not installed. "
            "Install them with: python -m pip install -r requirements-realtime.txt. "
            f"Missing: {joined}."
        )


def require_livekit_dependency():
    try:
        from livekit import rtc
    except ImportError as error:
        raise RealtimeTransportUnavailable(
            "LiveKit Python SDK is not installed. "
            "Install realtime dependencies with: python -m pip install -r requirements-realtime.txt."
        ) from error
    return rtc


@dataclass
class RealtimeAgentStats:
    audio_tracks: int = 0
    audio_frames: int = 0
    audio_turns: int = 0
    transcripts: int = 0
    ignored_without_wake_word: int = 0
    responses_sent: int = 0


class LiveKitRealtimeAgent:
    def __init__(self, settings: Settings):
        self.settings = settings
        self.stats = RealtimeAgentStats()
        self._running = asyncio.Event()
        self._tasks: set[asyncio.Task[None]] = set()
        self._rtc = require_livekit_dependency()
        self._room = self._rtc.Room()
        self._core = JarvisCoreClient(settings.core_url, settings.request_timeout_seconds)
        self._stt = STTClient(settings.realtime_stt_url, settings.realtime_stt_language)
        self._logger = logging.getLogger("jarvis_voice_agent.realtime")

    async def run_forever(self) -> None:
        self._running.set()
        self._register_handlers()
        token_issuer = LiveKitTokenIssuer(
            url=self.settings.livekit_url,
            api_key=self.settings.livekit_api_key,
            api_secret=self.settings.livekit_api_secret,
            room_prefix=self.settings.livekit_room_prefix,
            token_ttl_seconds=self.settings.livekit_token_ttl_seconds,
        )
        session = token_issuer.create_session(
            room=self.settings.livekit_room,
            identity=self.settings.livekit_agent_identity,
        )

        self._logger.info(
            "connecting realtime agent: url=%s room=%s identity=%s",
            session.url,
            session.room,
            session.identity,
        )
        await self._connect_with_retry(session.url, session.token)
        self._logger.info("realtime agent connected: room=%s", self._room.name)

        try:
            while self._running.is_set():
                await asyncio.sleep(1)
        finally:
            await self.stop()

    async def stop(self) -> None:
        self._running.clear()
        for task in list(self._tasks):
            task.cancel()
        if self._tasks:
            await asyncio.gather(*self._tasks, return_exceptions=True)
            self._tasks.clear()
        await self._room.disconnect()

    async def _connect_with_retry(self, url: str, token: str) -> None:
        attempts = max(1, self.settings.livekit_connect_retries)
        last_error: BaseException | None = None
        for attempt in range(1, attempts + 1):
            try:
                await asyncio.wait_for(
                    self._room.connect(
                        url,
                        token,
                        self._rtc.RoomOptions(
                            auto_subscribe=True,
                            connect_timeout=self.settings.livekit_connect_timeout_seconds,
                        ),
                    ),
                    timeout=self.settings.livekit_connect_timeout_seconds + 2,
                )
                return
            except Exception as error:
                last_error = error
                if attempt >= attempts:
                    break
                self._logger.warning(
                    "livekit connect attempt failed: attempt=%d/%d error=%s",
                    attempt,
                    attempts,
                    error,
                )
                await asyncio.sleep(self.settings.livekit_connect_retry_delay_seconds)
        raise TimeoutError("LiveKit connection attempts exhausted") from last_error

    def _register_handlers(self) -> None:
        rtc = self._rtc

        @self._room.on("participant_connected")
        def on_participant_connected(participant) -> None:
            self._logger.info("participant connected: %s", participant.identity)

        @self._room.on("participant_disconnected")
        def on_participant_disconnected(participant) -> None:
            self._logger.info("participant disconnected: %s", participant.identity)

        @self._room.on("track_subscribed")
        def on_track_subscribed(track, publication, participant) -> None:
            self._logger.info(
                "track subscribed: kind=%s participant=%s publication=%s",
                track.kind,
                participant.identity,
                publication.sid,
            )
            if track.kind != rtc.TrackKind.KIND_AUDIO:
                return
            self.stats.audio_tracks += 1
            self._spawn_task(self._consume_audio_track(track, participant.identity))

        @self._room.on("disconnected")
        def on_disconnected() -> None:
            self._logger.info("realtime agent disconnected")
            self._running.clear()

    async def _consume_audio_track(self, track, participant_identity: str) -> None:
        rtc = self._rtc
        stream = rtc.AudioStream(track)
        turn_detector = AudioTurnDetector(
            threshold=self.settings.realtime_vad_threshold,
            silence_ms=self.settings.realtime_silence_ms,
            min_speech_ms=self.settings.realtime_min_speech_ms,
            max_speech_ms=self.settings.realtime_max_speech_ms,
        )
        self._logger.info("audio stream started: participant=%s", participant_identity)
        try:
            async for frame_event in stream:
                self.stats.audio_frames += 1
                if self.stats.audio_frames % 100 == 0:
                    self._logger.info(
                        "audio frames received: participant=%s total=%d",
                        participant_identity,
                        self.stats.audio_frames,
                    )
                turn = turn_detector.push(frame_event.frame)
                if turn is not None:
                    self.stats.audio_turns += 1
                    self._logger.info(
                        "audio turn detected: participant=%s duration_ms=%d sample_rate=%d channels=%d",
                        participant_identity,
                        turn.duration_ms,
                        turn.sample_rate,
                        turn.num_channels,
                    )
                    self._spawn_task(self._handle_audio_turn(turn.wav, participant_identity))
        except asyncio.CancelledError:
            raise
        finally:
            aclose = getattr(stream, "aclose", None)
            if callable(aclose):
                await aclose()
            self._logger.info("audio stream stopped: participant=%s", participant_identity)

    async def _handle_audio_turn(self, wav: bytes, participant_identity: str) -> None:
        try:
            transcript = await asyncio.to_thread(self._stt.transcribe_wav, wav)
        except STTError as error:
            self._logger.warning("stt failed: participant=%s error=%s", participant_identity, error)
            await self._publish_voice_event(
                participant_identity,
                {"type": "error", "message": "stt_failed", "detail": str(error)},
            )
            return

        if not transcript:
            self._logger.info("empty transcript ignored: participant=%s", participant_identity)
            await self._publish_voice_event(
                participant_identity,
                {"type": "ignored", "reason": "empty_transcript"},
            )
            return

        self.stats.transcripts += 1
        command, wake_detected = strip_wake_word(transcript, self.settings.realtime_wake_words)
        await self._publish_voice_event(
            participant_identity,
            {
                "type": "transcript",
                "transcript": transcript,
                "command": command,
                "wakeDetected": wake_detected,
            },
        )
        self._logger.info(
            "transcript received: participant=%s wake=%s transcript=%r command=%r",
            participant_identity,
            wake_detected,
            transcript,
            command,
        )

        if self.settings.realtime_wake_required and not wake_detected:
            self.stats.ignored_without_wake_word += 1
            self._logger.info("transcript ignored without wake word: participant=%s", participant_identity)
            await self._publish_voice_event(
                participant_identity,
                {
                    "type": "ignored",
                    "reason": "wake_word_required",
                    "transcript": transcript,
                },
            )
            return

        try:
            response = await asyncio.to_thread(self._core.chat, command)
        except JarvisCoreError as error:
            self._logger.warning("jarvis core failed: participant=%s error=%s", participant_identity, error)
            await self._publish_voice_event(
                participant_identity,
                {"type": "error", "message": "jarvis_core_failed", "detail": str(error)},
            )
            return

        self.stats.responses_sent += 1
        await self._publish_voice_event(
            participant_identity,
            {
                "type": "response",
                "transcript": transcript,
                "command": command,
                "response": response,
            },
        )
        self._logger.info(
            "jarvis response: participant=%s response=%r",
            participant_identity,
            response,
        )

    async def _publish_voice_event(self, participant_identity: str, payload: dict[str, object]) -> None:
        message = json.dumps(payload, ensure_ascii=False)
        try:
            await self._room.local_participant.publish_data(
                message,
                reliable=True,
                destination_identities=[participant_identity],
                topic="jarvis.voice",
            )
        except Exception as error:
            self._logger.warning(
                "failed to publish voice event: participant=%s error=%s payload=%s",
                participant_identity,
                error,
                message,
            )

    def _spawn_task(self, coroutine) -> None:
        task = asyncio.create_task(coroutine)
        self._tasks.add(task)
        task.add_done_callback(self._tasks.discard)


async def run_livekit_agent(settings: Settings) -> None:
    agent = LiveKitRealtimeAgent(settings)
    await agent.run_forever()


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    settings = Settings.from_env()
    try:
        asyncio.run(run_livekit_agent(settings))
    except RealtimeTransportUnavailable as error:
        print(error, file=sys.stderr)
        raise SystemExit(1) from error
    except TimeoutError as error:
        print(
            "Timed out connecting to LiveKit. Check JARVIS_LIVEKIT_URL and whether the LiveKit server is running.",
            file=sys.stderr,
        )
        raise SystemExit(1) from error


if __name__ == "__main__":
    main()
