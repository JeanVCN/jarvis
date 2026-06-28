export type VoiceConversationResponse = {
  transcript: string;
  response: string;
  audioContentType: string;
  audioBase64: string;
  ignored: boolean;
  wakeWordDetected: boolean;
};

export type VoiceStatus = {
  sttAvailable: boolean;
  ttsAvailable: boolean;
  sttCommand: string;
  ttsCommand: string;
  message: string;
  privacyMode: string;
};

export type AssistantState = "idle" | "listening" | "thinking" | "speaking";
