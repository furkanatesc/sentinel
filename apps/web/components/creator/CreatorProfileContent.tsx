"use client";
import { useCreator } from "@/lib/hooks/queries";
import { SCORE_DEFS } from "@/lib/token/score-defs";
import { Skeleton } from "@/components/ui/skeleton";
import { ScoreCard } from "@/components/token/ScoreCard";
import { ExplainableScore } from "@/components/token/ExplainableScore";
import { CreatorHeader } from "./CreatorHeader";
import { CreatorMetrics } from "./CreatorMetrics";
import { CreatorTokenHistoryTable } from "./CreatorTokenHistoryTable";
import { CreatorBehaviorPanel } from "./CreatorBehaviorPanel";

const REP_DEF = SCORE_DEFS.find((d) => d.key === "creatorReputation")!;

export function CreatorProfileContent({ address }: { address: string }) {
  const { data: profile, isError } = useCreator(address);
  if (isError) return <div className="rounded-lg border border-border bg-card p-8 text-center text-muted-foreground">Üretici bulunamadı: {address}</div>;
  if (!profile) return <div className="space-y-4"><Skeleton className="h-24 w-full" /><Skeleton className="h-40 w-full" /></div>;
  return (
    <div className="space-y-5">
      <CreatorHeader profile={profile} />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div><ScoreCard def={REP_DEF} score={profile.reputation} selected onExplain={() => {}} /></div>
        <div className="lg:col-span-2"><ExplainableScore def={REP_DEF} score={profile.reputation} /></div>
      </div>
      <CreatorMetrics metrics={profile.metrics} />
      <CreatorTokenHistoryTable history={profile.history} />
      <CreatorBehaviorPanel behavior={profile.behavior} />
    </div>
  );
}
