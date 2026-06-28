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
