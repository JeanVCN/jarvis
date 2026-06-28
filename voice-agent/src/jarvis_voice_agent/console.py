from __future__ import annotations

from jarvis_voice_agent.config import Settings
from jarvis_voice_agent.jarvis_core import JarvisCoreClient, JarvisCoreError


def main() -> None:
    settings = Settings.from_env()
    core = JarvisCoreClient(settings.core_url, settings.request_timeout_seconds)

    print(f"Jarvis voice console connected to {settings.core_url}")
    print("Type a message and press Enter. Empty input exits.")

    while True:
        message = input("> ").strip()
        if not message:
            return

        try:
            for chunk in core.stream_chat(message):
                print(chunk, end="", flush=True)
            print()
        except JarvisCoreError as error:
            print(f"[core error] {error}")
