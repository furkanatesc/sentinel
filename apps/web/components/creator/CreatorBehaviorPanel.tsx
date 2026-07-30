import { Check, X } from "lucide-react";
import type { CreatorBehavior } from "@/lib/api/types";

function Flag({ label, on }: { label: string; on: boolean }) {
  return (
    <div className="flex items-center gap-2" style={{ fontSize: 12 }}>
      {on ? <X size={14} className="text-critical" /> : <Check size={14} className="text-positive" />}
      <span className={on ? "text-foreground" : "text-muted-foreground"}>{label}</span>
    </div>
  );
}

export function CreatorBehaviorPanel({ behavior: b }: { behavior: CreatorBehavior }) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <h3 className="mb-3">Davranış Paterni</h3>
      <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
        <div style={{ fontSize: 12 }}>Deploy frekansı: <span className="font-mono">{b.deployFrequency}</span></div>
        <div style={{ fontSize: 12 }}>Ort. ilk satış: <span className="font-mono">{b.avgFirstSellMinutes} dk</span></div>
        <div style={{ fontSize: 12 }}>Tekrarlanan funder: <span className="font-mono text-muted-foreground">{b.repeatedFunders.join(", ")}</span></div>
        <div className="space-y-1">
          <Flag label="Benzer metadata" on={b.similarMetadata} />
          <Flag label="Aynı sosyal hesap" on={b.sameSocial} />
          <Flag label="Aynı likidite davranışı" on={b.sameLiquidityPattern} />
        </div>
      </div>
    </div>
  );
}
