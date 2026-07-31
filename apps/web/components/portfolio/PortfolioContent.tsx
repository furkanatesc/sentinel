"use client";
import { usePortfolio } from "@/lib/hooks/queries";
import { Skeleton } from "@/components/ui/skeleton";
import { EquityCurve } from "@/components/sentinel/EquityCurve";
import { PortfolioKpis } from "./PortfolioKpis";
import { PnlByStrategyChart } from "./PnlByStrategyChart";
import { RiskAllocationChart } from "./RiskAllocationChart";
import { WinLossChart } from "./WinLossChart";
import { OpenPositionsSummary } from "./OpenPositionsSummary";

export function PortfolioContent() {
  const { data, isError } = usePortfolio();
  if (isError) return <div className="rounded-lg border border-border bg-card p-8 text-center text-muted-foreground">Portföy yüklenemedi.</div>;
  if (!data) return <div className="space-y-4"><Skeleton className="h-24 w-full" /><Skeleton className="h-56 w-full" /></div>;
  return (
    <div className="space-y-5">
      <h1>Portföy</h1>
      <PortfolioKpis summary={data.summary} />
      <EquityCurve data={data.equityCurve} title="Portföy Değeri" />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <PnlByStrategyChart data={data.pnlByStrategy} />
        <RiskAllocationChart data={data.riskAllocation} />
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <WinLossChart data={data.winLoss} />
        <OpenPositionsSummary />
      </div>
    </div>
  );
}
