import { LoginForm } from "@/components/login-form";

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ next?: string }>;
}) {
  const { next } = await searchParams;
  const safeNext =
    next?.startsWith("/") && !next.startsWith("//") && !next.includes("\\") ? next : "/";

  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-950 px-4 text-slate-100">
      <section className="w-full max-w-sm rounded-2xl border border-cyan-900/60 bg-slate-900/80 p-6 shadow-2xl shadow-cyan-950/30">
        <p className="mb-2 text-xs uppercase tracking-[0.3em] text-cyan-400">Jarvis</p>
        <h1 className="mb-6 text-2xl font-semibold">Entrar</h1>
        <LoginForm next={safeNext} />
      </section>
    </main>
  );
}
