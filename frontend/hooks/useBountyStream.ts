import { useEffect, useState } from 'react';
import io, { Socket } from 'socket.io-client';

export interface Bounty {
  id: string;
  title: string;
  amount: number;
  skills: string[];
  expiresAt: string;
}

export function useBountyStream(engineerId: string) {
  const [bounties, setBounties] = useState<Bounty[]>([]);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    // Connect to WebSocket gateway with token/identity
    const socket: Socket = io(process.env.NEXT_PUBLIC_WS_URL || 'http://localhost:8080', {
      query: { engineerId },
      reconnectionAttempts: 5,
      timeout: 10000,
    });

    socket.on('connect', () => setConnected(true));
    socket.on('disconnect', () => setConnected(false));

    // Handle incoming targeted bounties
    socket.on('bounty.matched', (newBounty: Bounty) => {
      setBounties(prev => {
        // Prevent duplicate injection
        const filtered = prev.filter(b => b.id !== newBounty.id);
        return [newBounty, ...filtered];
      });
    });

    // Handle decays/expiries dropping off the board
    socket.on('bounty.expired', (data: { id: string }) => {
      setBounties(prev => prev.filter(b => b.id !== data.id));
    });

    return () => {
      socket.disconnect();
    };
  }, [engineerId]);

  return { bounties, connected };
}
