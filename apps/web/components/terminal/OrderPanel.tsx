"use client";
import { useState } from "react";
import { useMarketData } from "@/lib/hooks/queries";
import { validateOrder, simulateOrder } from "@/lib/terminal/order-logic";
import { DEFAULT_ORDER_DRAFT, ORDER_SIDE_DEFS, ORDER_TYPE_DEFS, type OrderDraft } from "@/lib/terminal/order-defs";
import { OrderConfirmDialog } from "./OrderConfirmDialog";
import { cn } from "@/lib/utils";

function NumberField({ id, label, value, onChange }: { id: string; label: string; value: number; onChange: (v: number) => void }) {
  return (
    <label htmlFor={id} className="flex flex-col gap-1" style={{ fontSize: 12 }}>
      <span className="text-muted-foreground">{label}</span>
      <input
        id={id} aria-label={label} type="number" value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="rounded-md border border-border bg-background px-2 py-1.5"
      />
    </label>
  );
}

export function OrderPanel({ mint }: { mint: string }) {
  const { data: market } = useMarketData(mint);
  const [draft, setDraft] = useState<OrderDraft>(DEFAULT_ORDER_DRAFT);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const set = (patch: Partial<OrderDraft>) => setDraft((d) => ({ ...d, ...patch }));

  if (!market) return <div className="h-full rounded-lg border border-border bg-card p-3 text-muted-foreground" style={{ fontSize: 13 }}>Yükleniyor…</div>;

  const errors = validateOrder(draft, market);
  const sim = simulateOrder(draft, market);
  const valid = Object.keys(errors).length === 0;

  return (
    <div className="flex h-full flex-col gap-3 rounded-lg border border-border bg-card p-3">
      <div className="flex gap-1">
        {ORDER_SIDE_DEFS.map((s) => (
          <button key={s.key} onClick={() => set({ side: s.key })}
            className="flex-1 rounded-md px-2 py-1.5"
            style={{ fontSize: 12, fontWeight: 600, backgroundColor: draft.side === s.key ? s.color : "transparent", color: draft.side === s.key ? "#08210F" : s.color, border: `1px solid ${s.color}` }}>
            {s.label}
          </button>
        ))}
      </div>
      <div className="flex gap-1">
        {ORDER_TYPE_DEFS.map((t) => (
          <button key={t.key} onClick={() => set({ type: t.key })}
            className={cn("flex-1 rounded-md border px-2 py-1", draft.type === t.key ? "border-primary bg-accent" : "border-border")}
            style={{ fontSize: 12 }}>
            {t.label}
          </button>
        ))}
      </div>

      <NumberField id="amount" label="Miktar (SOL)" value={draft.amountSol} onChange={(v) => set({ amountSol: v })} />
      {errors.amountSol && <span style={{ fontSize: 11, color: "#F0476B" }}>{errors.amountSol}</span>}

      {draft.type === "limit" && (
        <>
          <NumberField id="limit" label="Limit Fiyatı" value={draft.limitPrice ?? 0} onChange={(v) => set({ limitPrice: v })} />
          {errors.limitPrice && <span style={{ fontSize: 11, color: "#F0476B" }}>{errors.limitPrice}</span>}
        </>
      )}

      {/* sizePct is captured in the draft but not yet wired into sizing/exit logic (future increment). */}
      <NumberField id="size" label="Pozisyon %" value={draft.sizePct} onChange={(v) => set({ sizePct: v })} />
      <NumberField id="slippage" label="Slippage %" value={draft.slippagePct} onChange={(v) => set({ slippagePct: v })} />
      {errors.slippagePct && <span style={{ fontSize: 11, color: "#F0476B" }}>{errors.slippagePct}</span>}
      <NumberField id="fee" label="Öncelik Ücreti (SOL)" value={draft.priorityFee} onChange={(v) => set({ priorityFee: v })} />
      {errors.priorityFee && <span style={{ fontSize: 11, color: "#F0476B" }}>{errors.priorityFee}</span>}
      <NumberField id="sl" label="Stop-Loss %" value={draft.stopLossPct ?? 0} onChange={(v) => set({ stopLossPct: v })} />
      {errors.stopLossPct && <span style={{ fontSize: 11, color: "#F0476B" }}>{errors.stopLossPct}</span>}
      <NumberField id="tp" label="Take-Profit %" value={draft.takeProfitPct ?? 0} onChange={(v) => set({ takeProfitPct: v })} />
      {errors.takeProfitPct && <span style={{ fontSize: 11, color: "#F0476B" }}>{errors.takeProfitPct}</span>}
      {/* trailingPct is captured in the draft but not yet wired into sizing/exit logic (future increment). */}
      <NumberField id="trail" label="Trailing %" value={draft.trailingPct ?? 0} onChange={(v) => set({ trailingPct: v })} />

      <div className="mt-auto rounded-md border border-border p-2" style={{ fontSize: 12 }}>
        <div className="flex justify-between"><span className="text-muted-foreground">Tahmini Fiyat</span><span>{sim.estPrice} SOL</span></div>
        <div className="flex justify-between"><span className="text-muted-foreground">Fiyat Etkisi</span><span>%{sim.priceImpactPct}</span></div>
        <div className="flex justify-between"><span className="text-muted-foreground">Min. Alınan</span><span>{sim.minReceived}</span></div>
        <div className="flex justify-between"><span className="text-muted-foreground">Rota</span><span>{sim.route}</span></div>
      </div>

      <button disabled={!valid} onClick={() => setConfirmOpen(true)}
        className="rounded-md px-3 py-2 disabled:opacity-40"
        style={{ fontSize: 13, fontWeight: 600, backgroundColor: "#3E9BFF", color: "#04121F" }}>
        Önizle
      </button>

      <OrderConfirmDialog open={confirmOpen} draft={draft} market={market} onClose={() => setConfirmOpen(false)} />
    </div>
  );
}
