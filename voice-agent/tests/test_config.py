import unittest
from unittest.mock import patch

from jarvis_voice_agent.config import Settings, parse_csv


class ConfigTest(unittest.TestCase):
    def test_parse_csv_trims_empty_items(self):
        self.assertEqual(parse_csv(" a, ,b "), ("a", "b"))

    def test_parse_csv_returns_local_default_when_empty(self):
        self.assertEqual(parse_csv(""), ("http://127.0.0.1:13000",))

    def test_settings_load_livekit_room_and_agent_identity(self):
        with patch.dict(
            "os.environ",
            {
                "JARVIS_LIVEKIT_ROOM": "jarvis-lab",
                "JARVIS_LIVEKIT_AGENT_IDENTITY": "agent-01",
                "JARVIS_LIVEKIT_CONNECT_TIMEOUT_SECONDS": "3.5",
                "JARVIS_LIVEKIT_CONNECT_RETRIES": "4",
                "JARVIS_LIVEKIT_CONNECT_RETRY_DELAY_SECONDS": "1.5",
                "JARVIS_REALTIME_STT_URL": "http://127.0.0.1:18081/transcribe",
                "JARVIS_REALTIME_WAKE_REQUIRED": "false",
            },
        ):
            settings = Settings.from_env()

        self.assertEqual(settings.livekit_room, "jarvis-lab")
        self.assertEqual(settings.livekit_agent_identity, "agent-01")
        self.assertEqual(settings.livekit_connect_timeout_seconds, 3.5)
        self.assertEqual(settings.livekit_connect_retries, 4)
        self.assertEqual(settings.livekit_connect_retry_delay_seconds, 1.5)
        self.assertFalse(settings.realtime_wake_required)
        self.assertEqual(settings.realtime_stt_url, "http://127.0.0.1:18081/transcribe")


if __name__ == "__main__":
    unittest.main()
