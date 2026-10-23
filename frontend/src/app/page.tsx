"use client";

import { FormEvent, ReactNode, useCallback, useEffect, useMemo, useState } from "react";
import RelayAssistant from "./components/RelayAssistant";
import VoiceInputButton from "./components/VoiceInputButton";

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
type ConnectionState = "checking" | "online" | "offline";

export default function Home() {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [posterTeam, setPosterTeam] = useState("Commerce");
  const [skills, setSkills] = useState("Go, Neo4j, Distributed Systems");
  const [amount, setAmount] = useState(150);
  const [ttlSeconds, setTtlSeconds] = useState(3600);
  const [bounties, setBounties] = useState<Bounty[]>([]);
  const [connection, setConnection] = useState<ConnectionState>("checking");
  const [loadingFeed, setLoadingFeed] = useState(true);
  const [search, setSearch] = useState("");
  const [submitState, setSubmitState] = useState<"idle" | "submitting" | "success" | "error">("idle");
  const [submitMessage, setSubmitMessage] = useState("");
  const [theme, setTheme] = useState<"light" | "dark">("light");

  const openBounties = useMemo(
    () => bounties.filter((item) => item.status.toLowerCase() === "open"),
    [bounties],
  );
  const totalOpenValue = useMemo(
    () => openBounties.reduce((total, item) => total + item.amount, 0),
    [openBounties],
  );
  const totalMatches = useMemo(
    () => openBounties.reduce((total, item) => total + (item.matchedExperts?.length || 0), 0),
    [openBounties],
  );
  const filteredBounties = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return openBounties;
    return openBounties.filter((bounty) =>
      [bounty.title, bounty.description, bounty.posterTeam, ...bounty.skills].join(" ").toLowerCase().includes(query),
    );
  }, [openBounties, search]);

  const loadBounties = useCallback(async () => {
    setLoadingFeed(true);
    try {
      const response = await fetch(API_URL + "/api/v1/bounties", { cache: "no-store" });
      if (!response.ok) throw new Error("The feed returned status " + response.status);
      const data = await response.json();
      setBounties(Array.isArray(data.bounties) ? data.bounties : []);
      setConnection("online");
    } catch {
      setConnection("offline");
    } finally {
      setLoadingFeed(false);
    }
  }, []);

  useEffect(() => {
    const savedTheme = window.localStorage.getItem("relay-theme");
    const initialTheme = savedTheme === "light" || savedTheme === "dark"
      ? savedTheme
      : window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
    document.documentElement.dataset.theme = initialTheme;
    setTheme(initialTheme);
    void loadBounties();
  }, [loadBounties]);

  function toggleTheme() {
    const nextTheme = theme === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = nextTheme;
    window.localStorage.setItem("relay-theme", nextTheme);
    setTheme(nextTheme);
  }

  async function submitBounty(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const skillList = skills.split(",").map((skill) => skill.trim()).filter(Boolean);
    if (skillList.length > 20) {
      setSubmitState("error");
      setSubmitMessage("Add up to 20 skills, separated by commas.");
      return;
    }

    setSubmitState("submitting");
    setSubmitMessage("");
    try {
      const response = await fetch(API_URL + "/api/v1/bounties", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: title.trim(),
          description: description.trim(),
          poster_team: posterTeam.trim(),
          skills: skillList,
          bounty_amount: amount,
          ttl_seconds: ttlSeconds,
        }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Could not submit your request.");
      setBounties((current) => [data.bounty, ...current.filter((item) => item.id !== data.bounty.id)]);
      setTitle("");
      setDescription("");
      setSubmitState("success");
      setSubmitMessage("Request posted. " + (data.bounty.matchedExperts?.length || 0) + " cross-team experts suggested.");
      setConnection("online");
    } catch (error) {
      setSubmitState("error");
      setSubmitMessage(error instanceof Error ? error.message : "Could not submit your request.");
    }
  }

  return (
    <main className="min-h-screen bg-[#f5f7fb] text-slate-900">
      <header className="border-b border-slate-200/80 bg-white/90">
        <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-4 py-4 sm:px-6 lg:px-8">
          <a href="#" className="flex min-w-0 items-center gap-3 rounded-xl focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-indigo-600">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-indigo-600 text-white shadow-sm shadow-indigo-200"><RelayMark /></span>
            <span className="min-w-0">
              <span className="block truncate text-sm font-bold tracking-tight sm:text-base">Knowledge Relay</span>
              <span className="hidden text-xs text-slate-500 sm:block">Cross-team help, in one place</span>
            </span>
          </a>
          <nav aria-label="Main navigation" className="flex items-center gap-2 sm:gap-3">
            <a href="#requests" className="hidden rounded-lg px-3 py-2 text-sm font-medium text-slate-600 transition hover:bg-slate-100 hover:text-slate-950 sm:inline-flex">Browse requests</a>
            <a href="#create" className="inline-flex min-h-10 items-center justify-center gap-2 rounded-xl bg-indigo-600 px-3.5 py-2 text-sm font-semibold text-white shadow-sm transition hover:bg-indigo-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 sm:px-4">
              <span className="text-lg leading-none" aria-hidden="true">+</span><span>Ask for help</span>
            </a>
            <button type="button" onClick={toggleTheme} aria-pressed={theme === "dark"} aria-label={theme === "dark" ? "Switch to light mode" : "Switch to dark mode"} title={theme === "dark" ? "Switch to light mode" : "Switch to dark mode"} className="inline-flex min-h-10 items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600">
              <ThemeIcon theme={theme} />
              <span>{theme === "dark" ? "Light" : "Dark"}</span>
            </button>
          </nav>
        </div>
      </header>

      <div className="mx-auto max-w-7xl px-4 pb-10 pt-6 sm:px-6 sm:pb-14 sm:pt-8 lg:px-8">
        <section className="relative isolate overflow-hidden rounded-[1.75rem] bg-[#111827] px-5 py-8 text-white shadow-xl shadow-slate-900/10 sm:px-9 sm:py-10 lg:px-12 lg:py-12">
          <div className="pointer-events-none absolute -right-16 -top-28 -z-10 size-80 rounded-full bg-indigo-500/30 blur-3xl" />
          <div className="pointer-events-none absolute bottom-[-8rem] right-[24%] -z-10 size-72 rounded-full bg-cyan-400/15 blur-3xl" />
          <div className="grid gap-8 lg:grid-cols-[1.25fr_0.75fr] lg:items-center lg:gap-12">
            <div>
              <div className="mb-5 inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/[0.07] px-3 py-1.5 text-xs font-medium text-indigo-100">
                <span className={"size-2 rounded-full " + (connection === "online" ? "bg-emerald-400" : connection === "offline" ? "bg-rose-400" : "animate-pulse bg-amber-300")} />
                {connection === "online" ? "Relay is connected" : connection === "offline" ? "Relay is reconnecting" : "Connecting to relay"}
              </div>
              <p className="mb-3 text-xs font-semibold uppercase tracking-[0.2em] text-indigo-200">Find the right people, faster</p>
              <h1 className="max-w-2xl text-3xl font-semibold leading-tight tracking-tight sm:text-4xl lg:text-5xl">Get unstuck with knowledge from across your team.</h1>
              <p className="mt-4 max-w-xl text-sm leading-6 text-slate-300 sm:text-base sm:leading-7">Share a technical blocker. The relay finds teammates with relevant skills and experience so useful knowledge can move where it is needed.</p>
              <div className="mt-7 flex flex-col gap-3 sm:flex-row">
                <a href="#create" className="inline-flex min-h-11 items-center justify-center gap-2 rounded-xl bg-white px-5 py-3 text-sm font-semibold text-slate-900 transition hover:bg-indigo-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white">Post a request <span aria-hidden="true">→</span></a>
                <a href="#requests" className="inline-flex min-h-11 items-center justify-center rounded-xl border border-white/20 px-5 py-3 text-sm font-semibold text-white transition hover:bg-white/10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white">Explore the feed</a>
              </div>
            </div>
            <div className="rounded-2xl border border-white/10 bg-white/[0.07] p-5 backdrop-blur-sm sm:p-6">
              <div className="flex items-center justify-between">
                <p className="text-sm font-semibold text-white">How the relay works</p>
                <span className="rounded-full bg-indigo-400/15 px-2.5 py-1 text-[11px] font-semibold text-indigo-100">3 steps</span>
              </div>
              <ol className="mt-5 space-y-4">
                <Step number="01" title="Describe the blocker" text="Add the context and skills you need." />
                <Step number="02" title="Find relevant expertise" text="See suggested teammates outside your team." />
                <Step number="03" title="Share what you know" text="Connect directly and move the work forward." />
              </ol>
            </div>
          </div>
        </section>

        <section aria-label="Request overview" className="my-6 grid grid-cols-1 gap-3 sm:my-7 sm:grid-cols-3 sm:gap-4">
          <MetricCard icon="↗" label="Open requests" value={String(openBounties.length)} detail="Waiting for a helping hand" />
          <MetricCard icon="$" label="Reward pool" value={"$" + totalOpenValue.toLocaleString()} detail="Illustrative demo amounts" />
          <MetricCard icon="✳" label="Expert suggestions" value={String(totalMatches)} detail="Matched across teams" />
        </section>

        <section className="grid items-start gap-5 lg:grid-cols-[minmax(0,1.4fr)_minmax(19rem,0.8fr)] lg:gap-7">
          <div id="requests" className="min-w-0 scroll-mt-6 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm shadow-slate-900/[0.02] sm:p-6">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <h2 className="text-lg font-bold tracking-tight sm:text-xl">Open requests</h2>
                  <span className="rounded-full bg-indigo-50 px-2.5 py-1 text-xs font-semibold text-indigo-700">{openBounties.length}</span>
                </div>
                <p className="mt-1 text-sm text-slate-500">Browse blockers and the people suggested to help.</p>
              </div>
              <button type="button" onClick={() => void loadBounties()} disabled={loadingFeed} className="inline-flex min-h-10 items-center justify-center gap-2 self-start rounded-lg border border-slate-200 px-3 py-2 text-sm font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-wait disabled:opacity-60 sm:self-auto">
                <span aria-hidden="true" className={loadingFeed ? "animate-spin" : ""}>↻</span>Refresh
              </button>
            </div>
            <label className="mt-5 flex min-h-11 items-center gap-2 rounded-xl border border-slate-200 bg-slate-50 px-3.5 transition focus-within:border-indigo-400 focus-within:bg-white focus-within:ring-4 focus-within:ring-indigo-100">
              <SearchIcon /><span className="sr-only">Search requests</span>
              <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search title, team, or skill..." className="w-full bg-transparent py-2 text-sm text-slate-900 outline-none placeholder:text-slate-400" />
              {search && <button type="button" onClick={() => setSearch("")} aria-label="Clear search" className="rounded p-1 text-slate-400 hover:text-slate-700">×</button>}
            </label>

            <div aria-live="polite" className="mt-5 space-y-4">
              {loadingFeed && bounties.length === 0 ? (
                <div className="space-y-3" aria-label="Loading requests"><div className="h-40 animate-pulse rounded-xl bg-slate-100" /><div className="h-40 animate-pulse rounded-xl bg-slate-100" /></div>
              ) : connection === "offline" && bounties.length === 0 ? (
                <div className="rounded-xl border border-amber-200 bg-amber-50 p-5">
                  <p className="font-semibold text-amber-950">We couldn’t load the feed.</p>
                  <p className="mt-1 text-sm leading-6 text-amber-900/80">The demo backend may be waking up. Try refreshing in a few moments.</p>
                  <button type="button" onClick={() => void loadBounties()} className="mt-4 min-h-10 rounded-lg bg-amber-900 px-4 py-2 text-sm font-semibold text-white transition hover:bg-amber-800">Try again</button>
                </div>
              ) : filteredBounties.length === 0 ? (
                <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-5 py-10 text-center">
                  <span className="mx-auto flex size-11 items-center justify-center rounded-full bg-white text-xl text-indigo-600 shadow-sm" aria-hidden="true">⌕</span>
                  <p className="mt-3 font-semibold text-slate-900">{search ? "No matching requests" : "A clear board is a fresh start"}</p>
                  <p className="mx-auto mt-1 max-w-sm text-sm leading-6 text-slate-500">{search ? "Try another keyword or clear your search." : "Post a blocker and invite another team to share what they know."}</p>
                  {search ? <button type="button" onClick={() => setSearch("")} className="mt-3 text-sm font-semibold text-indigo-700 hover:text-indigo-900">Clear search</button> : null}
                </div>
              ) : (
                filteredBounties.map((bounty) => <BountyCard key={bounty.id} bounty={bounty} />)
              )}
            </div>
            {connection === "online" && <p className="mt-5 flex items-center gap-2 border-t border-slate-100 pt-4 text-xs text-slate-500"><span className="size-1.5 rounded-full bg-emerald-500" />Feed is connected to the demo service</p>}
          </div>

          <form id="create" onSubmit={submitBounty} className="scroll-mt-6 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm shadow-slate-900/[0.02] sm:p-6 lg:sticky lg:top-6">
            <div className="mb-5">
              <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.15em] text-indigo-700"><span className="flex size-6 items-center justify-center rounded-md bg-indigo-50 text-base leading-none" aria-hidden="true">+</span>Help request</div>
              <h2 className="mt-3 text-xl font-bold tracking-tight">What are you working through?</h2>
              <p className="mt-1 text-sm leading-6 text-slate-500">Give teammates enough context to know how they can help.</p>
            </div>
            <div className="space-y-4">
              <Field label="Request title" hint="Be clear and specific">
                <div className="flex items-center gap-2">
                  <input required minLength={3} maxLength={200} value={title} onChange={(event) => setTitle(event.target.value)} placeholder="e.g. Need help reducing API latency" className="form-control min-w-0 flex-1" />
                  <VoiceInputButton fieldLabel="request title" onTranscript={(transcript) => setTitle((current) => appendVoiceTranscript(current, transcript))} />
                </div>
              </Field>
              <Field label="What’s the blocker?" hint="A little context goes a long way">
                <textarea maxLength={4000} rows={4} value={description} onChange={(event) => setDescription(event.target.value)} placeholder="What have you tried? What would a useful answer look like?" className="form-control min-h-28 resize-y" />
                <div className="mt-2 flex items-start justify-between gap-2">
                  <span className="text-xs text-slate-400">{description.length}/4000</span>
                  <VoiceInputButton fieldLabel="blocker description" onTranscript={(transcript) => setDescription((current) => appendVoiceTranscript(current, transcript))} />
                </div>
              </Field>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <Field label="Your team">
                  <div className="flex items-center gap-2">
                    <input required maxLength={80} value={posterTeam} onChange={(event) => setPosterTeam(event.target.value)} placeholder="e.g. Commerce" className="form-control min-w-0 flex-1" />
                    <VoiceInputButton fieldLabel="team name" onTranscript={(transcript) => setPosterTeam(transcript.trim())} />
                  </div>
                </Field>
                <Field label="Reward amount" hint="Demo value only">
                  <div className="relative"><span className="pointer-events-none absolute inset-y-0 left-3 flex items-center text-sm text-slate-400">$</span><input type="number" min={1} max={100000} required value={amount} onChange={(event) => setAmount(Number(event.target.value))} className="form-control pl-7" /></div>
                </Field>
              </div>
              <Field label="Skills you need" hint="Separate skills with commas or say “and” (up to 20)">
                <div className="flex items-center gap-2">
                  <input required value={skills} onChange={(event) => setSkills(event.target.value)} placeholder="Go, API design, PostgreSQL" className="form-control min-w-0 flex-1" />
                  <VoiceInputButton fieldLabel="required skills" onTranscript={(transcript) => setSkills((current) => appendSkillTranscript(current, transcript))} />
                </div>
              </Field>
              <Field label="Request stays open">
                <select value={ttlSeconds} onChange={(event) => setTtlSeconds(Number(event.target.value))} className="form-control">
                  <option value={3600}>1 hour</option><option value={86400}>1 day</option><option value={604800}>7 days</option><option value={2592000}>30 days</option>
                </select>
              </Field>
            </div>
            <p className="mt-4 rounded-lg bg-slate-50 px-3 py-2 text-xs leading-5 text-slate-500">Voice input uses your browser’s speech recognition; some browsers may process audio through an online service.</p>
            {submitMessage && <div role={submitState === "error" ? "alert" : "status"} className={"mt-4 rounded-xl px-3.5 py-3 text-sm leading-5 " + (submitState === "error" ? "bg-rose-50 text-rose-800" : "bg-emerald-50 text-emerald-800")}>{submitMessage}</div>}
            <button type="submit" disabled={submitState === "submitting"} className="mt-5 inline-flex min-h-12 w-full items-center justify-center gap-2 rounded-xl bg-indigo-600 px-4 py-3 text-sm font-semibold text-white shadow-sm shadow-indigo-200 transition hover:bg-indigo-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:cursor-wait disabled:opacity-70">
              {submitState === "submitting" ? <><span className="size-4 animate-spin rounded-full border-2 border-white/40 border-t-white" />Finding teammates…</> : <>Post request <span aria-hidden="true">→</span></>}
            </button>
            <p className="mt-3 text-center text-xs leading-5 text-slate-400">No payment is processed. This is a portfolio demo.</p>
          </form>
        </section>

        <footer className="mt-9 flex flex-col gap-2 border-t border-slate-200 pt-5 text-center text-xs text-slate-500 sm:flex-row sm:items-center sm:justify-between sm:text-left">
          <p>© {new Date().getFullYear()} Cross-Team Knowledge Relay</p>
          <p>Share knowledge. Unblock work. Move together.</p>
        </footer>
      </div>
      <RelayAssistant />
    </main>
  );
}

function appendVoiceTranscript(current: string, transcript: string) {
  const spoken = transcript.trim();
  if (!spoken) return current;
  return current.trim() ? current.trimEnd() + " " + spoken : spoken;
}

function appendSkillTranscript(current: string, transcript: string) {
  const spoken = transcript.trim().replace(/\s+(?:and|comma)\s+/gi, ", ");
  if (!spoken) return current;
  const existing = current.trim().replace(/[，,\s]+$/, "");
  return existing ? existing + ", " + spoken.replace(/^[，,\s]+/, "") : spoken;
}

function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return <label className="block text-sm font-semibold text-slate-800">
    <span className="flex flex-wrap items-baseline justify-between gap-x-2 gap-y-0.5"><span>{label}</span>{hint ? <span className="text-xs font-normal text-slate-400">{hint}</span> : null}</span>
    <span className="mt-1.5 block">{children}</span>
  </label>;
}

function Step({ number, title, text }: { number: string; title: string; text: string }) {
  return <li className="flex gap-3">
    <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-white/10 text-xs font-bold text-indigo-100">{number}</span>
    <span><span className="block text-sm font-semibold text-white">{title}</span><span className="mt-0.5 block text-xs leading-5 text-slate-300">{text}</span></span>
  </li>;
}

function MetricCard({ icon, label, value, detail }: { icon: string; label: string; value: string; detail: string }) {
  return <article className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-white px-4 py-4 shadow-sm shadow-slate-900/[0.02] sm:px-5">
    <span className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-indigo-50 text-lg font-semibold text-indigo-700">{icon}</span>
    <span className="min-w-0"><span className="block text-xs font-medium text-slate-500">{label}</span><span className="mt-0.5 block text-xl font-bold leading-6 tracking-tight text-slate-900">{value}</span></span>
    <span className="ml-auto hidden text-right text-xs text-slate-400 md:block">{detail}</span>
  </article>;
}

function BountyCard({ bounty }: { bounty: Bounty }) {
  const expiry = new Date(bounty.expiresAt);
  const expiryText = Number.isNaN(expiry.getTime()) ? "Expiry not available" : "Closes " + expiry.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  return <article className="rounded-xl border border-slate-200 p-4 transition hover:border-indigo-200 hover:shadow-md hover:shadow-indigo-950/[0.04] sm:p-5">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2"><span className="rounded-full bg-emerald-50 px-2.5 py-1 text-[11px] font-bold uppercase tracking-wide text-emerald-700">Open</span><span className="text-xs text-slate-400">{expiryText}</span></div>
        <h3 className="mt-2 break-words text-base font-bold leading-6 text-slate-900">{bounty.title}</h3>
      </div>
      <span className="shrink-0 rounded-xl bg-slate-50 px-3 py-2 text-sm font-bold text-slate-800">{"$" + bounty.amount.toLocaleString()}</span>
    </div>
    {bounty.description ? <p className="mt-3 whitespace-pre-line break-words text-sm leading-6 text-slate-600">{bounty.description}</p> : null}
    <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-slate-500">
      <span className="inline-flex items-center gap-1 rounded-lg bg-slate-50 px-2.5 py-1.5"><span className="size-1.5 rounded-full bg-indigo-500" />{bounty.posterTeam || "Team not specified"}</span>
      {bounty.skills.map((skill) => <span key={skill} className="rounded-lg border border-slate-200 px-2.5 py-1.5">{skill}</span>)}
    </div>
    <div className="mt-4 border-t border-slate-100 pt-4">
      <div className="flex items-center justify-between gap-2"><p className="text-xs font-bold uppercase tracking-[0.12em] text-slate-500">Suggested teammates</p><span className="text-xs font-medium text-slate-400">{bounty.matchedExperts?.length || 0} matches</span></div>
      {bounty.matchedExperts?.length ? (
        <ul className="mt-3 grid gap-2 sm:grid-cols-2">
          {bounty.matchedExperts.slice(0, 3).map((expert) => <li key={expert.expert_id} className="flex min-w-0 items-start gap-2.5 rounded-xl bg-slate-50 p-3">
            <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-indigo-100 text-xs font-bold text-indigo-700">{expert.name.split(/\s+/).map((part) => part[0]).join("").slice(0, 2)}</span>
            <span className="min-w-0 flex-1"><span className="block truncate text-sm font-semibold text-slate-800">{expert.name}</span><span className="block truncate text-xs text-slate-500">{expert.team}</span><span className="mt-1 block break-words text-[11px] leading-4 text-slate-500">Skills: {expert.matched_skills.join(", ")}</span></span>
            <span className="shrink-0 rounded-md bg-white px-1.5 py-1 text-[10px] font-bold text-indigo-700">{Math.round(expert.score)} pts</span>
          </li>)}
        </ul>
      ) : <p className="mt-2 text-sm text-slate-500">No teammates matched yet. Try adding more skills to a request.</p>}
    </div>
  </article>;
}

function ThemeIcon({ theme }: { theme: "light" | "dark" }) {
  return theme === "dark" ? (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className="size-4">
      <circle cx="12" cy="12" r="3.5" stroke="currentColor" strokeWidth="1.7" />
      <path d="M12 2.5v2m0 15v2M4.7 4.7l1.4 1.4m11.8 11.8 1.4 1.4M2.5 12h2m15 0h2M4.7 19.3l1.4-1.4M17.9 6.1l1.4-1.4" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    </svg>
  ) : (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className="size-4">
      <path d="M20.2 15.1A8.5 8.5 0 0 1 8.9 3.8 8.6 8.6 0 1 0 20.2 15.1Z" stroke="currentColor" strokeWidth="1.7" strokeLinejoin="round" />
    </svg>
  );
}

function RelayMark() {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className="size-5">
    <path d="M7 7.5h6.5a3 3 0 0 1 0 6H11m6-3h-6.5a3 3 0 0 0 0 6H13" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
    <circle cx="6" cy="7.5" r="2" fill="currentColor" /><circle cx="18" cy="10.5" r="2" fill="currentColor" /><circle cx="15" cy="16.5" r="2" fill="currentColor" />
  </svg>;
}

function SearchIcon() {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className="size-4 shrink-0 text-slate-400">
    <circle cx="10.8" cy="10.8" r="6.3" stroke="currentColor" strokeWidth="1.7" /><path d="m15.5 15.5 4 4" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
  </svg>;
}