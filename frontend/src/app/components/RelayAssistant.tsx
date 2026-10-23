"use client";

import { useEffect, useRef, useState } from "react";

type ChatMessage = {
  id: string;
  role: "assistant" | "user";
  text: string;
  action?: "create";
};

const quickQuestions = [
  "How do I post a bounty?",
  "How are experts matched?",
  "Are rewards real?",
];

function getAnswer(question: string): Omit<ChatMessage, "id" | "role"> {
  const text = question.trim().toLowerCase();

  if (/^(hi|hello|hey|good morning|good afternoon)\b/.test(text)) {
    return { text: "Hi! I can explain the relay, help you post a bounty, or describe how expert matching works. What would you like to know?" };
  }

  if (/create|make|post|submit|new bounty|new request|how.*bount|how.*request|form/.test(text)) {
    return {
      text: "To post a bounty, choose “Ask for help” or “Post a request.” Add a clear title, describe the blocker, enter your team, list the skills you need separated by commas, then choose a demo reward and how long it should stay open. Select “Post request” to see suggested teammates. The reward amount is illustrative; no payment is processed.",
      action: "create",
    };
  }

  if (/match|expert|skill|score|recommend|suggest/.test(text)) {
    return { text: "The demo compares the skills on a bounty with its sample expert profiles, excludes people from the same team, and suggests up to three teammates. The score considers skill level, how recently a skill was used, and reputation. The profile data is sample data, not a live employee directory." };
  }

  if (/reward|money|pay|payment|amount|price|dollar|real/.test(text)) {
    return { text: "Reward amounts are only illustrative in this portfolio demo. It does not process payments or hold funds. You can enter an amount to describe the request, but no money moves." };
  }

  if (/offline|connect|refresh|load|feed|wake|error/.test(text)) {
    return { text: "If the request feed cannot load, the demo backend may be starting up. Use the “Refresh” button in the feed after a few moments. If the problem continues, the service may be unavailable." };
  }

  if (/privacy|sensitive|personal|data|persist|saved|store/.test(text)) {
    return { text: "Please do not include confidential or personal information. This is a public portfolio demo with sample expert profiles. Bounty requests are held in the backend’s in-memory demo store, so they may disappear when that service restarts." };
  }

  if (/neo4j|spreadsheet|excel|admin|upload|database/.test(text)) {
    return { text: "This website currently demonstrates bounty posting and sample expert matching. It does not yet provide an admin console, spreadsheet import, or Neo4j data management." };
  }

  if (/what is|about|website|relay|purpose|help/.test(text)) {
    return { text: "Cross-Team Knowledge Relay is a demo for sharing engineering blockers and finding relevant expertise outside your own team. Post a bounty with the skills you need, then review suggested teammates in the feed." };
  }

  return { text: "I can help with posting a bounty, expert matching, demo rewards, the request feed, and what data this prototype stores. Try one of those topics, or ask “How do I post a bounty?”" };
}

export default function RelayAssistant() {
  const [isOpen, setIsOpen] = useState(false);
  const [input, setInput] = useState("");
  const [messages, setMessages] = useState<ChatMessage[]>([
    {
      id: "welcome",
      role: "assistant",
      text: "Welcome to the Relay guide. Ask me how to post a bounty, how matches work, or what this demo does.",
    },
  ]);
  const inputRef = useRef<HTMLInputElement>(null);
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isOpen) inputRef.current?.focus();
  }, [isOpen]);

  useEffect(() => {
    if (isOpen) endRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [isOpen, messages]);

  function sendMessage(prompt = input) {
    const question = prompt.trim();
    if (!question) return;

    const answer = getAnswer(question);
    const messageId = Date.now().toString() + Math.random().toString(36).slice(2);
    setMessages((current) => [
      ...current,
      { id: messageId + "-user", role: "user", text: question },
      { id: messageId + "-assistant", role: "assistant", ...answer },
    ]);
    setInput("");
  }

  return (
    <div className="relay-chat-widget">
      {isOpen ? (
        <section id="relay-help-panel" role="dialog" aria-label="Relay website help" className="relay-chat-panel overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl shadow-slate-950/20">
          <header className="flex shrink-0 items-center justify-between gap-3 bg-[#111827] px-4 py-3.5 text-white">
            <div className="flex min-w-0 items-center gap-3">
              <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-indigo-500/25 text-indigo-100"><AssistantIcon /></span>
              <span className="min-w-0">
                <span className="block text-sm font-bold">Relay guide</span>
                <span className="block text-xs text-slate-300">Instant answers about this demo</span>
              </span>
            </div>
            <button type="button" onClick={() => setIsOpen(false)} aria-label="Close help chat" className="flex size-9 shrink-0 items-center justify-center rounded-lg text-slate-300 transition hover:bg-white/10 hover:text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-white">×</button>
          </header>

          <div className="min-h-0 flex-1 overflow-y-auto bg-slate-50 px-3 py-4 sm:px-4">
            <div role="log" aria-live="polite" aria-relevant="additions" className="space-y-3">
              {messages.map((message) => (
                <div key={message.id} className={message.role === "user" ? "flex justify-end" : "flex justify-start"}>
                  <div className={message.role === "user" ? "max-w-[88%] rounded-2xl rounded-br-md bg-indigo-600 px-3.5 py-2.5 text-sm leading-5 text-white" : "max-w-[92%] rounded-2xl rounded-bl-md border border-slate-200 bg-white px-3.5 py-2.5 text-sm leading-5 text-slate-700 shadow-sm"}>
                    <p className="whitespace-pre-line">{message.text}</p>
                    {message.action === "create" && (
                      <a href="#create" onClick={() => setIsOpen(false)} className="mt-3 inline-flex min-h-9 items-center gap-1.5 rounded-lg bg-indigo-50 px-3 py-2 text-xs font-bold text-indigo-700 transition hover:bg-indigo-100">
                        Open bounty form <span aria-hidden="true">→</span>
                      </a>
                    )}
                  </div>
                </div>
              ))}
              {messages.length === 1 && (
                <div className="flex flex-col items-start gap-2 pt-1">
                  {quickQuestions.map((question) => (
                    <button key={question} type="button" onClick={() => sendMessage(question)} className="rounded-full border border-slate-200 bg-white px-3 py-2 text-left text-xs font-semibold text-slate-600 transition hover:border-indigo-200 hover:bg-indigo-50 hover:text-indigo-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600">
                      {question}
                    </button>
                  ))}
                </div>
              )}
              <div ref={endRef} />
            </div>
          </div>

          <form onSubmit={(event) => { event.preventDefault(); sendMessage(); }} className="flex shrink-0 items-center gap-2 border-t border-slate-200 bg-white p-3">
            <label htmlFor="relay-chat-input" className="sr-only">Ask the Relay guide a question</label>
            <input id="relay-chat-input" ref={inputRef} value={input} onChange={(event) => setInput(event.target.value)} placeholder="Ask a question…" maxLength={300} className="min-h-10 min-w-0 flex-1 rounded-xl border border-slate-200 bg-slate-50 px-3 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-indigo-400 focus:bg-white focus:ring-4 focus:ring-indigo-100" />
            <button type="submit" disabled={!input.trim()} aria-label="Send message" className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-indigo-600 text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-40 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"><SendIcon /></button>
          </form>
          <p className="shrink-0 bg-white px-3 pb-2 text-center text-[10px] text-slate-400">Guided answers for the portfolio demo</p>
        </section>
      ) : null}

      <button type="button" onClick={() => setIsOpen((open) => !open)} aria-expanded={isOpen} aria-controls="relay-help-panel" aria-label={isOpen ? "Close Relay guide" : "Open Relay guide"} className="ml-auto flex min-h-12 items-center gap-2.5 rounded-full bg-indigo-600 px-4 text-sm font-semibold text-white shadow-lg shadow-indigo-950/25 transition hover:-translate-y-0.5 hover:bg-indigo-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600">
        {isOpen ? <CloseIcon /> : <AssistantIcon />}
        <span>{isOpen ? "Close guide" : "Ask Relay"}</span>
      </button>
    </div>
  );
}

function AssistantIcon() {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className="size-5">
    <path d="M5 18.5 3.7 21l4.4-1.5a9 9 0 1 0-3.1-3Z" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" />
    <path d="M8 10h8M8 14h5" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
  </svg>;
}

function SendIcon() {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className="size-4">
    <path d="m21 3-7.2 18-3.7-7.1L3 10.2 21 3Z" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    <path d="M10.1 13.9 15 9" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
  </svg>;
}

function CloseIcon() {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" className="size-4">
    <path d="m6 6 12 12M18 6 6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
  </svg>;
}