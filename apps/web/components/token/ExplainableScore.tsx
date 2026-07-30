"use client";
import { scoreDisplayLevel, type ScoreKey } from "@/lib/token/score-defs";
import { riskMeta } from "@/lib/format";
import type { ScoreDetail } from "@/lib/api/types";

export function ExplainableScore({ def, score }: { def: { key: ScoreKey; label: string; higherIsBetter: boolean }; score: ScoreDetail }) {
  const meta = riskMeta[scoreDisplayLevel(score.value, def.higherIsBetter)];
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-center justify-between border-b border-border pb-2">
        <h3>{def.label} — neden {score.value}/100?</h3>
        <span style={{ color: meta.color, fontSize: 12, fontWeight: 600 }}>{meta.label}</span>
      </div>
      <ul className="mt-3 space-y-2">
        {score.breakdown.map((b, i) => (
          <li key={i} className="flex items-start gap-3">
            <span className="mt-0.5 shrink-0 rounded bg-surface-2 px-1.5 py-0.5 font-mono text-muted-foreground" style={{ fontSize: 10 }}>%{b.weight}</span>
            <div>
              <div style={{ fontSize: 13, fontWeight: 500 }}>{b.label}</div>
              <div className="text-muted-foreground" style={{ fontSize: 12 }}>{b.detail}</div>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
