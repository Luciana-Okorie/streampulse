"use client";

import { useEffect, useRef, useState } from "react";

export interface DashboardTick {
  events_per_second: number;
  active_users: number;
  total_events: number;
  total_errors: number;
  error_rate: number;
  event_counts: Record<string, number>;
  timestamp: string;
}

export interface FeedItem {
  time: string;
  event_type: string;
}

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:4003/ws";

/**
 * Connects to the processor's WebSocket hub and keeps the latest tick plus a
 * capped rolling feed of recent events. Reconnects with backoff if the
 * connection drops, since the dashboard should recover on its own after a
 * processor restart or network blip.
 */
export function useStreamPulseSocket() {
  const [tick, setTick] = useState<DashboardTick | null>(null);
  const [connected, setConnected] = useState(false);
  const retryDelay = useRef(1000);

  useEffect(() => {
    let socket: WebSocket;
    let cancelled = false;

    function connect() {
      socket = new WebSocket(WS_URL);

      socket.onopen = () => {
        if (cancelled) return;
        setConnected(true);
        retryDelay.current = 1000;
      };

      socket.onmessage = (event) => {
        try {
          const parsed: DashboardTick = JSON.parse(event.data);
          setTick(parsed);
        } catch {
          // ignore malformed frames
        }
      };

      socket.onclose = () => {
        if (cancelled) return;
        setConnected(false);
        setTimeout(connect, retryDelay.current);
        retryDelay.current = Math.min(retryDelay.current * 2, 15000);
      };

      socket.onerror = () => {
        socket.close();
      };
    }

    connect();
    return () => {
      cancelled = true;
      socket?.close();
    };
  }, []);

  return { tick, connected };
}
