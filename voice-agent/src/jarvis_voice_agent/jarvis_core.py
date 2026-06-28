from __future__ import annotations

import json
import urllib.error
import urllib.request
from collections.abc import Iterator
from dataclasses import dataclass


class JarvisCoreError(RuntimeError):
    """Raised when the Jarvis Core API cannot complete a request."""


@dataclass(frozen=True)
class JarvisCoreClient:
    base_url: str
    timeout_seconds: float = 300

    def stream_chat(self, message: str) -> Iterator[str]:
        message = message.strip()
        if not message:
            raise ValueError("message is required")

        request = urllib.request.Request(
            f"{self.base_url.rstrip('/')}/api/chat/stream",
            data=json.dumps({"message": message}).encode("utf-8"),
            headers={
                "Accept": "text/event-stream",
                "Content-Type": "application/json",
            },
            method="POST",
        )

        try:
            with urllib.request.urlopen(request, timeout=self.timeout_seconds) as response:
                if response.status != 200:
                    raise JarvisCoreError(f"unexpected core status: {response.status}")
                yield from parse_sse(response)
        except urllib.error.HTTPError as error:
            detail = error.read().decode("utf-8", errors="replace")
            raise JarvisCoreError(f"core http error {error.code}: {detail}") from error
        except urllib.error.URLError as error:
            raise JarvisCoreError(f"core unavailable: {error.reason}") from error

    def chat(self, message: str) -> str:
        return "".join(self.stream_chat(message)).strip()


def parse_sse(lines: Iterator[bytes]) -> Iterator[str]:
    event = "message"
    data_lines: list[str] = []

    for raw_line in lines:
        line = raw_line.decode("utf-8", errors="replace").rstrip("\r\n")
        if line == "":
            yield from event_payload(event, data_lines)
            event = "message"
            data_lines = []
            continue

        if line.startswith(":"):
            continue

        field, _, value = line.partition(":")
        if value.startswith(" "):
            value = value[1:]

        if field == "event":
            event = value
        elif field == "data":
            data_lines.append(value)

    yield from event_payload(event, data_lines)


def event_payload(event: str, data_lines: list[str]) -> Iterator[str]:
    if not data_lines:
        return

    payload = "\n".join(data_lines)
    if event == "done":
        return
    if event == "error":
        try:
            error = json.loads(payload).get("error", payload)
        except json.JSONDecodeError:
            error = payload
        raise JarvisCoreError(f"core stream error: {error}")
    if event != "delta":
        return

    try:
        data = json.loads(payload)
    except json.JSONDecodeError as error:
        raise JarvisCoreError(f"invalid stream payload: {payload}") from error

    delta = data.get("delta")
    if isinstance(delta, str) and delta:
        yield delta
