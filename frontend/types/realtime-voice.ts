export type RealtimeVoiceSession = {
  url: string;
  token: string;
  room: string;
  identity: string;
  expiresAt: number;
};

export type RealtimeVoiceSessionRequest = {
  room?: string;
  identity?: string;
};

export type RealtimeVoiceConnection = {
  roomName: string;
  identity: string;
  disconnect: () => Promise<void>;
};

export type RealtimeVoiceEvent =
  | {
      type: "transcript";
      transcript: string;
      command: string;
      wakeDetected: boolean;
    }
  | {
      type: "response";
      transcript: string;
      command: string;
      response: string;
    }
  | {
      type: "ignored";
      reason: string;
      transcript?: string;
    }
  | {
      type: "error";
      message: string;
      detail?: string;
    };
