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
    livekit_room_prefix: str
    livekit_token_ttl_seconds: int
    cors_allowed_origins: tuple[str, ...]

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
            livekit_room_prefix=os.environ.get("JARVIS_LIVEKIT_ROOM_PREFIX", "jarvis"),
            livekit_token_ttl_seconds=int(os.environ.get("JARVIS_LIVEKIT_TOKEN_TTL_SECONDS", "3600")),
            cors_allowed_origins=parse_csv(
                os.environ.get(
                    "JARVIS_VOICE_AGENT_CORS_ALLOWED_ORIGIN",
                    "http://127.0.0.1:13000,http://localhost:13000",
                )
            ),
        )


def parse_csv(value: str) -> tuple[str, ...]:
    items = tuple(item.strip() for item in value.split(",") if item.strip())
    return items or ("http://127.0.0.1:13000",)
