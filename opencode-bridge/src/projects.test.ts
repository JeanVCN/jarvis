import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, realpathSync, rmSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { loadAllowedProjects } from "./projects.js";

test("loads an explicit allowlist and canonicalizes its directories", () => {
  const root = mkdtempSync(path.join(os.tmpdir(), "jarvis-projects-"));
  const repo = path.join(root, "repo");
  mkdirSync(repo);
  try {
    const projects = loadAllowedProjects(
      JSON.stringify([{ id: "jarvis", name: "Jarvis", directory: repo }]),
    );
    assert.deepEqual(projects, [
      { id: "jarvis", name: "Jarvis", directory: realpathSync(repo) },
    ]);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("rejects empty, malformed, duplicate, relative, and missing project entries", () => {
  assert.throws(() => loadAllowedProjects(undefined), /required/);
  assert.throws(() => loadAllowedProjects("{"), /valid JSON/);
  assert.throws(() => loadAllowedProjects("[]"), /non-empty array/);
  assert.throws(
    () =>
      loadAllowedProjects(
        JSON.stringify([
          { id: "repo", name: "One", directory: "/tmp" },
          { id: "repo", name: "Two", directory: "/tmp" },
        ]),
      ),
    /duplicate project id/,
  );
  assert.throws(
    () => loadAllowedProjects(JSON.stringify([{ id: "repo", name: "Repo", directory: "../repo" }])),
    /absolute path/,
  );
  assert.throws(
    () =>
      loadAllowedProjects(
        JSON.stringify([{ id: "repo", name: "Repo", directory: "/definitely/not/a/project" }]),
      ),
    /does not exist/,
  );
});
