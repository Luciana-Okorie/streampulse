interface StatCardProps {
  label: string;
  value: string;
  sublabel?: string;
  accent?: "signal" | "ok" | "error" | "info";
  hero?: boolean;
}

const accentClasses: Record<string, string> = {
  signal: "text-signal",
  ok: "text-ok",
  error: "text-error",
  info: "text-info",
};

export function StatCard({ label, value, sublabel, accent = "info", hero = false }: StatCardProps) {
  return (
    <div className="bg-surface border border-line rounded-lg p-5 flex flex-col justify-between instrument-texture">
      <div className="flex items-center justify-between">
        <span className="text-[11px] uppercase tracking-widest text-textMuted">{label}</span>
        <span className={`status-dot w-1.5 h-1.5 rounded-full ${accentClasses[accent]} bg-current`} />
      </div>
      <div
        className={`font-mono font-semibold mt-3 ${accentClasses[accent]} ${
          hero ? "text-5xl" : "text-3xl"
        }`}
      >
        {value}
      </div>
      {sublabel && <div className="text-xs text-textMuted mt-1">{sublabel}</div>}
    </div>
  );
}
