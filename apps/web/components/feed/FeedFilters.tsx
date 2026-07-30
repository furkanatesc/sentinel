"use client";
import { X } from "lucide-react";
import type { FeedFilters as Filters, EventType } from "@/lib/api/types";
import { EMPTY_FILTERS } from "@/lib/api/types";
import { EVENT_TYPE_DEFS } from "@/lib/feed/event-defs";
import { riskMeta, type RiskLevel } from "@/lib/format";

const LAUNCHPADS = ["Pump.fun", "Raydium", "Moonshot", "Meteora"];
const DEXES = ["Raydium", "Meteora", "Orca", "Jupiter"];
const RISKS: RiskLevel[] = ["critical", "high", "medium", "good", "strong"];

const inputCls = "h-8 w-28 rounded-md border border-border bg-input px-2 text-foreground focus:border-primary focus:outline-none";
const selectCls = "h-8 rounded-md border border-border bg-input px-2 text-foreground focus:outline-none";

export function FeedFilters({ value, onChange }: { value: Filters; onChange: (f: Filters) => void }) {
  const set = (patch: Partial<Filters>) => onChange({ ...value, ...patch });
  const toggle = <T,>(arr: T[], v: T): T[] => (arr.includes(v) ? arr.filter((x) => x !== v) : [...arr, v]);
  const num = (s: string) => (s === "" ? 0 : Number(s));

  return (
    <div className="space-y-3 rounded-lg border border-border bg-card p-3" style={{ fontSize: 12 }}>
      {/* Event tipi çipleri */}
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-muted-foreground">Event:</span>
        {EVENT_TYPE_DEFS.map((d) => {
          const on = value.types.includes(d.key);
          return (
            <button key={d.key} onClick={() => set({ types: toggle(value.types, d.key) })}
              className={`rounded px-2 py-1 ${on ? "bg-primary text-primary-foreground" : "bg-surface-2 text-muted-foreground hover:text-foreground"}`}>
              {d.label}
            </button>
          );
        })}
      </div>
      {/* Risk çipleri */}
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-muted-foreground">Risk:</span>
        {RISKS.map((r) => {
          const on = value.risks.includes(r);
          const m = riskMeta[r];
          return (
            <button key={r} onClick={() => set({ risks: toggle(value.risks, r) })}
              className="rounded px-2 py-1" style={{ color: on ? m.color : undefined, backgroundColor: on ? m.bg : "var(--sentinel-surface-2)", border: on ? `1px solid ${m.border}` : "1px solid transparent" }}>
              {m.label}
            </button>
          );
        })}
      </div>
      {/* Select + sayı inputları */}
      <div className="flex flex-wrap items-center gap-2">
        <select className={selectCls} value={value.launchpad} onChange={(e) => set({ launchpad: e.target.value })}>
          <option value="all">Launchpad: Tümü</option>
          {LAUNCHPADS.map((l) => <option key={l} value={l}>{l}</option>)}
        </select>
        <select className={selectCls} value={value.dex} onChange={(e) => set({ dex: e.target.value })}>
          <option value="all">DEX: Tümü</option>
          {DEXES.map((d) => <option key={d} value={d}>{d}</option>)}
        </select>
        <input className={inputCls} type="number" placeholder="Min likidite" value={value.minLiquidity || ""} onChange={(e) => set({ minLiquidity: num(e.target.value) })} />
        <input className={inputCls} type="number" placeholder="Min creator" value={value.minCreatorScore || ""} onChange={(e) => set({ minCreatorScore: num(e.target.value) })} />
        <input className={inputCls} type="number" placeholder="Max yaş (sn)" value={value.maxAgeSeconds ?? ""} onChange={(e) => set({ maxAgeSeconds: e.target.value === "" ? null : Number(e.target.value) })} />
        <input className={inputCls} type="number" placeholder="Min hacim" value={value.minVolume || ""} onChange={(e) => set({ minVolume: num(e.target.value) })} />
        <input className={inputCls} type="number" placeholder="Min holder %" value={value.minHolderGrowth || ""} onChange={(e) => set({ minHolderGrowth: num(e.target.value) })} />
        <label className="flex items-center gap-1.5 text-muted-foreground">
          <input type="checkbox" checked={value.watchlistOnly} onChange={(e) => set({ watchlistOnly: e.target.checked })} /> Watchlist
        </label>
        <button onClick={() => onChange(EMPTY_FILTERS)} className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-muted-foreground hover:text-foreground">
          <X size={12} /> Temizle
        </button>
      </div>
    </div>
  );
}
