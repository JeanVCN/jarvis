from __future__ import annotations

import base64
import hashlib
import hmac
import json
import secrets
import time
from dataclasses import dataclass


class LiveKitConfigError(RuntimeError):
    pass


@dataclass(frozen=True)
class LiveKitSession:
    url: str
    token: str
    room: str
    identity: str
    expires_at: int


@dataclass(frozen=True)
class LiveKitTokenIssuer:
    url: str
    api_key: str
    api_secret: str
    room_prefix: str = "jarvis"
    token_ttl_seconds: int = 3600

    def create_session(self, room: str = "", identity: str = "") -> LiveKitSession:
        self.validate()

        now = int(time.time())
        room = safe_name(room) or f"{self.room_prefix}-{secrets.token_hex(4)}"
        identity = safe_name(identity) or f"jarvis-user-{secrets.token_hex(4)}"
        expires_at = now + self.token_ttl_seconds

        payload = {
            "iss": self.api_key,
            "sub": identity,
            "nbf": now - 10,
            "exp": expires_at,
            "video": {
                "room": room,
                "roomJoin": True,
                "canPublish": True,
                "canSubscribe": True,
                "canPublishData": True,
            },
        }

        return LiveKitSession(
            url=self.url,
            token=encode_jwt(payload, self.api_secret),
            room=room,
            identity=identity,
            expires_at=expires_at,
        )

    def validate(self) -> None:
        missing = []
        if not self.url:
            missing.append("JARVIS_LIVEKIT_URL")
        if not self.api_key:
            missing.append("JARVIS_LIVEKIT_API_KEY")
        if not self.api_secret:
            missing.append("JARVIS_LIVEKIT_API_SECRET")
        if missing:
            joined = ", ".join(missing)
            raise LiveKitConfigError(f"missing LiveKit settings: {joined}")


def encode_jwt(payload: dict[str, object], secret: str) -> str:
    header = {"alg": "HS256", "typ": "JWT"}
    segments = [
        base64url_json(header),
        base64url_json(payload),
    ]
    signing_input = ".".join(segments).encode("utf-8")
    signature = hmac.new(secret.encode("utf-8"), signing_input, hashlib.sha256).digest()
    segments.append(base64url(signature))
    return ".".join(segments)


def base64url_json(value: dict[str, object]) -> str:
    data = json.dumps(value, separators=(",", ":"), sort_keys=True).encode("utf-8")
    return base64url(data)


def base64url(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).decode("ascii").rstrip("=")


def safe_name(value: str) -> str:
    value = value.strip()
    allowed = []
    for char in value:
        if char.isalnum() or char in ("-", "_"):
            allowed.append(char)
    return "".join(allowed)[:80]
