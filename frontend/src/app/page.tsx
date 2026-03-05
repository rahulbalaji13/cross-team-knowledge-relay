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
        <MetricCard label="Backend URL" value={API_URL} />
      </section>

      <section className="grid flex-1 gap-8 md:grid-cols-[1.2fr_1fr]">
        <div className="rounded-2xl border border-black/10 bg-white p-6 shadow-sm dark:border-white/10 dark:bg-neutral-900">
          <h2 className="mb-4 text-xl font-semibold">Live bounty feed + matches</h2>
          <p className="mb-4 text-xs text-neutral-500">
          <button className="w-full rounded-lg bg-black px-4 py-2 text-sm font-semibold text-white dark:bg-white dark:text-black">Submit bounty</button>
        </form>
      </section>

      <footer className="mt-10 border-t border-black/10 pt-4 text-center text-xs font-bold text-white dark:border-white/10">
        © {new Date().getFullYear()} Cross-Team Knowledge Relay. Created by Rahul Balaji.
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
