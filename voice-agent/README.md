# Jarvis Voice Agent

This module is the new realtime voice boundary for Jarvis.

The current browser voice flow still exists as a fallback, but the intended architecture is:

```text
Desktop browser / Android app
        |
        v
Realtime transport, usually LiveKit WebRTC
        |
        v
Jarvis Voice Agent
        |
        v
Jarvis Core Go API
  - chat streaming
  - tools
  - memory
  - permissions
```

The first implemented slice keeps the agent dependency-light and testable:

- `JarvisCoreClient` streams responses from `POST /api/chat/stream`.
- `jarvis-voice-console` validates the agent-to-core loop without audio.
- `jarvis-voice-agent` exposes a tiny local HTTP service for health checks and chat smoke tests.
- `POST /livekit/session` issues LiveKit-compatible room tokens for desktop and Android clients.
- Pipecat/LiveKit dependencies are optional and will be used by the next transport implementation.

## Local smoke test

Start the Jarvis backend first, then run:

```bash
cd voice-agent
python -m venv .venv
. .venv/bin/activate
python -m pip install -e .
JARVIS_CORE_URL=http://127.0.0.1:18080 jarvis-voice-console
```

Type a message such as:

```text
que horas sao?
```

If the core streams a response, the realtime agent can reuse the same path after STT.

## Local HTTP service

```bash
cd voice-agent
JARVIS_CORE_URL=http://127.0.0.1:18080 python -m jarvis_voice_agent.server
```

Health:

```bash
curl http://127.0.0.1:18100/health
```

Chat smoke test:

```bash
curl -X POST http://127.0.0.1:18100/chat \
  -H 'Content-Type: application/json' \
  -d '{"message":"que horas sao?"}'
```

LiveKit session token:

```bash
curl -X POST http://127.0.0.1:18100/livekit/session \
  -H 'Content-Type: application/json' \
  -d '{"room":"jarvis-main","identity":"desktop"}'
```

Default development values:

```env
JARVIS_LIVEKIT_URL=ws://127.0.0.1:7880
JARVIS_LIVEKIT_API_KEY=devkey
JARVIS_LIVEKIT_API_SECRET=secret
JARVIS_LIVEKIT_ROOM_PREFIX=jarvis
JARVIS_LIVEKIT_TOKEN_TTL_SECONDS=3600
```

The token endpoint does not execute tools. It only gives a client permission to join a LiveKit room. Command execution still goes through the Jarvis Core.

Run the local LiveKit server in development mode from the repository root:

```bash
docker compose --profile realtime up livekit
```

Run the Jarvis realtime room participant:

```bash
cd voice-agent
python -m pip install -r requirements-realtime.txt
JARVIS_LIVEKIT_ROOM=jarvis-main python -m jarvis_voice_agent.realtime
```

This participant joins the room as `jarvis-agent`, subscribes to audio tracks and logs received audio frames. The next step is to feed those frames into STT/turn detection.

## Realtime direction

The target transport is LiveKit WebRTC with a Pipecat pipeline:

```text
LiveKit room audio
  -> VAD / turn detection
  -> streaming STT
  -> JarvisCoreClient.stream_chat
  -> streaming TTS
  -> LiveKit room audio
```

The Go backend remains the authority for tools and permissions. The Python agent should not execute local OS commands directly.
