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
            },
        ):
            settings = Settings.from_env()

        self.assertEqual(settings.livekit_room, "jarvis-lab")
        self.assertEqual(settings.livekit_agent_identity, "agent-01")


if __name__ == "__main__":
    unittest.main()
