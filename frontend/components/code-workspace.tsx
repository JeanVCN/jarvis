"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
  createJarvisSession,
  getJarvisSession,
  getJarvisSessions,
  getOpenCodeProjects,
  getOpenCodeSessions,
  getOpenCodeStatus,
  interruptOpenCodeSession,
  replyOpenCodePermission,
  sendOpenCodePrompt,
  type JarvisSession,
  type JarvisSessionSnapshot,
  type OpenCodeOperation,
  type OpenCodeProject,
  type OpenCodeSession,
  type OpenCodeStatus,
} from "@/services/opencode";

function newOperationID(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

function operationIsActive(operation: OpenCodeOperation): boolean {
  return ["submitting", "accepted", "running", "uncertain"].includes(operation.state);
}

function operationLabel(state: string): string {
  return ({
    submitting: "Enviando…",
    accepted: "Aceito pelo OpenCode",
    running: "Em execução",
    completed: "Concluído",
    failed: "Falhou",
    uncertain: "Resultado incerto — reconcilie antes de qualquer novo envio",
  } as Record<string, string>)[state] ?? state;
}

export function CodeWorkspace() {
  const [status, setStatus] = useState<OpenCodeStatus | null>(null);
  const [projects, setProjects] = useState<OpenCodeProject[]>([]);
  const [selectedProject, setSelectedProject] = useState("");
  const [sessions, setSessions] = useState<JarvisSession[]>([]);
  const [recentSessions, setRecentSessions] = useState<OpenCodeSession[]>([]);
  const [selectedSession, setSelectedSession] = useState("");
  const [snapshot, setSnapshot] = useState<JarvisSessionSnapshot | null>(null);
  const [prompt, setPrompt] = useState("");
  const [loading, setLoading] = useState(true);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const activeOperation = useMemo(
    () => snapshot?.operations.find((operation) => operation.kind !== "create_session" && operationIsActive(operation)),
    [snapshot],
  );
  const hasActiveOperation = Boolean(activeOperation);

  const refreshSessions = useCallback(async (projectID: string) => {
    const [items, recent] = await Promise.all([getJarvisSessions(projectID), getOpenCodeSessions(projectID)]);
    setSessions(items);
    setRecentSessions(recent);
    setSelectedSession((current) => (items.some((item) => item.id === current) ? current : items[0]?.id ?? ""));
  }, []);

  useEffect(() => {
    let active = true;
    void Promise.all([getOpenCodeStatus(), getOpenCodeProjects()])
      .then(async ([service, allowedProjects]) => {
        if (!active) return;
        setStatus(service);
        setProjects(allowedProjects);
        setSelectedProject(allowedProjects[0]?.id ?? "");
        if (allowedProjects.length > 0) await refreshSessions(allowedProjects[0].id);
        setLoading(false);
      })
      .catch(() => {
        if (active) {
          setError("Não foi possível consultar o OpenCode local.");
          setLoading(false);
        }
      });
    return () => { active = false; };
  }, [refreshSessions]);

  const refreshSelected = useCallback(async () => {
    if (!selectedSession) return;
    const current = await getJarvisSession(selectedSession);
    setSnapshot(current);
    setSessions((items) => items.map((item) => item.id === current.session.id ? current.session : item));
    return current;
  }, [selectedSession]);

  useEffect(() => {
    if (!selectedSession) return;
    let active = true;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const poll = async () => {
      try {
        const current = await getJarvisSession(selectedSession);
        if (!active) return;
        setSnapshot(current);
        setSessions((items) => items.map((item) => item.id === current.session.id ? current.session : item));
        const hasRunning = current.operations.some((operation) => ["submitting", "accepted", "running"].includes(operation.state));
        if (hasRunning) timer = setTimeout(poll, 2000);
      } catch {
        if (active) setError("Não foi possível atualizar a sessão. O estado salvo continua disponível.");
      }
    };
    void poll();
    return () => { active = false; if (timer) clearTimeout(timer); };
  }, [selectedSession, hasActiveOperation]);

  async function createSession() {
    if (!selectedProject || !status?.write_enabled || working) return;
    const operationID = newOperationID();
    setWorking(true);
    setError("");
    setNotice("Criando sessão com permissões conservadoras. O OpenCode poderá pedir autorização para ferramentas.");
    setSelectedSession(operationID);
    try {
      await createJarvisSession(selectedProject, operationID);
      await refreshSessions(selectedProject);
      setSelectedSession(operationID);
      setNotice("Sessão criada.");
    } catch {
      setError("Não foi possível confirmar a criação. A sessão ficou marcada para reconciliação; não criamos outra automaticamente.");
      await refreshSessions(selectedProject).catch(() => undefined);
      setSelectedSession(operationID);
    } finally {
      setWorking(false);
    }
  }

  async function submitPrompt(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedSession || !prompt.trim() || !status?.write_enabled || activeOperation || working) return;
    const operationID = newOperationID();
    const sentText = prompt.trim();
    setWorking(true);
    setError("");
    setNotice("");
    try {
      await sendOpenCodePrompt(selectedSession, operationID, sentText);
      setPrompt("");
      setNotice("O OpenCode aceitou a tarefa. A conclusão só será indicada após confirmação do estado da sessão.");
      await refreshSelected();
    } catch {
      setError("Não foi possível confirmar o envio. O resultado foi salvo como incerto; reconcilie a sessão e não reenvie a tarefa.");
      await refreshSelected().catch(() => undefined);
    } finally {
      setWorking(false);
    }
  }

  async function reconcile() {
    setWorking(true);
    setError("");
    try {
      await refreshSelected();
      setNotice("Estado consultado novamente no OpenCode.");
    } catch {
      setError("Reconciliação indisponível. A tarefa permanece sem reenvio automático.");
    } finally {
      setWorking(false);
    }
  }

  async function replyPermission(requestID: string, decision: "once" | "reject") {
    if (!selectedSession || !status?.write_enabled || working) return;
    setWorking(true);
    setError("");
    try {
      await replyOpenCodePermission(selectedSession, requestID, newOperationID(), decision);
      setNotice(decision === "once" ? "Permissão concedida somente para esta ação." : "Pedido de permissão rejeitado.");
      await refreshSelected();
    } catch {
      setError("Não foi possível confirmar a decisão. Reconcilie a sessão antes de tentar novamente.");
      await refreshSelected().catch(() => undefined);
    } finally {
      setWorking(false);
    }
  }

  async function interruptSession() {
    if (!selectedSession || !status?.write_enabled || working) return;
    setWorking(true);
    setError("");
    setNotice("");
    try {
      const result = await interruptOpenCodeSession(selectedSession, newOperationID());
      setNotice(result.notice === "interrupt_requested_not_undo"
        ? "Interrupção solicitada. Isso não desfaz alterações já feitas pela tarefa."
        : "Interrupção solicitada ao OpenCode.");
      await refreshSelected();
    } catch {
      setError("Não foi possível confirmar a interrupção. Reconcilie a sessão; a tarefa pode ter continuado ou parado.");
      await refreshSelected().catch(() => undefined);
    } finally {
      setWorking(false);
    }
  }

  return (
    <main className="min-h-screen bg-slate-950 px-4 py-6 text-slate-100 sm:px-8">
      <div className="mx-auto max-w-4xl">
        <Link href="/" className="text-sm text-cyan-300 hover:text-cyan-200">← Jarvis</Link>
        <header className="mb-8 mt-5">
          <p className="text-xs uppercase tracking-[0.3em] text-cyan-400">Jarvis + OpenCode</p>
          <h1 className="mt-2 text-3xl font-semibold">Espaço de código</h1>
          <p className="mt-2 max-w-2xl text-sm text-slate-400">
            O Jarvis registra tarefas e sessões. O OpenCode executa no computador autorizado. Cada envio é único e não é repetido automaticamente se a conexão cair.
          </p>
        </header>

        <section className="rounded-xl border border-slate-800 bg-slate-900/70 p-4 sm:p-6">
          <h2 className="text-lg font-medium">Conexão</h2>
          <p className="mt-2 text-sm text-slate-300" role="status">
            {status?.status === "connected"
              ? `OpenCode conectado${status.version ? ` · versão ${status.version}` : ""}`
              : error || "Verificando serviço local…"}
          </p>
          <p className="mt-1 text-xs text-slate-500">
            {status?.write_enabled ? "Sessões, prompts e interrupção habilitados pelo servidor." : "Modo somente leitura. O servidor não habilitou envios."}
          </p>
        </section>

        <section className="mt-5 grid gap-5 rounded-xl border border-slate-800 bg-slate-900/70 p-4 sm:grid-cols-2 sm:p-6">
          <label className="block text-sm text-slate-300">
            Projeto permitido
            <select
              value={selectedProject}
              onChange={(event) => {
                const projectID = event.target.value;
                setLoading(true);
                setSelectedProject(projectID);
                setSelectedSession("");
                setSnapshot(null);
                void refreshSessions(projectID)
                  .catch(() => setError("Não foi possível carregar as sessões do projeto."))
                  .finally(() => setLoading(false));
              }}
              className="mt-2 block w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-3 text-base"
            >
              {projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}
            </select>
          </label>
          <div className="flex items-end">
            <button
              type="button"
              onClick={() => void createSession()}
              disabled={!status?.write_enabled || !selectedProject || working}
              className="w-full rounded-lg bg-cyan-700 px-4 py-3 font-medium text-white hover:bg-cyan-600 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {working && !selectedSession ? "Preparando…" : "Nova sessão"}
            </button>
          </div>
        </section>

        {sessions.length > 0 ? (
          <section className="mt-5 rounded-xl border border-slate-800 bg-slate-900/70 p-4 sm:p-6">
            <label className="block text-sm text-slate-300">
              Sessão Jarvis
              <select
                value={selectedSession}
                onChange={(event) => { setSelectedSession(event.target.value); setSnapshot(null); setError(""); }}
                className="mt-2 block w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-3 text-base"
              >
                {sessions.map((session) => (
                  <option key={session.id} value={session.id}>
                    {session.title} · {session.state === "ready" ? "pronta" : session.state === "uncertain" ? "incerta" : session.state}
                  </option>
                ))}
              </select>
            </label>
          </section>
        ) : null}

        {selectedSession ? (
          <section className="mt-5 rounded-xl border border-slate-800 bg-slate-900/70 p-4 sm:p-6">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h2 className="text-lg font-medium">{snapshot?.session.title ?? "Sessão"}</h2>
                <p className="mt-1 text-xs text-slate-400">
                  {snapshot?.detail?.session.state === "running" ? "OpenCode está trabalhando" : snapshot?.detail?.session.outcome ?? snapshot?.session.state ?? "Carregando estado…"}
                </p>
              </div>
              {snapshot?.detail?.session.state === "running" ? (
                <button
                  type="button"
                  onClick={() => void interruptSession()}
                  disabled={!status?.write_enabled || working}
                  className="rounded-lg border border-amber-700 px-3 py-2 text-sm text-amber-200 hover:bg-amber-950 disabled:opacity-50"
                >
                  Interromper tarefa
                </button>
              ) : null}
              {snapshot?.operations.some((operation) => operation.state === "uncertain") ? (
                <button type="button" onClick={() => void reconcile()} disabled={working} className="rounded-lg border border-amber-700 px-3 py-2 text-sm text-amber-200 disabled:opacity-50">
                  Reconciliar sem reenviar
                </button>
              ) : null}
            </div>

            {snapshot?.detail ? (
              <div className="mt-5 space-y-4">
                {snapshot.detail.messages.map((message) => (
                  <article key={message.id} className="rounded-lg border border-slate-800 bg-slate-950/70 p-3">
                    <p className="mb-1 text-xs uppercase text-cyan-400">{message.role === "assistant" ? "OpenCode" : "Você"}</p>
                    <p className="whitespace-pre-wrap break-words text-sm text-slate-200">{message.content}</p>
                  </article>
                ))}
                {snapshot.detail.permissions.length > 0 ? (
                  <div className="rounded-lg border border-amber-800 bg-amber-950/30 p-3 text-sm text-amber-100">
                    <p className="font-medium">Aguardando aprovação no OpenCode</p>
                    {snapshot.detail.permissions.map((permission) => (
                      <div key={permission.id} className="mt-3 border-t border-amber-900/60 pt-3">
                        <p>{permission.action}: {permission.message || permission.resources.join(", ")}</p>
                        {permission.resources.length > 0 ? <p className="mt-1 break-words text-xs text-amber-200/80">{permission.resources.join(", ")}</p> : null}
                        <div className="mt-2 flex flex-wrap gap-2">
                          <button type="button" onClick={() => void replyPermission(permission.id, "once")} disabled={!status?.write_enabled || working} className="rounded border border-amber-500 px-3 py-2 text-xs font-medium disabled:opacity-50">Permitir uma vez</button>
                          <button type="button" onClick={() => void replyPermission(permission.id, "reject")} disabled={!status?.write_enabled || working} className="rounded border border-slate-600 px-3 py-2 text-xs disabled:opacity-50">Rejeitar</button>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : null}
                {snapshot.detail.diff.length > 0 ? (
                  <details className="rounded-lg border border-slate-800 p-3">
                    <summary className="cursor-pointer text-sm font-medium">Alterações ({snapshot.detail.diff.length} arquivos)</summary>
                    <div className="mt-3 space-y-3">
                      {snapshot.detail.diff.map((file) => (
                        <article key={file.file} className="overflow-hidden rounded border border-slate-800">
                          <p className="bg-slate-950 px-3 py-2 text-xs text-cyan-200">{file.status} · {file.file} · +{file.additions} −{file.deletions}{file.truncated ? " · diff truncado" : ""}</p>
                          <pre className="max-h-96 overflow-auto p-3 text-xs text-slate-300">{file.patch}</pre>
                        </article>
                      ))}
                    </div>
                  </details>
                ) : null}
              </div>
            ) : snapshot?.session.state === "uncertain" ? (
              <p className="mt-4 text-sm text-amber-200">Não foi possível confirmar a criação da sessão. Reconciliar consulta o OpenCode sem criar outra sessão.</p>
            ) : null}

            <form onSubmit={(event) => void submitPrompt(event)} className="mt-5 space-y-3">
              <label htmlFor="opencode-prompt" className="block text-sm text-slate-300">Tarefa para o OpenCode</label>
              <textarea
                id="opencode-prompt"
                value={prompt}
                onChange={(event) => setPrompt(event.target.value)}
                maxLength={8000}
                rows={4}
                disabled={!status?.write_enabled || snapshot?.session.state !== "ready" || Boolean(activeOperation) || working}
                placeholder="Descreva a tarefa…"
                className="block w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-3 text-base disabled:opacity-60"
              />
              <button
                type="submit"
                disabled={!status?.write_enabled || !prompt.trim() || snapshot?.session.state !== "ready" || Boolean(activeOperation) || working}
                className="rounded-lg bg-cyan-700 px-4 py-3 font-medium text-white hover:bg-cyan-600 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {working ? "Enviando…" : activeOperation ? operationLabel(activeOperation.state) : "Enviar tarefa uma vez"}
              </button>
            </form>
            {snapshot?.operations.length ? (
              <ul className="mt-5 space-y-2 border-t border-slate-800 pt-4">
                {snapshot.operations.filter((operation) => operation.kind === "prompt").map((operation) => (
                  <li key={operation.id} className="text-sm">
                    <span className="font-medium">{operationLabel(operation.state)}</span>
                    {operation.input ? <p className="mt-1 whitespace-pre-wrap break-words text-slate-400">{operation.input}</p> : null}
                  </li>
                ))}
              </ul>
            ) : null}
          </section>
        ) : null}

        {recentSessions.length > 0 ? (
          <details className="mt-5 rounded-xl border border-slate-800 bg-slate-900/70 p-4 sm:p-6">
            <summary className="cursor-pointer text-sm font-medium">Sessões recentes do OpenCode (somente leitura)</summary>
            <ul className="mt-3 divide-y divide-slate-800">
              {recentSessions.map((session) => (
                <li key={session.id} className="py-3">
                  <p className="break-words font-medium">{session.title}</p>
                  <p className="mt-1 text-xs text-slate-400">{new Date(session.updated_at).toLocaleString("pt-BR")}{session.outcome ? ` · ${session.outcome}` : " · em andamento ou sem resultado"}</p>
                </li>
              ))}
            </ul>
          </details>
        ) : null}

        {loading ? <p className="mt-4 text-sm text-slate-500">Carregando…</p> : null}
        {notice ? <p role="status" className="mt-4 text-sm text-cyan-200">{notice}</p> : null}
        {error ? <p role="alert" className="mt-4 text-sm text-rose-300">{error}</p> : null}
      </div>
    </main>
  );
}
