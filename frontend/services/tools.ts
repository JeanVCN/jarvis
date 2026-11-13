import type {
  ToolExecuteRequest,
  ToolExecuteResponse,
  ToolMetadata,
} from "@/types/tools";

const publicApiUrl = process.env.NEXT_PUBLIC_API_URL ?? "";

export async function listTools(): Promise<ToolMetadata[]> {
  const response = await fetch(`${publicApiUrl}/api/tools`, {
    cache: "no-store",
  });

  if (!response.ok) {
    throw new Error("tools_list_failed");
  }

  return (await response.json()) as ToolMetadata[];
}

export async function executeTool(
  payload: ToolExecuteRequest,
): Promise<ToolExecuteResponse> {
  const response = await fetch(`${publicApiUrl}/api/tools/execute`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error("tool_execution_failed");
  }

  return (await response.json()) as ToolExecuteResponse;
}
