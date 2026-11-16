"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import {
  getOpenCodeProjects,
  getOpenCodeSessions,
  getOpenCodeStatus,
  type OpenCodeProject,
  type OpenCodeSession,
  type OpenCodeStatus,
} from "@/services/opencode";

export function CodeWorkspace() {
  const [status, setStatus] = useState<OpenCodeStatus | null>(null);
  const [projects, setProjects] = useState<OpenCodeProject[]>([]);
  const [sessions, setSessions] = useState<OpenCodeSession[]>([]);
  const [loadedProject, setLoadedProject] = useState("");
  const [selectedProject, setSelectedProject] = useState("");
  const [error, setError] = useState("");
  const loadingSessions = Boolean(selectedProject) && loadedProject !== selectedProject;

  useEffect(() => {
    let active = true;
    void Promise.all([getOpenCodeStatus(), getOpenCodeProjects()])
      .then(([service, allowedProjects]) => {
        if (!active) return;
        setStatus(service);
        setProjects(allowedProjects);
        setSelectedProject(allowedProjects[0]?.id ?? "");
      })
      .catch((cause: unknown) => {
        if (!active) return;
        const message = cause instanceof Error ? cause.message : "opencode_unavailable";
        setError(
          message === "bridge_disabled"
            ? "A integração OpenCode está desativada."
            : "Não foi possível consultar o OpenCode local.",
        );
      });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!selectedProject) return;
    let active = true;
    void getOpenCodeSessions(selectedProject)
      .then((items) => {
        if (active) {
          setSessions(items);
          setLoadedProject(selectedProject);
        }
      })
      .catch(() => {
        if (active) {
          setError("Não foi possível carregar as sessões deste projeto.");
          setLoadedProject(selectedProject);
        }
      });
    return () => {
      active = false;
    };
  }, [selectedProject]);

  return (
    <main className="min-h-screen bg-slate-950 px-4 py-6 text-slate-100 sm:px-8">
      <div className="mx-auto max-w-3xl">
        <Link href="/" className="text-sm text-cyan-300 hover:text-cyan-200">← Jarvis</Link>
        <header className="mb-8 mt-5">
          <p className="text-xs uppercase tracking-[0.3em] text-cyan-400">Jarvis + OpenCode</p>
          <h1 className="mt-2 text-3xl font-semibold">Espaço de código</h1>
          <p className="mt-2 max-w-xl text-sm text-slate-400">
            Painel inicial somente leitura. Envio de tarefas e alterações ainda estão desativados.
          </p>
        </header>

        <section className="rounded-xl border border-slate-800 bg-slate-900/70 p-4 sm:p-6">
          <h2 className="text-lg font-medium">Conexão</h2>
          <p className="mt-2 text-sm text-slate-300" role="status">
            {status?.status === "connected"
              ? `OpenCode conectado${status.version ? ` · versão ${status.version}` : ""}`
              : error || "Verificando serviço local…"}
          </p>
        </section>

        <section className="mt-5 rounded-xl border border-slate-800 bg-slate-900/70 p-4 sm:p-6">
          <h2 className="text-lg font-medium">Projetos permitidos</h2>
          {projects.length === 0 ? (
            <p className="mt-3 text-sm text-slate-400">Nenhum projeto autorizado disponível.</p>
          ) : (
            <label className="mt-4 block text-sm text-slate-300">
              Projeto
              <select
                value={selectedProject}
                onChange={(event) => setSelectedProject(event.target.value)}
                className="mt-2 block w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-3 text-base"
              >
                {projects.map((project) => (
                  <option key={project.id} value={project.id}>{project.name}</option>
                ))}
              </select>
            </label>
          )}
        </section>

        <section className="mt-5 rounded-xl border border-slate-800 bg-slate-900/70 p-4 sm:p-6">
          <h2 className="text-lg font-medium">Sessões recentes</h2>
          {loadingSessions ? <p className="mt-3 text-sm text-slate-400">Carregando…</p> : null}
          {!loadingSessions && sessions.length === 0 ? (
            <p className="mt-3 text-sm text-slate-400">Nenhuma sessão encontrada para este projeto.</p>
          ) : null}
          <ul className="mt-3 divide-y divide-slate-800">
            {sessions.map((session) => (
              <li key={session.id} className="py-3">
                <p className="break-words font-medium">{session.title}</p>
                <p className="mt-1 text-xs text-slate-400">
                  {new Date(session.updated_at).toLocaleString("pt-BR")}
                  {session.outcome ? ` · ${session.outcome}` : " · sem resultado final"}
                </p>
              </li>
            ))}
          </ul>
        </section>

        {error && status?.status === "connected" ? (
          <p role="alert" className="mt-4 text-sm text-rose-300">{error}</p>
        ) : null}
      </div>
    </main>
  );
}
