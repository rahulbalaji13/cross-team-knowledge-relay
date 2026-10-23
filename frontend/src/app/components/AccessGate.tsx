"use client";

import { FormEvent, useState } from "react";

export default function AccessGate({ configured }: { configured: boolean }) {
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function unlock(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const response = await fetch("/api/access/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ password }),
        cache: "no-store",
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Could not unlock the site.");
      window.location.reload();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Could not unlock the site.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-[#f5f7fb] px-4 py-10 text-slate-900">
      <section className="w-full max-w-md rounded-3xl border border-slate-200 bg-white p-6 shadow-xl shadow-slate-900/[0.06] sm:p-8">
        <div className="flex size-12 items-center justify-center rounded-2xl bg-indigo-600 text-white shadow-lg shadow-indigo-200">
          <LockIcon />
        </div>
        <p className="mt-6 text-xs font-bold uppercase tracking-[0.18em] text-indigo-700">Private preview</p>
        <h1 className="mt-2 text-2xl font-bold tracking-tight">Knowledge Relay</h1>
        <p className="mt-2 text-sm leading-6 text-slate-500">Enter the site password to continue to the cross-team help board.</p>

        {!configured ? (
          <div role="alert" className="mt-6 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-950">
            Site access is not configured yet. The owner must add the access password and session secret in the hosting settings.
          </div>
        ) : (
          <form onSubmit={unlock} className="mt-6 space-y-4">
            <label className="block text-sm font-semibold text-slate-800">
              Site password
              <input
                type="password"
                autoComplete="current-password"
                required
                maxLength={1024}
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                className="form-control mt-2"
                placeholder="Enter your password"
              />
            </label>
            {error ? <p role="alert" className="rounded-lg bg-rose-50 px-3 py-2 text-sm text-rose-800">{error}</p> : null}
            <button type="submit" disabled={busy || !password} className="inline-flex min-h-12 w-full items-center justify-center gap-2 rounded-xl bg-indigo-600 px-4 py-3 text-sm font-semibold text-white transition hover:bg-indigo-700 disabled:cursor-wait disabled:opacity-60">
              {busy ? "Checking…" : "Unlock site"} <span aria-hidden="true">→</span>
            </button>
          </form>
        )}

        <p className="mt-5 border-t border-slate-100 pt-4 text-xs leading-5 text-slate-400">Your password is checked on the server. This preview uses a shared site password, not personal accounts.</p>
      </section>
    </main>
  );
}

function LockIcon() {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className="size-6">
    <rect x="5" y="10" width="14" height="11" rx="2" stroke="currentColor" strokeWidth="1.7" />
    <path d="M8 10V7a4 4 0 1 1 8 0v3m-4 4v3" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" />
  </svg>;
}
