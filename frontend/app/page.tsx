import { HealthPanel } from "@/components/health-panel";

export default function Home() {
  return (
    <main className="min-h-screen bg-surface text-slate-50">
      <section className="mx-auto flex min-h-screen w-full max-w-5xl flex-col justify-center gap-8 px-6 py-12">
        <div className="max-w-2xl">
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-accent">
            Fase 1
          </p>
          <h1 className="mt-4 text-4xl font-semibold tracking-normal sm:text-6xl">
            Jarvis
          </h1>
          <p className="mt-5 text-lg leading-8 text-slate-300">
            Fundacao inicial do assistente pessoal, pronta para evoluir com
            memoria, ferramentas, voz e agentes nas proximas fases.
          </p>
        </div>

        <HealthPanel />
      </section>
    </main>
  );
}
