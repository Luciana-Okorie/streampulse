"use client";

import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

interface EventTypeChartProps {
  data: { event_type: string; count: number }[];
}

export function EventTypeChart({ data }: EventTypeChartProps) {
  return (
    <div className="bg-surface border border-line rounded-lg p-5 h-full flex flex-col">
      <span className="text-[11px] uppercase tracking-widest text-textMuted mb-4">
        Event types
      </span>
      <div className="flex-1 min-h-[220px]">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} layout="vertical" margin={{ left: 10, right: 20 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#232C3D" horizontal={false} />
            <XAxis type="number" stroke="#6B7684" fontSize={11} fontFamily="var(--font-mono)" />
            <YAxis
              type="category"
              dataKey="event_type"
              stroke="#6B7684"
              fontSize={11}
              fontFamily="var(--font-mono)"
              width={110}
            />
            <Tooltip
              contentStyle={{ background: "#1A2130", border: "1px solid #232C3D", borderRadius: 8 }}
              labelStyle={{ color: "#E8ECF1" }}
            />
            <Bar dataKey="count" fill="#FF8A3D" radius={[0, 4, 4, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
