"use client";

interface EventFeedProps {
  items: { time: string; eventType: string }[];
}

const typeColor = (eventType: string) => {
  if (eventType.endsWith(".error") || eventType.endsWith(".failed")) return "text-error";
  if (eventType.startsWith("payment")) return "text-ok";
  if (eventType.startsWith("api")) return "text-info";
  return "text-textMuted";
};

export function EventFeed({ items }: EventFeedProps) {
  return (
    <div className="bg-surface border border-line rounded-lg overflow-hidden flex flex-col h-full">
      <div className="px-4 py-3 border-b border-line flex items-center justify-between">
        <span className="text-[11px] uppercase tracking-widest text-textMuted">Live event feed</span>
        <span className="status-dot w-1.5 h-1.5 rounded-full bg-ok" />
      </div>
      <div className="flex-1 overflow-y-auto font-mono text-sm p-4 space-y-1">
        {items.length === 0 && (
          <div className="text-textMuted">Waiting for events…</div>
        )}
        {items.map((item, i) => (
          <div key={i} className="flex gap-3">
            <span className="text-textMuted">{item.time}</span>
            <span className={typeColor(item.eventType)}>{item.eventType}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
