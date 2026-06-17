"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";

import { executeTool, listTools } from "@/services/tools";
import type { ToolMetadata } from "@/types/tools";

export function ToolsPanel() {
  const [tools, setTools] = useState<ToolMetadata[]>([]);
  const [selectedTool, setSelectedTool] = useState("");
  const [input, setInput] = useState("");
  const [result, setResult] = useState("");
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    let mounted = true;

    async function loadTools() {
      try {
        const availableTools = await listTools();
        if (!mounted) {
          return;
        }
        setTools(availableTools);
        setSelectedTool(availableTools[0]?.name ?? "");
      } catch {
        if (mounted) {
          setError("Nao foi possivel listar as ferramentas.");
        }
      }
    }

    void loadTools();

    return () => {
      mounted = false;
    };
  }, []);

  const selectedDescription = useMemo(() => {
    return tools.find((tool) => tool.name === selectedTool)?.description ?? "";
  }, [selectedTool, tools]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!selectedTool || isLoading) {
      return;
    }

    setIsLoading(true);
    setError("");
    setResult("");

    try {
      const response = await executeTool({
        tool: selectedTool,
        input,
      });
      setResult(response.result);
    } catch {
      setError("Nao foi possivel executar a ferramenta.");
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <div className="w-full max-w-3xl rounded-lg border border-slate-700 bg-panel p-5 shadow-xl shadow-black/20">
      <div className="flex flex-col gap-1">
        <h2 className="text-lg font-medium text-slate-100">Tools</h2>
      </div>

      <form onSubmit={handleSubmit} className="mt-5 flex flex-col gap-4">
        <div className="grid gap-2">
          <label className="text-sm font-medium text-slate-200" htmlFor="tool">
            Ferramenta
          </label>
          <select
            id="tool"
            value={selectedTool}
            onChange={(event) => setSelectedTool(event.target.value)}
            className="min-h-11 w-full rounded-md border border-slate-700 bg-surface px-4 py-2 text-base text-slate-50 outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/30"
          >
            {tools.map((tool) => (
              <option key={tool.name} value={tool.name}>
                {tool.name}
              </option>
            ))}
          </select>
          {selectedDescription ? (
            <p className="text-sm leading-6 text-slate-400">
              {selectedDescription}
            </p>
          ) : null}
        </div>

        <div className="grid gap-2">
          <label className="text-sm font-medium text-slate-200" htmlFor="input">
            Input
          </label>
          <textarea
            id="input"
            value={input}
            onChange={(event) => setInput(event.target.value)}
            className="min-h-24 w-full resize-y rounded-md border border-slate-700 bg-surface px-4 py-3 text-base leading-7 text-slate-50 outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/30"
            placeholder={selectedTool === "calculator" ? "25 * 4" : ""}
          />
        </div>

        <button
          type="submit"
          disabled={isLoading || selectedTool === ""}
          className="inline-flex min-h-11 w-fit items-center justify-center rounded-md bg-accent px-5 py-2 text-sm font-semibold text-slate-950 transition hover:bg-accent/90 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {isLoading ? "Executando..." : "Executar"}
        </button>
      </form>

      <div className="mt-6 min-h-24 rounded-md border border-slate-700 bg-surface p-4">
        <h3 className="text-sm font-medium text-slate-300">Resultado</h3>
        {error ? (
          <p className="mt-3 text-sm leading-6 text-red-300">{error}</p>
        ) : (
          <p className="mt-3 whitespace-pre-wrap text-base leading-7 text-slate-100">
            {result || "O resultado da ferramenta aparecera aqui."}
          </p>
        )}
      </div>
    </div>
  );
}
