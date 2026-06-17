import type { ChatRequest, ChatResponse } from "@/types/chat";

const publicApiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://127.0.0.1:18080";

export async function sendChatMessage(message: string): Promise<ChatResponse> {
  const payload: ChatRequest = { message };

  const response = await fetch(`${publicApiUrl}/api/chat`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error("chat_request_failed");
  }

  return (await response.json()) as ChatResponse;
}

export async function streamChatMessage(
  message: string,
  onDelta: (delta: string) => void,
): Promise<void> {
  const payload: ChatRequest = { message };

  const response = await fetch(`${publicApiUrl}/api/chat/stream`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok || !response.body) {
    throw new Error("chat_stream_request_failed");
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { value, done } = await reader.read();
    if (done) {
      break;
    }

    buffer += decoder.decode(value, { stream: true });
    const events = buffer.split("\n\n");
    buffer = events.pop() ?? "";

    for (const eventText of events) {
      const event = parseSSEEvent(eventText);
      if (event.event === "delta") {
        const data = JSON.parse(event.data) as { delta: string };
        onDelta(data.delta);
      }
      if (event.event === "error") {
        throw new Error("chat_stream_failed");
      }
      if (event.event === "done") {
        return;
      }
    }
  }
}

function parseSSEEvent(eventText: string): { event: string; data: string } {
  const lines = eventText.split("\n");
  let event = "message";
  const dataLines: string[] = [];

  for (const line of lines) {
    if (line.startsWith("event:")) {
      event = line.slice("event:".length).trim();
    }
    if (line.startsWith("data:")) {
      dataLines.push(line.slice("data:".length).trimStart());
    }
  }

  return {
    event,
    data: dataLines.join("\n"),
  };
}
