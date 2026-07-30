"use client";
import { Search, Bell, Pause, Gauge, Fuel } from "lucide-react";
import { useSessionStore } from "@/lib/store/session";

export function Header() {
  const mode = useSessionStore((s) => s.tradingMode);
  return (
    <header className="relative flex h-14 shrink-0 items-center gap-4 border-b border-border bg-background/80 px-5 backdrop-blur">
      <div className="relative w-full max-w-md">
        <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
        <input placeholder="Token, cüzdan, üretici veya işlem ara"
          className="h-9 w-full rounded-md border border-border bg-input pl-9 pr-16 text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none" style={{ fontSize: 13 }} />
        <kbd className="absolute right-3 top-1/2 -translate-y-1/2 rounded border border-border px-1.5 py-0.5 font-mono text-muted-foreground" style={{ fontSize: 10 }}>⌘K</kbd>
      </div>
      <div className="ml-auto flex items-center gap-4">
        <select className="hidden h-8 rounded-md border border-border bg-input px-2 text-foreground focus:outline-none md:block" style={{ fontSize: 12 }} defaultValue="mainnet">
          <option value="mainnet">Solana Mainnet</option>
          <option value="devnet">Devnet</option>
        </select>
        <div className="hidden items-center gap-3 lg:flex">
          <Metric icon={<Gauge size={13} />} label="RPC" value="142ms" color="#2FD98B" />
          <Metric icon={<Fuel size={13} />} label="Ücret" value="0.00021" color="#8A94A6" />
          <span className="text-muted-foreground" style={{ fontSize: 11 }}>12sn önce güncellendi</span>
        </div>
        <button className="flex h-8 items-center gap-1.5 rounded-md px-3 transition-colors"
          style={{ backgroundColor: "rgba(240,71,107,0.12)", border: "1px solid rgba(240,71,107,0.4)", color: "#F0476B", fontSize: 12, fontWeight: 600 }}
          title="Halt all automated trading immediately">
          <Pause size={14} />Acil Durdur
        </button>
        <button className="relative text-muted-foreground transition-colors hover:text-foreground" title="Notifications">
          <Bell size={18} />
          <span className="absolute -right-0.5 -top-0.5 h-2 w-2 rounded-full bg-critical" />
        </button>
        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary font-mono" style={{ fontSize: 12, fontWeight: 600, color: "#fff" }}>AK</div>
      </div>
      {mode === "live" && (
        <div className="pointer-events-none absolute left-0 top-14 z-30 h-0.5 w-full" style={{ background: "linear-gradient(90deg,#F0476B,transparent)" }} />
      )}
    </header>
  );
}

function Metric({ icon, label, value, color }: { icon: React.ReactNode; label: string; value: string; color: string }) {
  return (
    <span className="flex items-center gap-1 text-muted-foreground" style={{ fontSize: 11 }}>
      <span style={{ color }}>{icon}</span>{label} <span className="font-mono text-foreground">{value}</span>
    </span>
  );
}
