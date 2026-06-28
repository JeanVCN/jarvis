from __future__ import annotations

import asyncio
import logging
from dataclasses import dataclass

from jarvis_voice_agent.config import Settings
from jarvis_voice_agent.livekit import LiveKitTokenIssuer

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


class LiveKitRealtimeAgent:
    def __init__(self, settings: Settings):
        self.settings = settings
        self.stats = RealtimeAgentStats()
        self._running = asyncio.Event()
        self._tasks: set[asyncio.Task[None]] = set()
        self._rtc = require_livekit_dependency()
        self._room = self._rtc.Room()
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
        await self._room.connect(session.url, session.token)
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
            task = asyncio.create_task(self._consume_audio_track(track, participant.identity))
            self._tasks.add(task)
            task.add_done_callback(self._tasks.discard)

        @self._room.on("disconnected")
        def on_disconnected() -> None:
            self._logger.info("realtime agent disconnected")
            self._running.clear()

    async def _consume_audio_track(self, track, participant_identity: str) -> None:
        rtc = self._rtc
        stream = rtc.AudioStream(track)
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
                _ = frame_event.frame
        except asyncio.CancelledError:
            raise
        finally:
            aclose = getattr(stream, "aclose", None)
            if callable(aclose):
                await aclose()
            self._logger.info("audio stream stopped: participant=%s", participant_identity)


async def run_livekit_agent(settings: Settings) -> None:
    agent = LiveKitRealtimeAgent(settings)
    await agent.run_forever()


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    settings = Settings.from_env()
    asyncio.run(run_livekit_agent(settings))


if __name__ == "__main__":
    main()
