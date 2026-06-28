from __future__ import annotations

import json
import uuid
import urllib.error
import urllib.request
from dataclasses import dataclass


class STTError(RuntimeError):
    pass


@dataclass(frozen=True)
class STTClient:
    url: str
    language: str = "pt"
    timeout_seconds: float = 120

    def transcribe_wav(self, wav: bytes) -> str:
        boundary = f"jarvis-{uuid.uuid4().hex}"
        body = multipart_body(
            boundary,
            fields={"language": self.language},
            files={
                "audio": {
                    "filename": "speech.wav",
                    "content_type": "audio/wav",
                    "content": wav,
                }
            },
        )
        request = urllib.request.Request(
            self.url,
            data=body,
            headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
            method="POST",
        )

        try:
            with urllib.request.urlopen(request, timeout=self.timeout_seconds) as response:
                payload = json.loads(response.read().decode("utf-8"))
        except urllib.error.HTTPError as error:
            detail = error.read().decode("utf-8", errors="replace")
            raise STTError(f"stt http error {error.code}: {detail}") from error
        except urllib.error.URLError as error:
            raise STTError(f"stt unavailable: {error.reason}") from error
        except json.JSONDecodeError as error:
            raise STTError("stt returned invalid json") from error

        transcript = payload.get("transcript")
        if not isinstance(transcript, str):
            raise STTError("stt response missing transcript")
        return transcript.strip()


def multipart_body(boundary: str, *, fields: dict[str, str], files: dict[str, dict]) -> bytes:
    parts: list[bytes] = []
    for name, value in fields.items():
        parts.extend(
            [
                f"--{boundary}\r\n".encode("utf-8"),
                f'Content-Disposition: form-data; name="{name}"\r\n\r\n'.encode("utf-8"),
                value.encode("utf-8"),
                b"\r\n",
            ]
        )
    for name, file_info in files.items():
        parts.extend(
            [
                f"--{boundary}\r\n".encode("utf-8"),
                (
                    f'Content-Disposition: form-data; name="{name}"; '
                    f'filename="{file_info["filename"]}"\r\n'
                ).encode("utf-8"),
                f'Content-Type: {file_info["content_type"]}\r\n\r\n'.encode("utf-8"),
                file_info["content"],
                b"\r\n",
            ]
        )
    parts.append(f"--{boundary}--\r\n".encode("utf-8"))
    return b"".join(parts)
