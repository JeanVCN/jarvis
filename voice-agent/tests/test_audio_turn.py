import math
import struct
import unittest

from jarvis_voice_agent.audio_turn import AudioTurnDetector, pcm_to_wav


class FakeFrame:
    def __init__(self, pcm: bytes, sample_rate: int = 16000):
        self.data = pcm
        self.sample_rate = sample_rate
        self.num_channels = 1
        self.samples_per_channel = len(pcm) // 2


class AudioTurnDetectorTest(unittest.TestCase):
    def test_detects_turn_after_silence(self):
        detector = AudioTurnDetector(
            threshold=0.02,
            silence_ms=100,
            min_speech_ms=100,
            max_speech_ms=2000,
        )

        for frame in [tone_frame(), tone_frame(), silence_frame(), silence_frame()]:
            turn = detector.push(frame)
            if turn is not None:
                break

        self.assertIsNotNone(turn)
        self.assertGreater(turn.duration_ms, 0)
        self.assertTrue(turn.wav.startswith(b"RIFF"))

    def test_pcm_to_wav_writes_riff_bytes(self):
        wav = pcm_to_wav(tone_frame().data, sample_rate=16000, num_channels=1)

        self.assertTrue(wav.startswith(b"RIFF"))
        self.assertIn(b"WAVE", wav[:16])


def tone_frame(sample_rate: int = 16000, duration_ms: int = 100) -> FakeFrame:
    samples = int(sample_rate * duration_ms / 1000)
    values = []
    for index in range(samples):
        value = int(math.sin(index / 8) * 12000)
        values.append(struct.pack("<h", value))
    return FakeFrame(b"".join(values), sample_rate)


def silence_frame(sample_rate: int = 16000, duration_ms: int = 100) -> FakeFrame:
    samples = int(sample_rate * duration_ms / 1000)
    return FakeFrame(b"\x00\x00" * samples, sample_rate)


if __name__ == "__main__":
    unittest.main()
