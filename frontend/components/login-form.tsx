"use client";

import { useState, type FormEvent } from "react";

export function LoginForm({ next }: { next: string }) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const response = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ password }),
      });
      if (!response.ok) {
        const result = (await response.json().catch(() => ({}))) as { error?: string };
        setError(
          result.error === "too_many_attempts"
            ? "Muitas tentativas. Aguarde alguns minutos e tente novamente."
            : result.error === "auth_not_configured"
              ? "A autenticação ainda não foi configurada no servidor."
              : "Senha incorreta ou solicitação recusada.",
        );
        return;
      }
      window.location.assign(next);
    } catch {
      setError("Não foi possível conectar ao Jarvis.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <label className="block text-sm text-slate-300" htmlFor="password">
        Senha do Jarvis
      </label>
      <input
        id="password"
        autoComplete="current-password"
        type="password"
        required
        value={password}
        onChange={(event) => setPassword(event.target.value)}
        className="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-3 text-base outline-none focus:border-cyan-500"
      />
      {error && <p role="alert" className="text-sm text-rose-300">{error}</p>}
      <button
        type="submit"
        disabled={submitting}
        className="w-full rounded-lg bg-cyan-700 px-4 py-3 font-medium text-white hover:bg-cyan-600 disabled:opacity-50"
      >
        {submitting ? "Entrando…" : "Entrar"}
      </button>
    </form>
  );
}
