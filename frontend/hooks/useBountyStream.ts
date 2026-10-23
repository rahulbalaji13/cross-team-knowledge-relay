import { useEffect, useState } from "react";

export interface Bounty {
  id: string;
  title: string;
  amount: number;
  skills: string[];
  expiresAt: string;
  matchedExperts?: Array<{ expert_id: string; name: string; team: string; score: number }>;
}

export function useBountyStream(_engineerId: string) {
  const [bounties, setBounties] = useState<Bounty[]>([]);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const events = new EventSource("/api/bounties/stream");
    events.onopen = () => setConnected(true);
    events.onerror = () => setConnected(false);
    events.addEventListener("bounty.created", (event) => {
      const bounty = JSON.parse((event as MessageEvent<string>).data) as Bounty;
      setBounties((current) => [bounty, ...current.filter((item) => item.id !== bounty.id)]);
    });
    return () => events.close();
  }, []);

  return { bounties, connected };
}
