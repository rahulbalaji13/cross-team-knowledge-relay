"use client";

import { useState } from "react";

export default function LockButton() {
  const [busy, setBusy] = useState(false);

  async function lockSite() {
    setBusy(true);
    try {
      await fetch("/api/access/logout", { method: "POST", cache: "no-store" });
      window.location.reload();
    } finally {
      setBusy(false);
    }
  }

  return (
    <button type="button" onClick={lockSite} disabled={busy} aria-label="Lock site and sign out" title="Lock site" className="inline-flex min-h-10 items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:opacity-60">
      <LockIcon />
      <span className="hidden sm:inline">Lock</span>
    </button>
  );
}

function LockIcon() {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className="size-4">
    <rect x="5" y="10" width="14" height="11" rx="2" stroke="currentColor" strokeWidth="1.7" />
    <path d="M8 10V7a4 4 0 1 1 8 0v3" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
  </svg>;
}
