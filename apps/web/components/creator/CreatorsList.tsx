"use client";
import Link from "next/link";
import { useCreators } from "@/lib/hooks/queries";
import { ScoreBadge } from "@/components/sentinel/ScoreBadge";
import { WalletAddress } from "@/components/sentinel/WalletAddress";
import { riskMeta } from "@/lib/format";

export function CreatorsList() {
  const { data } = useCreators();
  const rows = data ?? [];
  return (
    <div className="space-y-4">
      <h1>Üreticiler</h1>
      <div className="rounded-lg border border-border bg-card">
        <div className="overflow-x-auto">
          <table className="w-full border-collapse" style={{ fontSize: 13 }}>
            <thead>
              <tr className="text-muted-foreground" style={{ fontSize: 11 }}>
                {["Adres", "İtibar", "Toplam", "Aktif", "Rug", "Başarı", "Risk", ""].map((h) => (
                  <th key={h} className="whitespace-nowrap px-3 py-2 text-left font-normal">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((c) => {
                const rm = riskMeta[c.riskLevel];
                return (
                  <tr key={c.address} className="border-t border-border hover:bg-accent/40">
                    <td className="px-3 py-2"><Link href={`/creators/${c.address}`} className="hover:underline"><WalletAddress address={c.address} explorer={false} /></Link></td>
                    <td className="px-3 py-2"><ScoreBadge score={c.reputationScore} /></td>
                    <td className="px-3 py-2 font-mono tabular-nums">{c.totalTokens}</td>
                    <td className="px-3 py-2 font-mono tabular-nums">{c.activeTokens}</td>
                    <td className="px-3 py-2 font-mono tabular-nums">{c.ruggedTokens}</td>
                    <td className="px-3 py-2 font-mono tabular-nums">%{c.successRatePct}</td>
                    <td className="px-3 py-2"><span style={{ color: rm.color, fontSize: 11 }}>{rm.label}</span></td>
                    <td className="px-3 py-2"><Link href={`/creators/${c.address}`} className="text-primary" style={{ fontSize: 12 }}>Profil →</Link></td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
