"use client";
import { useState } from "react";
import { Star, Activity, ShoppingCart, ArrowUpDown } from "lucide-react";
import { toast } from "sonner";
import { useTokens } from "@/lib/hooks/queries";
import { formatAge, formatPrice, formatUsd } from "@/lib/format";
import type { TokenRow } from "@/lib/api/types";
import { TokenAvatar } from "@/components/sentinel/TokenAvatar";
import { ScoreBadge } from "@/components/sentinel/ScoreBadge";
import { WalletAddress } from "@/components/sentinel/WalletAddress";
import { Sparkline } from "@/components/sentinel/Sparkline";

const signalMeta: Record<NonNullable<TokenRow["signal"]>, { label: string; color: string; bg: string }> = {
  buy: { label: "Al", color: "#2FD98B", bg: "rgba(47,217,139,0.12)" },
  watch: { label: "İzle", color: "#3E9BFF", bg: "rgba(62,155,255,0.12)" },
  avoid: { label: "Kaçın", color: "#F0476B", bg: "rgba(240,71,107,0.12)" },
};
type SortKey = "ageSeconds" | "liquidity" | "momentum" | "creatorScore";

export function LiveTokenFeed() {
  const { data } = useTokens();
  const [watch, setWatch] = useState<Record<string, boolean>>({});
  const [sortKey, setSortKey] = useState<SortKey>("ageSeconds");
  const rows = data ?? [];
  const sorted = [...rows].sort((a, b) => (sortKey === "ageSeconds" ? a[sortKey] - b[sortKey] : b[sortKey] - a[sortKey]));
  const isWatched = (t: TokenRow) => watch[t.id] ?? t.watchlisted;
  const toggle = (id: string) => setWatch((w) => ({ ...w, [id]: !(w[id] ?? rows.find((r) => r.id === id)?.watchlisted) }));

  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <span className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-positive opacity-60" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-positive" />
          </span>
          <h3>Canlı Token Akışı</h3>
          <span className="text-muted-foreground" style={{ fontSize: 12 }}>· gerçek zamanlı</span>
        </div>
        <div className="flex items-center gap-1">
          {(["ageSeconds", "liquidity", "momentum", "creatorScore"] as SortKey[]).map((k) => (
            <button key={k} onClick={() => setSortKey(k)}
              className={`flex items-center gap-1 rounded px-2 py-1 ${sortKey === k ? "bg-accent text-foreground" : "text-muted-foreground hover:text-foreground"}`}
              style={{ fontSize: 11 }}>
              <ArrowUpDown size={11} />
              {k === "ageSeconds" ? "Yaş" : k === "liquidity" ? "Lik." : k === "momentum" ? "Momentum" : "Üretici"}
            </button>
          ))}
        </div>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse" style={{ fontSize: 13 }}>
          <thead>
            <tr className="text-muted-foreground" style={{ fontSize: 11 }}>
              {["Token", "Yaş", "Fiyat", "Likidite", "5dk Hac.", "Sahip", "Üretici", "Güvenlik", "Momentum", "Sinyal", ""].map((h) => (
                <th key={h} className="whitespace-nowrap px-3 py-2 text-left font-normal">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {sorted.map((t) => {
              const sig = t.signal ? signalMeta[t.signal] : null;
              const fresh = t.ageSeconds < 60;
              return (
                <tr key={t.id} data-fresh={fresh} className="border-t border-border transition-colors hover:bg-accent/40"
                  style={fresh ? { boxShadow: "inset 2px 0 0 #2FD98B" } : undefined}>
                  <td className="px-3 py-2.5">
                    <div className="flex items-center gap-2.5">
                      <TokenAvatar symbol={t.symbol} />
                      <div className="flex flex-col leading-tight">
                        <span style={{ fontWeight: 500 }}>{t.name} <span className="text-muted-foreground">{t.symbol}</span></span>
                        <WalletAddress address={t.mint} />
                      </div>
                    </div>
                  </td>
                  <td className="whitespace-nowrap px-3 py-2.5 font-mono tabular-nums" style={{ color: fresh ? "#2FD98B" : undefined }}>{formatAge(t.ageSeconds)}</td>
                  <td className="whitespace-nowrap px-3 py-2.5 font-mono tabular-nums">{formatPrice(t.price)}</td>
                  <td className="whitespace-nowrap px-3 py-2.5 font-mono tabular-nums">{formatUsd(t.liquidity)}</td>
                  <td className="whitespace-nowrap px-3 py-2.5 font-mono tabular-nums">{formatUsd(t.vol5m)}</td>
                  <td className="whitespace-nowrap px-3 py-2.5 font-mono tabular-nums">{t.holders}</td>
                  <td className="px-3 py-2.5"><ScoreBadge score={t.creatorScore} label="Üretici" /></td>
                  <td className="px-3 py-2.5"><ScoreBadge score={t.safetyScore} label="Güvenlik" /></td>
                  <td className="px-3 py-2.5">
                    <Sparkline data={t.spark} color={t.momentum >= 70 ? "#2FD98B" : t.momentum >= 50 ? "#3E9BFF" : "#F0476B"} width={64} height={22} />
                  </td>
                  <td className="px-3 py-2.5">
                    {sig && <span className="rounded px-2 py-0.5" style={{ color: sig.color, backgroundColor: sig.bg, fontSize: 11, fontWeight: 600 }}>{sig.label}</span>}
                  </td>
                  <td className="px-3 py-2.5">
                    <div className="flex items-center gap-1">
                      <button onClick={() => toggle(t.id)} className="rounded p-1 hover:bg-accent" title="İzleme">
                        <Star size={14} className={isWatched(t) ? "fill-warning text-warning" : "text-muted-foreground"} />
                      </button>
                      <button onClick={() => toast("Analiz: " + t.symbol)} className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground" title="Analiz"><Activity size={14} /></button>
                      <button onClick={() => toast("İşlem: " + t.symbol)} className="rounded p-1 text-primary hover:bg-accent" title="İşlem"><ShoppingCart size={14} /></button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
