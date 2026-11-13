"""Lazy CPU engines. Only the selected STT/TTS is loaded; no remote audio by default."""
from __future__ import annotations

import os
import threading
from dataclasses import dataclass
from collections.abc import Iterator
import numpy as np


@dataclass(frozen=True)
class Profile:
    stt: str = "base"
    tts: str = "kokoro"
    voice: str = "pf_dora"
    turn: str = "silence"

    def validate(self):
        if self.stt not in ("base", "small", "fixture"):
            raise ValueError("invalid_stt")
        if self.tts not in ("kokoro", "pocket", "tone"):
            raise ValueError("invalid_tts")
        if self.turn not in ("silence", "smart"):
            raise ValueError("invalid_turn_detector")
        if self.tts == "kokoro" and self.voice not in ("pf_dora", "pm_alex", "pm_santa"):
            raise ValueError("invalid_kokoro_voice")
        if self.tts == "pocket" and self.voice != "rafael":
            raise ValueError("invalid_pocket_voice")
        if (self.stt == "fixture" or self.tts == "tone") and os.getenv("JARVIS_TEST_AUDIO") != "true":
            raise ValueError("synthetic_audio_disabled")


class Engines:
    def __init__(self, profile: Profile):
        profile.validate()
        self.profile = profile
        self.stt = None
        self.tts = None
        self.voice_state = None
        if profile.stt != "fixture":
            from faster_whisper import WhisperModel
            self.stt = WhisperModel(profile.stt, device="cpu", compute_type="int8", cpu_threads=2, num_workers=1)
        if profile.tts != "tone":
            import torch
            torch.set_num_threads(2)
        if profile.tts == "kokoro":
            from kokoro import KPipeline
            self.tts = KPipeline(lang_code="p", device="cpu")
        elif profile.tts == "pocket":
            # Voice/model licenses must be reviewed before enabling this profile.
            if os.getenv("JARVIS_POCKET_LICENSE_ACCEPTED") != "true":
                raise ValueError("pocket_voice_license_review_required")
            from pocket_tts import TTSModel
            self.tts = TTSModel.load_model(language="portuguese")
            self.voice_state = self.tts.get_state_for_audio_prompt("rafael")

    def transcribe(self, pcm: bytes) -> str:
        if self.profile.stt == "fixture":
            return os.getenv("JARVIS_FIXTURE_TRANSCRIPT", "Quero uma resposta longa.")
        audio = np.frombuffer(pcm, dtype="<i2").astype(np.float32) / 32768
        segments, _ = self.stt.transcribe(audio, language="pt", beam_size=1, condition_on_previous_text=False, vad_filter=False)
        return "".join(s.text for s in segments).strip()

    def synthesize(self, text: str, stop: threading.Event) -> Iterator[tuple[np.ndarray, int]]:
        if self.profile.tts == "tone":
            # Explicit test fixture, never presented as speech synthesis.
            for _ in range(10):
                if stop.is_set(): return
                t = np.arange(2400) / 24000
                yield (0.08 * np.sin(2 * np.pi * 440 * t)).astype(np.float32), 24000
        elif self.profile.tts == "kokoro":
            for result in self.tts(text, voice=self.profile.voice, speed=1):
                if stop.is_set(): return
                audio = result.audio if hasattr(result, "audio") else result[2]
                yield np.asarray(audio, dtype=np.float32), 24000
        else:
            for audio in self.tts.generate_audio_stream(self.voice_state, text, stop=stop):
                if stop.is_set(): return
                yield audio.detach().cpu().numpy().reshape(-1), self.tts.sample_rate
