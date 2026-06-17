import type { VoiceConversationResponse, VoiceStatus } from "@/types/voice";

const publicApiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://127.0.0.1:18080";

export async function sendVoiceConversation(
  audio: Blob,
): Promise<VoiceConversationResponse> {
  const payload = new FormData();
  payload.append("audio", audio, "speech.webm");
  payload.append("language", "pt");

  const response = await fetch(`${publicApiUrl}/api/voice/conversation`, {
    method: "POST",
    body: payload,
  });

  if (!response.ok) {
    throw new Error("voice_conversation_failed");
  }

  return (await response.json()) as VoiceConversationResponse;
}

export async function getVoiceStatus(): Promise<VoiceStatus> {
  const response = await fetch(`${publicApiUrl}/api/voice/status`, {
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error("voice_status_failed");
  }

  return (await response.json()) as VoiceStatus;
}

export function audioBase64ToBlob(base64: string, contentType: string): Blob {
  const binary = window.atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }

  return new Blob([bytes], { type: contentType });
}
