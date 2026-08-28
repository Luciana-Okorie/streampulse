"use client";

import { useEffect, useState } from "react";
import { PulseBar } from "@/components/PulseBar";
import { StatCard } from "@/components/StatCard";
import { EventFeed } from "@/components/EventFeed";
import { EventTypeChart } from "@/components/EventTypeChart";
import { useStreamPulseSocket } from "@/lib/websocket";

const MAX_FEED_ITEMS = 40;

export default function DashboardPage() {
  const { tick, connected } = useStreamPulseSocket();
  const [feed, setFeed] = useState<{ time: string; eventType: string }[]>([]);

  // Derive a synthetic feed entry from the dominant event type each tick —
  // a stand-in until the processor also streams raw per-event lines.
  useEffect(() => {
    if (!tick) return;
    const entries = Object.entries(tick.event_counts || {});
    if (entries.length === 0) return;
    const top = entries.sort((a, b) => b[1] - a[1])[0];
    const time = new Date(tick.timestamp).toLocaleTimeString("en-US", { hour12: false });
    setFeed((prev) => [{ time, eventType: top[0] }, ...prev].slice(0, MAX_FEED_ITEMS));
  }, [tick]);

  const chartData = Object.entries(tick?.event_counts || {})
    .map(([event_type, count]) => ({ event_type, count }))
    .sort((a, b) => b.count - a.count);

  return (
    <main className="min-h-screen flex flex-col">
      <header className="flex items-center justify-between px-6 py-4 border-b border-line">
        <div>
          <h1 className="font-display font-bold text-lg tracking-tight">StreamPulse</h1>
          <p className="text-xs text-textMuted">Real-time event analytics</p>
        </div>
        <div className="flex items-center gap-2 text-xs text-textMuted">
          <span className={`status-dot w-1.5 h-1.5 rounded-full ${connected ? "bg-ok" : "bg-error"}`} />
          {connected ? "live" : "reconnecting…"}
        </div>
      </header>

      <PulseBar eventsPerSecond={tick?.events_per_second ?? 0} />

      <section className="grid grid-cols-2 lg:grid-cols-4 gap-4 p-6">
        <StatCard
          label="Events / sec"
          value={(tick?.events_per_second ?? 0).toLocaleString()}
          accent="signal"
          hero
        />
        <StatCard
          label="Total events today"
          value={(tick?.total_events ?? 0).toLocaleString()}
          accent="info"
        />
        <StatCard
          label="Error rate"
          value={`${(tick?.error_rate ?? 0).toFixed(1)}%`}
          accent="error"
        />
        <StatCard
          label="Active users"
          value={(tick?.active_users ?? 0).toLocaleString()}
          sublabel="last 5 minutes"
          accent="ok"
        />
      </section>

      <section className="grid grid-cols-1 lg:grid-cols-2 gap-4 px-6 pb-6 flex-1">
        <EventTypeChart data={chartData} />
        <EventFeed items={feed} />
      </section>
    </main>
  );
}
