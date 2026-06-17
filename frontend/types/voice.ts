export type VoiceConversationResponse = {
  transcript: string;
  response: string;
  audioContentType: string;
  audioBase64: string;
};

export type VoiceStatus = {
  sttAvailable: boolean;
  ttsAvailable: boolean;
  sttCommand: string;
  ttsCommand: string;
  message: string;
};

export type AssistantState = "idle" | "listening" | "thinking" | "speaking";
