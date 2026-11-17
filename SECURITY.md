# Security review instructions

This file guides security reviews of changes to Jarvis. Review the changed
code against the actual trust boundaries below; do not treat planned controls
as if they were already enforced.

## Trust boundaries and review priorities

- **WhatsApp identity and input:** the linked account is restricted to the
  configured owner's own **Você** chat. Verify sender/account checks happen
  before message extraction, media download, persistence, or forwarding. Treat
  message text, captions, transcripts, media metadata, and provider responses
  as untrusted input.
- **Voice privacy:** WhatsApp voice notes use local STT. The audio must not be
  sent to GPT/OpenCode or written to Jarvis history/logs. The transcript is
  forwarded only to the destination selected by the existing text `/code`
  mode: normal mode sends text to GPT/Codex; code mode sends text to the
  configured OpenCode session. Audio must not switch modes or select an
  executor by inference. A routing failure must not fall back to another
  provider.
- **Image privacy:** original JPEG, PNG, or WebP photos (up to 10 MiB) are sent
  inline to the explicitly selected Codex or OpenCode destination without local
  OCR or transformation. The Jarvis process does not persist the bytes, but the
  Codex thread or OpenCode session/provider may retain them. Verify no path
  silently copies them into Jarvis SQLite, logs, or a different provider.
  Image captions and media must never change the active `/code` route.
- **GPT/Codex isolation:** GPT/Codex must not gain local filesystem, shell,
  desktop, or arbitrary network tools. The persistent conversation history is
  stored by Codex under `CODEX_HOME`; the Jarvis SQLite database should retain
  only the provider thread ID and routing metadata, not duplicate chat bodies.
- **OpenCode authority:** OpenCode can run with the local user's authority.
  Verify project allowlisting, loopback binding, bridge authentication,
  explicit routing, permission handling, and safe behavior after timeouts or
  uncertain outcomes. Do not treat model instructions or sandbox labels as a
  replacement for those controls.
- **Remote access:** there is no Jarvis login. Core, bridge, and OpenCode should
  remain private; Tailscale/device access rules are a network boundary and
  must not be described as validated unless the change includes that proof.
- **Credentials and persistence:** check WhatsApp linked-device credentials,
  Codex authentication data, bridge tokens, database permissions, logs, and
  error paths for disclosure. Avoid persisting message/audio content unless a
  specifically approved design requires it.
- **Planned protections:** device-specific authorization and TOTP for `/code`
  or `/cmd` are documented options, not current guarantees. `/cmd` is deferred
  until the Android/Google Home bridge exists. Review changes so they do not
  accidentally imply these protections are already active.

## Reporting

Report a concrete vulnerability only when the changed code creates a plausible
attack path or meaningful privacy/security impact. State the relevant attacker
capability, affected boundary, and consequence. Do not include secrets or
unnecessary exploit details in a public issue. Use GitHub's private vulnerability
reporting feature when available; otherwise contact the repository owner
privately. This repository does not define a response-time guarantee.
