from __future__ import annotations

import unicodedata


def strip_wake_word(transcript: str, wake_words: tuple[str, ...]) -> tuple[str, bool]:
    normalized = normalize(transcript)
    for wake_word in wake_words:
        normalized_wake = normalize(wake_word)
        if not normalized_wake:
            continue
        index = normalized.find(normalized_wake)
        if index < 0:
            continue
        original = transcript.strip()
        command = original[:index] + original[index + len(wake_word):]
        command = command.strip(" ,;:.!?")
        return command or original, True
    return transcript.strip(), False


def normalize(value: str) -> str:
    value = unicodedata.normalize("NFD", value.lower())
    value = "".join(char for char in value if unicodedata.category(char) != "Mn")
    return value.strip()
