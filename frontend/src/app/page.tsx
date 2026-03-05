"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";

type MatchedExpert = {
  expert_id: string;
  name: string;
  team: string;
  score: number;
  matched_skills: string[];
};

type Bounty = {
  id: string;
  title: string;
  description: string;
  skills: string[];
  amount: number;
  expiresAt: string;
  status: string;
  posterTeam: string;
  matchedExperts: MatchedExpert[];
};

const DEFAULT_API_URL = "https://cross-team-knowledge-relay.onrender.com";
const API_URL = (process.env.NEXT_PUBLIC_API_URL || DEFAULT_API_URL).replace(/\/$/, "");

export default function Home() {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [posterTeam, setPosterTeam] = useState("Commerce");
  const [skills, setSkills] = useState("Go, Neo4j, Distributed Systems");
  const [amount, setAmount] = useState(150);
  const [ttlSeconds, setTtlSeconds] = useState(3600);
  const [bounties, setBounties] = useState<Bounty[]>([]);
  const [status, setStatus] = useState("Checking backend connection...");
  const [connected, setConnected] = useState(false);
  const currentYear = new Date().getFullYear();

  const totalOpenValue = useMemo(
    () => bounties.reduce((total, item) => total + item.amount, 0),
    [bounties],
  );

  const totalMatches = useMemo(
    () => bounties.reduce((total, item) => total + (item.matchedExperts?.length || 0), 0),
    [bounties],
  );

  async function loadBounties() {
    try {
      const response = await fetch(`${API_URL}/api/v1/bounties`, { cache: "no-store" });
      if (!response.ok) {
        throw new Error(`Failed with status ${response.status}`);
      }
      const data = await response.json();
      setBounties(data.bounties || []);
      setConnected(true);
      setStatus("Backend connected. Matching engine output loaded.");
    } catch (error) {
      setConnected(false);
      setStatus(`Unable to reach backend (${API_URL}). ${(error as Error).message}`);
    }
  }

  useEffect(() => {
    loadBounties();
  }, []);

  async function submitBounty(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setStatus("Submitting bounty and running cross-team matching...");

    try {
      const response = await fetch(`${API_URL}/api/v1/bounties`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title,
          description,
          poster_team: posterTeam,
          skills: skills.split(",").map((s) => s.trim()).filter(Boolean),
          bounty_amount: amount,
          ttl_seconds: ttlSeconds,
        }),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || `Failed with status ${response.status}`);
      }

      const data = await response.json();
      setBounties((prev) => [data.bounty, ...prev.filter((b) => b.id !== data.bounty.id)]);
      setTitle("");
      setDescription("");
      setStatus(`Bounty accepted. ${data.bounty.matchedExperts?.length || 0} cross-team experts suggested.`);
      setConnected(true);
    } catch (error) {
      setStatus(`Submission failed against ${API_URL}: ${(error as Error).message}`);
    }
  }

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-6xl flex-col px-6 py-10">
      <header className="mb-8 rounded-2xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-neutral-900">
        <h1 className="text-3xl font-bold">Cross-Team Knowledge Relay</h1>
        <p className="mt-2 text-sm text-neutral-600 dark:text-neutral-300">
          Post technical blockers and get ranked experts outside your own team.
        </p>
        <p className={`mt-4 text-sm font-medium ${connected ? "text-green-600" : "text-amber-600"}`}>{status}</p>
      </header>

      <section className="mb-8 grid gap-4 md:grid-cols-4">
        <MetricCard label="Open bounties" value={String(bounties.length)} />
        <MetricCard label="Total reward value" value={`$${totalOpenValue}`} />
        <MetricCard label="Cross-team matches" value={String(totalMatches)} />
        <MetricCard label="Backend URL" value={API_URL} />
      </section>

      <section className="grid flex-1 gap-8 md:grid-cols-[1.2fr_1fr]">
        <div className="rounded-2xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-neutral-900">
          <h2 className="mb-4 text-xl font-semibold">Live bounty feed + matches</h2>
          <p className="mb-4 text-xs text-neutral-500">
            Match suggestions come from the backend expert directory and are scored from skill level, skill recency, and reputation.
            In this demo, expert names (for example Priya, Sara, and Maya) are seeded sample data rather than random users.
          </p>
          <div className="space-y-4">
            {bounties.length === 0 ? (
              <p className="text-sm text-neutral-500">No bounties yet. Create one to test matching.</p>
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
                  <p className="mt-2 text-xs text-neutral-500">Poster team: {bounty.posterTeam || "Unknown"}</p>
                  <p className="mt-1 text-xs text-neutral-500">Skills: {bounty.skills.join(", ")}</p>
                  <p className="mt-1 text-xs text-neutral-500">Expires: {new Date(bounty.expiresAt).toLocaleString()}</p>

                  <div className="mt-3 rounded-lg bg-black/[0.03] p-3 dark:bg-white/[0.03]">
                    <p className="text-xs font-semibold uppercase tracking-wide text-neutral-500">Top cross-team matches</p>
                    {bounty.matchedExperts?.length ? (
                      <ul className="mt-2 space-y-2 text-sm">
                        {bounty.matchedExperts.map((expert) => (
                          <li key={expert.expert_id} className="rounded-md border border-black/10 px-2 py-2 dark:border-white/10">
                            <div className="flex items-center justify-between">
                              <span className="font-medium">{expert.name} ({expert.team})</span>
                              <span className="text-xs">Score: {expert.score}</span>
                            </div>
                            <p className="mt-1 text-xs text-neutral-500">Matched skills: {expert.matched_skills.join(", ")}</p>
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p className="mt-2 text-xs text-neutral-500">No cross-team match found yet.</p>
                    )}
                  </div>
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
          <label className="mb-3 block text-sm">Poster Team
            <input required value={posterTeam} onChange={(e) => setPosterTeam(e.target.value)} className="mt-1 w-full rounded-lg border border-black/15 bg-transparent px-3 py-2" />
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

      <footer className="mt-10 border-t border-black/10 pt-4 text-center text-xs font-bold text-white dark:border-white/10">
        © {currentYear} Cross-Team Knowledge Relay. Created by Rahul Balaji.
      </footer>
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
