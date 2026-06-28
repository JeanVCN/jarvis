import base64
import json
import unittest

from jarvis_voice_agent.livekit import LiveKitConfigError, LiveKitTokenIssuer, safe_name


class LiveKitTokenIssuerTest(unittest.TestCase):
    def test_create_session_generates_join_token(self):
        issuer = LiveKitTokenIssuer(
            url="ws://127.0.0.1:7880",
            api_key="devkey",
            api_secret="secret",
            room_prefix="jarvis",
            token_ttl_seconds=3600,
        )

        session = issuer.create_session(room="jarvis-main", identity="desktop")
        payload = decode_payload(session.token)

        self.assertEqual(session.url, "ws://127.0.0.1:7880")
        self.assertEqual(session.room, "jarvis-main")
        self.assertEqual(session.identity, "desktop")
        self.assertEqual(payload["iss"], "devkey")
        self.assertEqual(payload["sub"], "desktop")
        self.assertEqual(payload["video"]["room"], "jarvis-main")
        self.assertTrue(payload["video"]["roomJoin"])
        self.assertTrue(payload["video"]["canPublish"])
        self.assertTrue(payload["video"]["canSubscribe"])
        self.assertTrue(payload["video"]["canPublishData"])

    def test_missing_settings_raise_config_error(self):
        issuer = LiveKitTokenIssuer(url="", api_key="", api_secret="")

        with self.assertRaisesRegex(LiveKitConfigError, "JARVIS_LIVEKIT_URL"):
            issuer.create_session()

    def test_safe_name_removes_unsafe_characters(self):
        self.assertEqual(safe_name(" sala principal!* "), "salaprincipal")
        self.assertEqual(safe_name("android-01_user"), "android-01_user")


def decode_payload(token: str) -> dict:
    payload = token.split(".")[1]
    padded = payload + "=" * (-len(payload) % 4)
    data = base64.urlsafe_b64decode(padded.encode("ascii"))
    return json.loads(data.decode("utf-8"))


if __name__ == "__main__":
    unittest.main()
