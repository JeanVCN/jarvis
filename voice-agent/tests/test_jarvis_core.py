import unittest

from jarvis_voice_agent.jarvis_core import JarvisCoreError, parse_sse


class ParseSSETest(unittest.TestCase):
    def test_yields_delta_events(self):
        lines = iter(
            [
                b"event: delta\n",
                b'data: {"delta":"Ola"}\n',
                b"\n",
                b"event: delta\n",
                b'data: {"delta":" mundo"}\n',
                b"\n",
                b"event: done\n",
                b'data: {"done":true}\n',
                b"\n",
            ]
        )

        self.assertEqual(list(parse_sse(lines)), ["Ola", " mundo"])

    def test_raises_on_error_event(self):
        lines = iter(
            [
                b"event: error\n",
                b'data: {"error":"llm_unavailable"}\n',
                b"\n",
            ]
        )

        with self.assertRaisesRegex(JarvisCoreError, "llm_unavailable"):
            list(parse_sse(lines))

    def test_ignores_unknown_events(self):
        lines = iter(
            [
                b"event: ping\n",
                b'data: {"ok":true}\n',
                b"\n",
                b"event: delta\n",
                b'data: {"delta":"ok"}\n',
                b"\n",
            ]
        )

        self.assertEqual(list(parse_sse(lines)), ["ok"])


if __name__ == "__main__":
    unittest.main()
