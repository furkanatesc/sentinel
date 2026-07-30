"use client";
import { ExternalLink } from "lucide-react";
import type { FeedEvent } from "@/lib/api/types";
import { formatUsd, riskMeta } from "@/lib/format";
import { TokenAvatar } from "@/components/sentinel/TokenAvatar";
import { ScoreBadge } from "@/components/sentinel/ScoreBadge";
import { EventTypeBadge } from "./EventTypeBadge";

export function FeedTable({ events, onRowClick }: { events: FeedEvent[]; onRowClick: (e: FeedEvent) => void }) {
  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="max-h-[600px] overflow-y-auto">
        <table className="w-full border-collapse" style={{ fontSize: 13 }}>
          <thead className="sticky top-0 bg-card">
            <tr className="text-muted-foreground" style={{ fontSize: 11 }}>
              {["Zaman", "Event", "Token", "Kaynak", "Likidite", "Creator", "Risk", ""].map((h) => (
                <th key={h} className="whitespace-nowrap border-b border-border px-3 py-2 text-left font-normal">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {events.map((e) => {
              const rm = riskMeta[e.riskLevel];
              const fresh = e.tokenAgeSeconds < 60;
              return (
                <tr key={e.id} onClick={() => onRowClick(e)}
                  className="cursor-pointer border-t border-border transition-colors hover:bg-accent/40"
                  style={fresh ? { boxShadow: "inset 2px 0 0 #2FD98B" } : undefined}>
                  <td className="whitespace-nowrap px-3 py-2 font-mono text-muted-foreground tabular-nums" style={{ fontSize: 11 }}>{e.time}</td>
                  <td className="px-3 py-2"><EventTypeBadge type={e.type} /></td>
                  <td className="px-3 py-2">
                    <div className="flex items-center gap-2">
                      <TokenAvatar symbol={e.symbol} size={22} />
                      <span style={{ fontWeight: 500 }}>{e.symbol}</span>
                    </div>
                  </td>
                  <td className="whitespace-nowrap px-3 py-2 text-muted-foreground" style={{ fontSize: 12 }}>{e.launchpad} · {e.dex}</td>
                  <td className="whitespace-nowrap px-3 py-2 font-mono tabular-nums">{formatUsd(e.liquidity)}</td>
                  <td className="px-3 py-2"><ScoreBadge score={e.creatorScore} /></td>
                  <td className="px-3 py-2"><span className="rounded px-1.5 py-0.5" style={{ color: rm.color, backgroundColor: rm.bg, fontSize: 11 }}>{rm.label}</span></td>
                  <td className="px-3 py-2"><ExternalLink size={14} className="text-muted-foreground" /></td>
                </tr>
              );
            })}
          </tbody>
        </table>
        {events.length === 0 && <div className="p-10 text-center text-muted-foreground" style={{ fontSize: 13 }}>Filtrelere uygun event yok</div>}
      </div>
    </div>
  );
}
