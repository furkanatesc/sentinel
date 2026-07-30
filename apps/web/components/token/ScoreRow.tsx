"use client";
import { SCORE_DEFS, type ScoreKey } from "@/lib/token/score-defs";
import type { TokenDetail } from "@/lib/api/types";
import { ScoreCard } from "./ScoreCard";

export function ScoreRow({ scores, selectedKey, onSelect }: { scores: TokenDetail["scores"]; selectedKey: ScoreKey; onSelect: (k: ScoreKey) => void }) {
  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      {SCORE_DEFS.map((def) => (
        <ScoreCard key={def.key} def={def} score={scores[def.key]} selected={selectedKey === def.key} onExplain={() => onSelect(def.key)} />
      ))}
    </div>
  );
}
