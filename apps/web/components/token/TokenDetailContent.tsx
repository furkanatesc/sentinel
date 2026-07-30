"use client";
import { useState } from "react";
import { useToken } from "@/lib/hooks/queries";
import { SCORE_DEFS, type ScoreKey } from "@/lib/token/score-defs";
import { Skeleton } from "@/components/ui/skeleton";
import { TokenHeader } from "./TokenHeader";
import { ScoreRow } from "./ScoreRow";
import { ExplainableScore } from "./ExplainableScore";
import { TokenTabs } from "./TokenTabs";

export function TokenDetailContent({ mint }: { mint: string }) {
  const { data: token, isError } = useToken(mint);
  const [selected, setSelected] = useState<ScoreKey>("opportunity");
  if (isError) return <div className="rounded-lg border border-border bg-card p-8 text-center text-muted-foreground">Token bulunamadı: {mint}</div>;
  if (!token) return <div className="space-y-4"><Skeleton className="h-28 w-full" /><Skeleton className="h-32 w-full" /></div>;
  const def = SCORE_DEFS.find((d) => d.key === selected)!;
  return (
    <div className="space-y-5">
      <TokenHeader token={token} />
      <ScoreRow scores={token.scores} selectedKey={selected} onSelect={setSelected} />
      <ExplainableScore def={def} score={token.scores[selected]} />
      <TokenTabs token={token} />
    </div>
  );
}
