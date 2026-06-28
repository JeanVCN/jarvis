from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    core_url: str
    host: str
    port: int
    request_timeout_seconds: float
    language: str
    transport: str
    livekit_url: str
    livekit_api_key: str
    livekit_api_secret: str
    livekit_room: str
    livekit_agent_identity: str
    livekit_connect_timeout_seconds: float
    livekit_connect_retries: int
    livekit_connect_retry_delay_seconds: float
    livekit_room_prefix: str
    livekit_token_ttl_seconds: int
    cors_allowed_origins: tuple[str, ...]
    realtime_stt_url: str
    realtime_stt_language: str
    realtime_wake_required: bool
    realtime_wake_words: tuple[str, ...]
    realtime_vad_threshold: float
    realtime_silence_ms: int
    realtime_min_speech_ms: int
    realtime_max_speech_ms: int

    @classmethod
    def from_env(cls) -> "Settings":
        return cls(
            core_url=os.environ.get("JARVIS_CORE_URL", "http://127.0.0.1:18080").rstrip("/"),
            host=os.environ.get("JARVIS_VOICE_AGENT_HOST", "127.0.0.1"),
            port=int(os.environ.get("JARVIS_VOICE_AGENT_PORT", "18100")),
            request_timeout_seconds=float(os.environ.get("JARVIS_CORE_TIMEOUT_SECONDS", "300")),
            language=os.environ.get("JARVIS_VOICE_LANGUAGE", "pt-BR"),
            transport=os.environ.get("JARVIS_VOICE_TRANSPORT", "console"),
            livekit_url=os.environ.get("JARVIS_LIVEKIT_URL", "ws://127.0.0.1:7880").rstrip("/"),
            livekit_api_key=os.environ.get("JARVIS_LIVEKIT_API_KEY", "devkey"),
            livekit_api_secret=os.environ.get("JARVIS_LIVEKIT_API_SECRET", "secret"),
            livekit_room=os.environ.get("JARVIS_LIVEKIT_ROOM", "jarvis-main"),
            livekit_agent_identity=os.environ.get("JARVIS_LIVEKIT_AGENT_IDENTITY", "jarvis-agent"),
            livekit_connect_timeout_seconds=float(os.environ.get("JARVIS_LIVEKIT_CONNECT_TIMEOUT_SECONDS", "10")),
            livekit_connect_retries=int(os.environ.get("JARVIS_LIVEKIT_CONNECT_RETRIES", "12")),
            livekit_connect_retry_delay_seconds=float(os.environ.get("JARVIS_LIVEKIT_CONNECT_RETRY_DELAY_SECONDS", "2")),
            livekit_room_prefix=os.environ.get("JARVIS_LIVEKIT_ROOM_PREFIX", "jarvis"),
            livekit_token_ttl_seconds=int(os.environ.get("JARVIS_LIVEKIT_TOKEN_TTL_SECONDS", "3600")),
            cors_allowed_origins=parse_csv(
                os.environ.get(
                    "JARVIS_VOICE_AGENT_CORS_ALLOWED_ORIGIN",
                    "http://127.0.0.1:13000,http://localhost:13000",
                )
            ),
            realtime_stt_url=os.environ.get("JARVIS_REALTIME_STT_URL", "http://127.0.0.1:18081/transcribe").rstrip("/"),
            realtime_stt_language=os.environ.get("JARVIS_REALTIME_STT_LANGUAGE", "pt"),
            realtime_wake_required=parse_bool(os.environ.get("JARVIS_REALTIME_WAKE_REQUIRED", "true")),
            realtime_wake_words=parse_csv(os.environ.get("JARVIS_REALTIME_WAKE_WORDS", "jarvis")),
            realtime_vad_threshold=float(os.environ.get("JARVIS_REALTIME_VAD_THRESHOLD", "0.025")),
            realtime_silence_ms=int(os.environ.get("JARVIS_REALTIME_SILENCE_MS", "900")),
            realtime_min_speech_ms=int(os.environ.get("JARVIS_REALTIME_MIN_SPEECH_MS", "650")),
            realtime_max_speech_ms=int(os.environ.get("JARVIS_REALTIME_MAX_SPEECH_MS", "12000")),
        )


def parse_csv(value: str) -> tuple[str, ...]:
    items = tuple(item.strip() for item in value.split(",") if item.strip())
    return items or ("http://127.0.0.1:13000",)


def parse_bool(value: str) -> bool:
    return value.strip().lower() in ("1", "true", "yes", "y", "on")
