"use client";
import { scoreDisplayLevel, type ScoreKey } from "@/lib/token/score-defs";
import { riskMeta } from "@/lib/format";
import type { ScoreDetail } from "@/lib/api/types";

interface Props {
  def: { key: ScoreKey; label: string; higherIsBetter: boolean };
  score: ScoreDetail;
  selected: boolean;
  onExplain: () => void;
}

export function ScoreCard({ def, score, selected, onExplain }: Props) {
  const level = scoreDisplayLevel(score.value, def.higherIsBetter);
  const meta = riskMeta[level];
  const R = 26, C = 2 * Math.PI * R;
  return (
    <div className={`rounded-lg border bg-card p-4 transition-colors ${selected ? "border-primary" : "border-border hover:border-white/15"}`}>
      <div className="flex items-center justify-between gap-2">
        <span className="text-muted-foreground" style={{ fontSize: 12 }}>{def.label}</span>
        <span className="rounded px-1.5 py-0.5" style={{ color: meta.color, backgroundColor: meta.bg, fontSize: 10, fontWeight: 600 }}>{meta.label}</span>
      </div>
      <div className="mt-2 flex items-center gap-3">
        <svg width="64" height="64" className="shrink-0 -rotate-90">
          <circle cx="32" cy="32" r={R} fill="none" stroke="var(--sentinel-surface-3)" strokeWidth="6" />
          <circle cx="32" cy="32" r={R} fill="none" stroke={meta.color} strokeWidth="6" strokeLinecap="round"
            strokeDasharray={C} strokeDashoffset={C * (1 - score.value / 100)} />
        </svg>
        <div className="flex flex-col">
          <span className="font-mono tabular-nums" style={{ fontSize: 24, fontWeight: 700, color: meta.color }}>{score.value}</span>
          <span className="text-muted-foreground" style={{ fontSize: 10 }}>Güven %{score.confidence} · {score.updatedAt}</span>
        </div>
      </div>
      <button onClick={onExplain} className="mt-2 text-primary hover:underline" style={{ fontSize: 11 }}>Neden bu skor?</button>
    </div>
  );
}
