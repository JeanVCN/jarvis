import { realpathSync, statSync } from "node:fs";
import path from "node:path";

export type AllowedProject = {
  id: string;
  name: string;
  directory: string;
};

const projectIDPattern = /^[a-z0-9][a-z0-9_-]{0,63}$/;

export function loadAllowedProjects(
  raw: string | undefined,
  resolveDirectory: (directory: string) => string = realpathSync,
  isDirectory: (directory: string) => boolean = (directory) =>
    statSync(directory).isDirectory(),
): AllowedProject[] {
  if (!raw?.trim()) {
    throw new Error("JARVIS_OPENCODE_PROJECTS is required when the bridge is enabled");
  }

  let entries: unknown;
  try {
    entries = JSON.parse(raw);
  } catch {
    throw new Error("JARVIS_OPENCODE_PROJECTS must be valid JSON");
  }
  if (!Array.isArray(entries) || entries.length === 0) {
    throw new Error("JARVIS_OPENCODE_PROJECTS must be a non-empty array");
  }

  const ids = new Set<string>();
  return entries.map((entry, index) => {
    if (!entry || typeof entry !== "object" || Array.isArray(entry)) {
      throw new Error(`project at index ${index} must be an object`);
    }
    const candidate = entry as Record<string, unknown>;
    const { id, name, directory } = candidate;
    if (typeof id !== "string" || !projectIDPattern.test(id)) {
      throw new Error(`project at index ${index} has an invalid id`);
    }
    if (ids.has(id)) throw new Error(`duplicate project id: ${id}`);
    if (typeof name !== "string" || !name.trim() || name.length > 100) {
      throw new Error(`project ${id} has an invalid name`);
    }
    if (
      typeof directory !== "string" ||
      !path.isAbsolute(directory) ||
      directory.includes("\0")
    ) {
      throw new Error(`project ${id} directory must be an absolute path`);
    }

    let canonicalDirectory: string;
    try {
      canonicalDirectory = resolveDirectory(directory);
      if (!isDirectory(canonicalDirectory)) throw new Error("not a directory");
    } catch {
      throw new Error(`project ${id} directory does not exist or is not a directory`);
    }

    ids.add(id);
    return { id, name: name.trim(), directory: canonicalDirectory };
  });
}
