from __future__ import annotations

import json
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse

from jarvis_voice_agent.config import Settings
from jarvis_voice_agent.jarvis_core import JarvisCoreClient, JarvisCoreError
from jarvis_voice_agent.livekit import LiveKitConfigError, LiveKitTokenIssuer


class VoiceAgentServer(ThreadingHTTPServer):
    def __init__(self, settings: Settings):
        super().__init__((settings.host, settings.port), Handler)
        self.settings = settings
        self.core = JarvisCoreClient(settings.core_url, settings.request_timeout_seconds)
        self.livekit = LiveKitTokenIssuer(
            url=settings.livekit_url,
            api_key=settings.livekit_api_key,
            api_secret=settings.livekit_api_secret,
            room_prefix=settings.livekit_room_prefix,
            token_ttl_seconds=settings.livekit_token_ttl_seconds,
        )


class Handler(BaseHTTPRequestHandler):
    server: VoiceAgentServer

    def do_OPTIONS(self) -> None:
        self.send_response(HTTPStatus.NO_CONTENT)
        self.send_cors_headers()
        self.send_header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()

    def do_GET(self) -> None:
        if urlparse(self.path).path != "/health":
            self.send_error(HTTPStatus.NOT_FOUND)
            return

        self.write_json(
            {
                "status": "ok",
                "transport": self.server.settings.transport,
                "coreUrl": self.server.settings.core_url,
                "language": self.server.settings.language,
            }
        )

    def do_POST(self) -> None:
        path = urlparse(self.path).path
        if path == "/livekit/session":
            self.handle_livekit_session()
            return
        if path != "/chat":
            self.send_error(HTTPStatus.NOT_FOUND)
            return

        try:
            payload = self.read_json()
            message = str(payload.get("message", "")).strip()
            if not message:
                self.write_json({"error": "message_required"}, status=HTTPStatus.BAD_REQUEST)
                return

            response = self.server.core.chat(message)
            self.write_json({"response": response})
        except JarvisCoreError as error:
            self.write_json(
                {"error": "core_unavailable", "detail": str(error)},
                status=HTTPStatus.BAD_GATEWAY,
            )
        except json.JSONDecodeError:
            self.write_json({"error": "invalid_json"}, status=HTTPStatus.BAD_REQUEST)

    def handle_livekit_session(self) -> None:
        try:
            payload = self.read_json()
            session = self.server.livekit.create_session(
                room=str(payload.get("room", "")),
                identity=str(payload.get("identity", "")),
            )
            self.write_json(
                {
                    "url": session.url,
                    "token": session.token,
                    "room": session.room,
                    "identity": session.identity,
                    "expiresAt": session.expires_at,
                }
            )
        except LiveKitConfigError as error:
            self.write_json(
                {"error": "livekit_not_configured", "detail": str(error)},
                status=HTTPStatus.SERVICE_UNAVAILABLE,
            )
        except json.JSONDecodeError:
            self.write_json({"error": "invalid_json"}, status=HTTPStatus.BAD_REQUEST)

    def read_json(self) -> dict[str, object]:
        content_length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(content_length)
        if not raw:
            return {}
        value = json.loads(raw.decode("utf-8"))
        if not isinstance(value, dict):
            raise json.JSONDecodeError("expected object", raw.decode("utf-8"), 0)
        return value

    def write_json(self, payload: dict[str, object], status: int = HTTPStatus.OK) -> None:
        data = json.dumps(payload).encode("utf-8")
        self.send_response(status)
        self.send_cors_headers()
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def send_cors_headers(self) -> None:
        origin = self.headers.get("Origin", "")
        allowed = self.server.settings.cors_allowed_origins
        if origin in allowed:
            self.send_header("Access-Control-Allow-Origin", origin)
            self.send_header("Vary", "Origin")

    def log_message(self, format: str, *args: object) -> None:
        return


def main() -> None:
    settings = Settings.from_env()
    server = VoiceAgentServer(settings)
    print(
        "jarvis voice agent listening on "
        f"http://{settings.host}:{settings.port} transport={settings.transport} core={settings.core_url}",
        flush=True,
    )
    server.serve_forever()


if __name__ == "__main__":
    main()
