from __future__ import annotations

import io
import struct
import wave
from dataclasses import dataclass


@dataclass(frozen=True)
class AudioTurn:
    wav: bytes
    duration_ms: int
    sample_rate: int
    num_channels: int


class AudioTurnDetector:
    def __init__(
        self,
        *,
        threshold: float,
        silence_ms: int,
        min_speech_ms: int,
        max_speech_ms: int,
    ):
        self.threshold = threshold
        self.silence_ms = silence_ms
        self.min_speech_ms = min_speech_ms
        self.max_speech_ms = max_speech_ms
        self._frames: list[bytes] = []
        self._sample_rate = 0
        self._num_channels = 0
        self._duration_ms = 0
        self._speech_ms = 0
        self._silence_ms = 0
        self._speech_started = False

    def push(self, frame) -> AudioTurn | None:
        pcm = bytes(frame.data)
        duration_ms = max(1, int(frame.samples_per_channel / frame.sample_rate * 1000))
        rms = pcm_rms(pcm)

        if rms >= self.threshold:
            self._speech_started = True
            self._silence_ms = 0
        elif self._speech_started:
            self._silence_ms += duration_ms

        if self._speech_started:
            self._frames.append(pcm)
            self._sample_rate = frame.sample_rate
            self._num_channels = frame.num_channels
            self._duration_ms += duration_ms
            self._speech_ms += duration_ms

        if not self._speech_started:
            return None

        should_flush = (
            self._speech_ms >= self.min_speech_ms
            and self._silence_ms >= self.silence_ms
        ) or self._duration_ms >= self.max_speech_ms

        if not should_flush:
            return None

        return self.flush()

    def flush(self) -> AudioTurn | None:
        if not self._frames or self._duration_ms < self.min_speech_ms:
            self.reset()
            return None

        wav = pcm_to_wav(
            b"".join(self._frames),
            sample_rate=self._sample_rate,
            num_channels=self._num_channels,
        )
        turn = AudioTurn(
            wav=wav,
            duration_ms=self._duration_ms,
            sample_rate=self._sample_rate,
            num_channels=self._num_channels,
        )
        self.reset()
        return turn

    def reset(self) -> None:
        self._frames = []
        self._sample_rate = 0
        self._num_channels = 0
        self._duration_ms = 0
        self._speech_ms = 0
        self._silence_ms = 0
        self._speech_started = False


def pcm_to_wav(pcm: bytes, *, sample_rate: int, num_channels: int) -> bytes:
    output = io.BytesIO()
    with wave.open(output, "wb") as wav:
        wav.setnchannels(num_channels)
        wav.setsampwidth(2)
        wav.setframerate(sample_rate)
        wav.writeframes(pcm)
    return output.getvalue()


def pcm_rms(pcm: bytes) -> float:
    if not pcm:
        return 0
    sample_count = len(pcm) // 2
    if sample_count == 0:
        return 0
    total = 0
    for (sample,) in struct.iter_unpack("<h", pcm[: sample_count * 2]):
        total += sample * sample
    return (total / sample_count) ** 0.5 / 32768
