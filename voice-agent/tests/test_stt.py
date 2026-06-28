import unittest

from jarvis_voice_agent.stt import multipart_body


class STTTest(unittest.TestCase):
    def test_multipart_body_contains_audio_and_language(self):
        body = multipart_body(
            "boundary",
            fields={"language": "pt"},
            files={
                "audio": {
                    "filename": "speech.wav",
                    "content_type": "audio/wav",
                    "content": b"wav",
                }
            },
        )

        self.assertIn(b'name="language"', body)
        self.assertIn(b"pt", body)
        self.assertIn(b'filename="speech.wav"', body)
        self.assertIn(b"audio/wav", body)
        self.assertTrue(body.endswith(b"--boundary--\r\n"))


if __name__ == "__main__":
    unittest.main()
