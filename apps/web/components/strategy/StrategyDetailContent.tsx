"use client";
import { useStrategy } from "@/lib/hooks/queries";
import { Skeleton } from "@/components/ui/skeleton";
import { MetricTile } from "@/components/sentinel/MetricTile";
import { StatusBadge } from "./StatusBadge";
import { ConditionList } from "./ConditionList";
import { StrategyPerformancePanel } from "./StrategyPerformancePanel";
import { BacktestSummaryPanel } from "./BacktestSummaryPanel";
import { EquityCurve } from "@/components/sentinel/EquityCurve";
import { VersionHistory } from "./VersionHistory";
import { AuditLog } from "./AuditLog";

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="space-y-3">
      <h2 style={{ fontSize: 14 }}>{title}</h2>
      {children}
    </section>
  );
}

export function StrategyDetailContent({ id }: { id: string }) {
  const { data: s, isError } = useStrategy(id);
  if (isError) {
    return <div className="rounded-lg border border-border bg-card p-8 text-center text-muted-foreground">Strateji bulunamadı: {id}</div>;
  }
  if (!s) {
    return <div className="space-y-4"><Skeleton className="h-24 w-full" /><Skeleton className="h-40 w-full" /></div>;
  }
  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-3">
            <h1>{s.name}</h1>
            <StatusBadge status={s.status} />
          </div>
          <div className="mt-1 text-muted-foreground" style={{ fontSize: 12 }}>{s.timeframe} · {s.description}</div>
        </div>
      </div>

      <Section title="Koşullar">
        <div className="grid grid-cols-1 gap-4 rounded-lg border border-border bg-card p-4 md:grid-cols-2">
          <ConditionList title="Giriş (IF)" conditions={s.entry} />
          <ConditionList title="Çıkış (THEN)" conditions={s.exit} />
        </div>
      </Section>

      <Section title="Risk & Pozisyon Boyutlandırma">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <MetricTile label="İşlem Başı Risk" value={`%${s.risk.riskPerTradePct}`} />
          <MetricTile label="Stop-Loss" value={`%${s.risk.stopLossPct}`} />
          <MetricTile label="Take-Profit" value={s.risk.takeProfitLevels.map((t) => `%${t}`).join(" / ")} />
          <MetricTile label="Drawdown Stop" value={`%${s.risk.maxDrawdownStopPct}`} />
          <MetricTile label="Boyut Modeli" value={s.sizing.model} />
          <MetricTile label="Pozisyon Boyutu" value={`%${s.sizing.sizePct}`} />
          <MetricTile label="Min Creator Skoru" value={`${s.minScores.creator}`} />
          <MetricTile label="Min Güvenlik Skoru" value={`${s.minScores.safety}`} />
        </div>
      </Section>

      <Section title="Performans">
        <StrategyPerformancePanel performance={s.performance} />
      </Section>

      <Section title="Equity Curve">
        <EquityCurve data={s.equityCurve} />
      </Section>

      <Section title="Backtest Özeti">
        <BacktestSummaryPanel backtest={s.backtest} />
      </Section>

      <Section title="Desteklenen Launchpad'ler">
        <div className="flex flex-wrap gap-2">
          {s.supportedLaunchpads.map((l) => (
            <span key={l} className="rounded-md border border-border bg-surface-2 px-2 py-1" style={{ fontSize: 12 }}>{l}</span>
          ))}
        </div>
      </Section>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <VersionHistory versions={s.versions} />
        <AuditLog audit={s.audit} />
      </div>
    </div>
  );
}
