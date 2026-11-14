import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
export async function GET() {
  try {
    const data = JSON.parse(await readFile(resolve(process.cwd(), "../.local/profile.json"), "utf8"));
    return Response.json({ stt: data.stt, tts: data.tts, voice: data.voice, turn: data.turn, provider: data.provider, model: data.model });
  } catch { return Response.json({}); }
}
