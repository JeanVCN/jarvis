"use client";

export function AuthControls() {
  async function logout() {
    await fetch("/api/auth/logout", { method: "POST" });
    window.location.assign("/login");
  }

  return (
    <button
      type="button"
      onClick={logout}
      className="fixed right-3 top-3 z-[100] rounded-md border border-slate-700 bg-slate-950/90 px-3 py-2 text-xs text-slate-200 shadow-lg"
    >
      Sair
    </button>
  );
}
