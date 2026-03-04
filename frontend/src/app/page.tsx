import Image from "next/image";

export default function Home() {
  return (
    <main className="mx-auto min-h-screen w-full max-w-5xl px-6 py-10">
      <header className="mb-8 rounded-2xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-neutral-900">
        <h1 className="text-3xl font-bold">Cross-Team Knowledge Relay</h1>
        <p className="mt-2 text-sm text-neutral-600 dark:text-neutral-300">
          Post technical blockers, match with experts outside your immediate team, and resolve issues faster.
        </p>
        <p className={`mt-4 text-sm font-medium ${connected ? "text-green-600" : "text-amber-600"}`}>{status}</p>
      </header>

      <section className="mb-8 grid gap-4 md:grid-cols-3">
        <MetricCard label="Open bounties" value={String(bounties.length)} />
        <MetricCard label="Total reward value" value={`$${totalOpenValue}`} />
        <MetricCard label="Backend URL" value={API_URL} />
      </section>

      <section className="grid gap-8 md:grid-cols-[1.2fr_1fr]">
        <div className="rounded-2xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-neutral-900">
          <h2 className="mb-4 text-xl font-semibold">Live bounty feed</h2>
          <div className="space-y-3">
            {bounties.length === 0 ? (
              <p className="text-sm text-neutral-500">No bounties yet. Create one to test end-to-end flow.</p>
            ) : (
              bounties.map((bounty) => (
                <article key={bounty.id} className="rounded-xl border border-black/10 p-4 dark:border-white/10">
                  <div className="flex items-start justify-between gap-3">
                    <h3 className="font-semibold">{bounty.title}</h3>
                    <span className="rounded-full bg-emerald-100 px-3 py-1 text-xs font-semibold text-emerald-700">
                      ${bounty.amount}
                    </span>
                  </div>
                  <p className="mt-2 text-sm text-neutral-600 dark:text-neutral-300">{bounty.description || "No description provided."}</p>
                  <p className="mt-2 text-xs text-neutral-500">Skills: {bounty.skills.join(", ")}</p>
                  <p className="mt-1 text-xs text-neutral-500">Expires: {new Date(bounty.expiresAt).toLocaleString()}</p>
                </article>
              ))
            )}
          </div>
        </div>

        <form onSubmit={submitBounty} className="rounded-2xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-neutral-900">
          <h2 className="mb-4 text-xl font-semibold">Create bounty</h2>
          <label className="mb-3 block text-sm">Title
            <input required value={title} onChange={(e) => setTitle(e.target.value)} className="mt-1 w-full rounded-lg border border-black/15 bg-transparent px-3 py-2" />
          </label>
          <label className="mb-3 block text-sm">Description
            <textarea value={description} onChange={(e) => setDescription(e.target.value)} className="mt-1 min-h-24 w-full rounded-lg border border-black/15 bg-transparent px-3 py-2" />
          </label>
          <label className="mb-3 block text-sm">Skills (comma separated)
            <input required value={skills} onChange={(e) => setSkills(e.target.value)} className="mt-1 w-full rounded-lg border border-black/15 bg-transparent px-3 py-2" />
          </label>
          <div className="mb-4 grid grid-cols-2 gap-3">
            <label className="block text-sm">Amount ($)
              <input type="number" min={1} required value={amount} onChange={(e) => setAmount(Number(e.target.value))} className="mt-1 w-full rounded-lg border border-black/15 bg-transparent px-3 py-2" />
            </label>
            <label className="block text-sm">TTL (seconds)
              <input type="number" min={60} required value={ttlSeconds} onChange={(e) => setTtlSeconds(Number(e.target.value))} className="mt-1 w-full rounded-lg border border-black/15 bg-transparent px-3 py-2" />
            </label>
          </div>
          <button className="w-full rounded-lg bg-black px-4 py-2 text-sm font-semibold text-white dark:bg-white dark:text-black">Submit bounty</button>
        </form>
      </section>
    </main>
  );
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-black/10 bg-white p-4 shadow-sm dark:border-white/10 dark:bg-neutral-900">
      <p className="text-xs uppercase tracking-wide text-neutral-500">{label}</p>
      <p className="mt-2 break-all text-lg font-semibold">{value}</p>
    </div>
  );
}
