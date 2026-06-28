import { Room } from "livekit-client";

import type {
  RealtimeVoiceConnection,
  RealtimeVoiceSession,
  RealtimeVoiceSessionRequest,
} from "@/types/realtime-voice";

const publicVoiceAgentUrl =
  process.env.NEXT_PUBLIC_VOICE_AGENT_URL ?? "http://127.0.0.1:18100";

export async function createRealtimeVoiceSession(
  request: RealtimeVoiceSessionRequest = {},
): Promise<RealtimeVoiceSession> {
  const response = await fetch(`${publicVoiceAgentUrl}/livekit/session`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    throw new Error("realtime_voice_session_failed");
  }

  return (await response.json()) as RealtimeVoiceSession;
}

export async function connectRealtimeVoiceRoom(
  session: RealtimeVoiceSession,
): Promise<RealtimeVoiceConnection> {
  const room = new Room({
    adaptiveStream: true,
    dynacast: true,
  });

  await room.connect(session.url, session.token);
  await room.localParticipant.setMicrophoneEnabled(true);

  return {
    roomName: session.room,
    identity: session.identity,
    disconnect: async () => {
      await room.localParticipant.setMicrophoneEnabled(false);
      room.disconnect();
    },
  };
}
