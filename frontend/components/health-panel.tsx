import { getHealth } from "@/services/health";

export async function HealthPanel() {
  const health = await getHealth();

  return (
    <div className="w-full max-w-xl rounded-lg border border-slate-700 bg-panel p-5 shadow-xl shadow-black/20">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h2 className="text-lg font-medium">Backend</h2>
          <p className="mt-1 text-sm text-slate-400">{health.service}</p>
        </div>
        <span className="rounded-full bg-accent px-3 py-1 text-sm font-semibold text-slate-950">
          {health.status}
        </span>
      </div>

      <dl className="mt-5 grid grid-cols-2 gap-4 text-sm">
        <div>
          <dt className="text-slate-400">Database</dt>
          <dd className="mt-1 font-medium text-slate-100">{health.database}</dd>
        </div>
        <div>
          <dt className="text-slate-400">API</dt>
          <dd className="mt-1 font-medium text-slate-100">{health.apiUrl}</dd>
        </div>
      </dl>
    </div>
  );
}
