"use client";
import { useState } from "react";
import { useBacktest } from "@/lib/hooks/queries";
import { BacktestParamsForm } from "./BacktestParamsForm";
import { BacktestMetrics } from "./BacktestMetrics";
import { EquityCurve } from "@/components/sentinel/EquityCurve";
import { DrawdownChart } from "./DrawdownChart";
import { MonthlyReturnChart } from "./MonthlyReturnChart";
import { TradeDistributionChart } from "./TradeDistributionChart";
import { PnlByScoreChart } from "./PnlByScoreChart";
import { EntryExitChart } from "./EntryExitChart";
import { Skeleton } from "@/components/ui/skeleton";
import type { BacktestParams } from "@/lib/api/types";

function EmptyState() {
  return (
    <div className="flex h-full items-center justify-center rounded-lg border border-dashed border-border bg-card py-24 text-center text-muted-foreground" style={{ fontSize: 13 }}>
      Parametreleri seç ve Çalıştır'a bas
    </div>
  );
}

export function BacktestContent() {
  const [params, setParams] = useState<BacktestParams | null>(null);
  const { data, isLoading, isError } = useBacktest(params);

  return (
    <div className="flex flex-col gap-3">
      <h1>Geriye Test</h1>
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-[300px_1fr]">
        <BacktestParamsForm onRun={setParams} />
        <div>
          {!params && <EmptyState />}
          {params && isLoading && <Skeleton className="h-96 w-full" />}
          {params && isError && (
            <div className="rounded-lg border border-border bg-card p-8 text-center text-muted-foreground">Backtest çalıştırılamadı.</div>
          )}
          {params && data && (
            <div className="flex flex-col gap-3">
              <BacktestMetrics metrics={data.metrics} />
              <EquityCurve data={data.equityCurve} title="Sermaye Eğrisi" />
              <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                <DrawdownChart data={data.drawdown} />
                <MonthlyReturnChart data={data.monthlyReturns} />
                <TradeDistributionChart data={data.tradeDistribution} />
                <PnlByScoreChart data={data.pnlByScore} />
              </div>
              <EntryExitChart priceSeries={data.priceSeries} trades={data.trades} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
