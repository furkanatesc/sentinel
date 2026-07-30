"use client";
import { useState } from "react";
import { FlaskConical, Eye, Zap, ChevronDown } from "lucide-react";
import { useSessionStore, type TradingMode } from "@/lib/store/session";

const modeMeta: Record<TradingMode, { label: string; color: string; bg: string; icon: typeof Zap }> = {
  paper: { label: "Paper", color: "#3E9BFF", bg: "rgba(62,155,255,0.14)", icon: FlaskConical },
  shadow: { label: "Shadow", color: "#FFB020", bg: "rgba(255,176,32,0.14)", icon: Eye },
  live: { label: "Live", color: "#F0476B", bg: "rgba(240,71,107,0.16)", icon: Zap },
};

export function TradingModeBadge({ collapsed }: { collapsed?: boolean }) {
  const mode = useSessionStore((s) => s.tradingMode);
  const onChange = useSessionStore((s) => s.setTradingMode);
  const [open, setOpen] = useState(false);
  const meta = modeMeta[mode];
  const Icon = meta.icon;
  return (
    <div className="relative">
      <button onClick={() => setOpen((o) => !o)} className="flex w-full items-center gap-2 rounded-md px-2.5 py-2"
        style={{ backgroundColor: meta.bg, border: `1px solid ${meta.color}55` }} title="Trading mode">
        <Icon size={15} style={{ color: meta.color }} />
        {!collapsed && (
          <>
            <span className="flex flex-col items-start leading-tight">
              <span className="text-muted-foreground" style={{ fontSize: 9 }}>MODE</span>
              <span style={{ fontSize: 12, fontWeight: 600, color: meta.color }}>{meta.label} Trading</span>
            </span>
            <ChevronDown size={14} className="ml-auto text-muted-foreground" />
          </>
        )}
      </button>
      {open && (
        <div className="absolute bottom-full left-0 z-20 mb-2 w-full min-w-[180px] rounded-md border border-border bg-popover p-1 shadow-xl">
          {(Object.keys(modeMeta) as TradingMode[]).map((m) => {
            const mm = modeMeta[m];
            const MIcon = mm.icon;
            return (
              <button key={m} onClick={() => { onChange(m); setOpen(false); }} className="flex w-full items-center gap-2 rounded px-2 py-1.5 hover:bg-accent">
                <MIcon size={14} style={{ color: mm.color }} />
                <span style={{ fontSize: 13 }}>{mm.label}</span>
                {m === "live" && (
                  <span className="ml-auto rounded px-1.5 py-0.5" style={{ fontSize: 9, color: "#F0476B", backgroundColor: "rgba(240,71,107,0.14)" }}>REAL FUNDS</span>
                )}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
