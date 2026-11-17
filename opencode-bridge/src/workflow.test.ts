import assert from "node:assert/strict";
import { mkdtempSync, realpathSync, rmSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import type { OpenCodeClient } from "@opencode/client";
import type { AllowedProject } from "./projects.js";
import { createProjectSession, findCreatedSession, interruptProjectSession, readProjectSession, safeDiffPath, sendProjectPrompt, validOperationID } from "./workflow.js";

const operationID = "0123456789abcdef0123456789abcdef";

function project(directory: string): AllowedProject {
  return { id: "jarvis", name: "Jarvis", directory };
}

test("operation IDs and paths are constrained before crossing the bridge", () => {
  assert.equal(validOperationID(operationID), true);
  assert.equal(validOperationID("../etc/passwd"), false);
  assert.equal(safeDiffPath("src/frontend.tsx"), "src/frontend.tsx");
  assert.equal(safeDiffPath("/home/jean/project/src/main.go"), "main.go");
  assert.equal(safeDiffPath("../../secret"), "secret");
});

test("session creation binds a configured directory and ask-only permissions", async () => {
  const directory = mkdtempSync(path.join(os.tmpdir(), "jarvis-project-"));
  let received: unknown;
  const client = {
    session: {
      create: async (input: unknown) => {
        received = input;
        return { id: "ses_test", title: "Nova sessão", time: { updated: 42 }, outcome: undefined };
      },
    },
  } as unknown as OpenCodeClient;
  try {
    const result = await createProjectSession(client, project(directory), operationID);
    assert.equal(result.id, "ses_test");
    assert.deepEqual(received, {
      location: { directory },
      permissions: [{ action: "*", resource: "*", effect: "ask" }],
      metadata: { jarvis_operation_id: operationID },
    });
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});

test("prompt rejects a session outside its allowlisted project before sending", async () => {
  const directory = mkdtempSync(path.join(os.tmpdir(), "jarvis-project-"));
  let sent = false;
  const client = {
    session: {
      get: async () => ({ location: { directory: os.tmpdir() } }),
      prompt: async () => { sent = true; return { id: "msg_test" }; },
    },
  } as unknown as OpenCodeClient;
  try {
    await assert.rejects(
      sendProjectPrompt(client, project(directory), "ses_foreign", "test"),
      /session_not_in_project/,
    );
    assert.equal(sent, false);
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});

test("prompt omits the Jarvis operation ID from OpenCode's message ID field", async () => {
  const directory = mkdtempSync(path.join(os.tmpdir(), "jarvis-project-"));
  let received: unknown;
  const client = {
    session: {
      get: async () => ({ location: { directory } }),
      prompt: async (input: unknown) => {
        received = input;
        return { id: "msg_generated_by_opencode" };
      },
    },
  } as unknown as OpenCodeClient;
  try {
    const result = await sendProjectPrompt(client, project(directory), "ses_test", "test prompt");
    assert.deepEqual(received, { sessionID: "ses_test", text: "test prompt" });
    assert.deepEqual(result, { inbox_id: "msg_generated_by_opencode" });
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});

test("interrupt validates project ownership and only requests interruption while busy", async () => {
  const directory = mkdtempSync(path.join(os.tmpdir(), "jarvis-project-"));
  let interrupted = false;
  const client = {
    session: {
      get: async () => ({ location: { directory }, time: { busy: true } }),
      interrupt: async () => { interrupted = true; },
    },
  } as unknown as OpenCodeClient;
  try {
    await interruptProjectSession(client, project(directory), "ses_running");
    assert.equal(interrupted, true);
    const foreign = {
      session: {
        get: async () => ({ location: { directory: os.tmpdir() }, time: { busy: true } }),
        interrupt: async () => { throw new Error("must_not_interrupt"); },
      },
    } as unknown as OpenCodeClient;
    await assert.rejects(interruptProjectSession(foreign, project(directory), "ses_foreign"), /session_not_in_project/);
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});

test("session view exposes user and assistant text, never reasoning/tool payloads", async () => {
  const directory = mkdtempSync(path.join(os.tmpdir(), "jarvis-project-"));
  const client = {
    session: {
      get: async () => ({ id: "ses_test", title: "Session", time: { updated: 42 }, location: { directory }, outcome: undefined }),
      diff: async () => [{ file: "src/main.go", patch: "+safe", additions: 1, deletions: 0, status: "modified" }],
    },
    message: {
      list: async () => ({ data: [
        { id: "assistant1", type: "assistant", time: { created: 2, completed: 3 }, content: [
          { type: "text", text: "answer" },
          { type: "reasoning", text: "private thought" },
          { type: "tool", name: "shell", id: "tool1", state: { name: "shell", text: "secret command" } },
        ] },
        { id: "user1", type: "user", text: "hello", time: { created: 1 } },
      ] }),
    },
    permission: { list: async () => [] },
  } as unknown as OpenCodeClient;
  try {
    const result = await readProjectSession(client, project(directory), "ses_test");
    assert.deepEqual(result.messages.map(({ role, content }) => ({ role, content })), [
      { role: "user", content: "hello" },
      { role: "assistant", content: "answer" },
    ]);
    assert.equal(JSON.stringify(result).includes("private thought"), false);
    assert.equal(JSON.stringify(result).includes("secret command"), false);
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});

test("recovery requires a unique operation match and verifies its project", async () => {
  const tempDirectory = mkdtempSync(path.join(os.tmpdir(), "jarvis-project-"));
  const directory = realpathSync(tempDirectory);
  let sessionDirectory = directory;
  let listedIDs = ["ses_recovered"];
  const lookups: string[] = [];
  const client = {
    session: {
      list: async () => ({
        data: listedIDs.map((id) => ({ id, metadata: { jarvis_operation_id: operationID } })),
      }),
      get: async ({ sessionID }: { sessionID: string }) => {
        lookups.push(sessionID);
        return { id: sessionID, location: { directory: sessionDirectory }, time: { updated: 42 } };
      },
    },
  } as unknown as OpenCodeClient;
  try {
    assert.equal((await findCreatedSession(client, project(directory), operationID))?.id, "ses_recovered");

    sessionDirectory = os.tmpdir();
    await assert.rejects(findCreatedSession(client, project(directory), operationID), /session_not_in_project/);

    listedIDs = ["ses_recovered", "ses_duplicate"];
    assert.equal(await findCreatedSession(client, project(directory), operationID), undefined);
    assert.deepEqual(lookups, ["ses_recovered", "ses_recovered"]);
  } finally {
    rmSync(tempDirectory, { recursive: true, force: true });
  }
});
