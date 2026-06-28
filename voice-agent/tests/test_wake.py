import unittest

from jarvis_voice_agent.wake import strip_wake_word


class WakeTest(unittest.TestCase):
    def test_strip_wake_word(self):
        command, detected = strip_wake_word("Jarvis, abra o navegador", ("jarvis",))

        self.assertTrue(detected)
        self.assertEqual(command, "abra o navegador")

    def test_ignores_without_wake_word(self):
        command, detected = strip_wake_word("abra o navegador", ("jarvis",))

        self.assertFalse(detected)
        self.assertEqual(command, "abra o navegador")


if __name__ == "__main__":
    unittest.main()
