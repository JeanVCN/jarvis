export type ToolMetadata = {
  name: string;
  description: string;
};

export type ToolExecuteRequest = {
  tool: string;
  input: string;
};

export type ToolExecuteResponse = {
  result: string;
};
