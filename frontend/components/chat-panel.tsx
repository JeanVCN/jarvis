"use client";

import { FormEvent, useState } from "react";

import { streamChatMessage } from "@/services/chat";

export function ChatPanel() {
  const [message, setMessage] = useState("");
  const [response, setResponse] = useState("");
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const trimmedMessage = message.trim();
    if (!trimmedMessage || isLoading) {
      return;
    }

    setIsLoading(true);
    setError("");
    setResponse("");

    try {
      await streamChatMessage(trimmedMessage, (delta) => {
        setResponse((currentResponse) => currentResponse + delta);
      });
    } catch {
      setError("Nao foi possivel consultar o Ollama pelo backend.");
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <div className="w-full max-w-3xl rounded-lg border border-slate-700 bg-panel p-5 shadow-xl shadow-black/20">
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <label className="text-sm font-medium text-slate-200" htmlFor="message">
          Mensagem
        </label>
        <textarea
          id="message"
          value={message}
          onChange={(event) => setMessage(event.target.value)}
          className="min-h-32 w-full resize-y rounded-md border border-slate-700 bg-surface px-4 py-3 text-base leading-7 text-slate-50 outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/30"
          placeholder="Ola Jarvis"
        />
        <button
          type="submit"
          disabled={isLoading || message.trim() === ""}
          className="inline-flex min-h-11 w-fit items-center justify-center rounded-md bg-accent px-5 py-2 text-sm font-semibold text-slate-950 transition hover:bg-accent/90 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {isLoading ? "Enviando..." : "Enviar"}
        </button>
      </form>

      <div className="mt-6 min-h-32 rounded-md border border-slate-700 bg-surface p-4">
        <h2 className="text-sm font-medium text-slate-300">Resposta</h2>
        {error ? (
          <p className="mt-3 text-sm leading-6 text-red-300">{error}</p>
        ) : (
          <p className="mt-3 whitespace-pre-wrap text-base leading-7 text-slate-100">
            {response || "A resposta do Jarvis aparecera aqui."}
          </p>
        )}
      </div>
    </div>
  );
}
