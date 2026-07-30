"use client";
import { useKpis } from "@/lib/hooks/queries";
import { useLiveTokens, useLiveAlerts } from "@/lib/hooks/live";
import { KpiCard } from "@/components/sentinel/KpiCard";
import { LiveTokenFeed } from "./LiveTokenFeed";
import { OpportunityRadar } from "./OpportunityRadar";
import { AlertsTimeline } from "./AlertsTimeline";

export function OverviewContent() {
  useLiveTokens();
  useLiveAlerts();
  const { data: kpis } = useKpis();
  return (
    <div className="space-y-5">
      <div>
        <h1>Genel Bakış</h1>
        <p className="text-muted-foreground" style={{ fontSize: 13 }}>
          Gerçek zamanlı Solana token istihbaratı · Keşfet → Analiz → Skorla → Uyar → İşlem → İzle
        </p>
      </div>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        {(kpis ?? []).map((k) => (<KpiCard key={k.id} kpi={k} />))}
      </div>
      <div className="grid grid-cols-1 gap-5 xl:grid-cols-3">
        <div className="xl:col-span-2"><LiveTokenFeed /></div>
        <div className="min-h-[420px]"><AlertsTimeline /></div>
      </div>
      <div className="min-h-[340px]"><OpportunityRadar /></div>
    </div>
  );
}
