"use client";
import { useState } from "react";
import { useStrategies } from "@/lib/hooks/queries";
import { validateParams } from "@/lib/backtest/validate";
import {
  DEFAULT_BACKTEST_PARAMS, RANGE_PRESETS, SLIPPAGE_MODELS, LATENCY_MODELS, LIQUIDITY_MODELS,
} from "@/lib/backtest/backtest-defs";
import type { BacktestParams } from "@/lib/api/types";

const selectCls = "h-8 w-full rounded-md border border-border bg-input px-2 text-foreground focus:outline-none";
const inputCls = "h-8 w-full rounded-md border border-border bg-input px-2 text-foreground focus:outline-none";

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1" style={{ fontSize: 12 }}>
      <span className="text-muted-foreground">{label}</span>
      {children}
    </label>
  );
}

export function BacktestParamsForm({ onRun }: { onRun: (p: BacktestParams) => void }) {
  const { data: strategies } = useStrategies();
  const [p, setP] = useState<BacktestParams>(DEFAULT_BACKTEST_PARAMS);
  const [errors, setErrors] = useState<{ [f: string]: string }>({});
  const set = (patch: Partial<BacktestParams>) => setP((prev) => ({ ...prev, ...patch }));

  const run = () => {
    const e = validateParams(p);
    setErrors(e);
    if (Object.keys(e).length === 0) onRun(p);
  };

  const Err = ({ f }: { f: string }) => (errors[f] ? <span style={{ fontSize: 11, color: "#F0476B" }}>{errors[f]}</span> : null);

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-border bg-card p-3">
      <div className="font-medium" style={{ fontSize: 13 }}>Parametreler</div>

      <Field label="Strateji">
        <select aria-label="Strateji" className={selectCls} style={{ fontSize: 12 }} value={p.strategyId} onChange={(e) => set({ strategyId: e.target.value })}>
          {(strategies ?? []).map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
        </select>
      </Field>
      <Err f="strategyId" />

      <Field label="Tarih Aralığı">
        <select aria-label="Tarih Aralığı" className={selectCls} style={{ fontSize: 12 }} value={p.rangePreset} onChange={(e) => set({ rangePreset: e.target.value })}>
          {RANGE_PRESETS.map((r) => <option key={r.key} value={r.key}>{r.label}</option>)}
        </select>
      </Field>

      <Field label="Başlangıç Sermayesi (SOL)">
        <input aria-label="Başlangıç Sermayesi (SOL)" type="number" className={inputCls} style={{ fontSize: 12 }} value={p.initialCapitalSol} onChange={(e) => set({ initialCapitalSol: Number(e.target.value) })} />
      </Field>
      <Err f="initialCapitalSol" />

      <Field label="Maks. Pozisyon">
        <input aria-label="Maks. Pozisyon" type="number" className={inputCls} style={{ fontSize: 12 }} value={p.maxPositions} onChange={(e) => set({ maxPositions: Number(e.target.value) })} />
      </Field>
      <Err f="maxPositions" />

      <Field label="Slippage Modeli">
        <select aria-label="Slippage Modeli" className={selectCls} style={{ fontSize: 12 }} value={p.slippageModel} onChange={(e) => set({ slippageModel: e.target.value })}>
          {SLIPPAGE_MODELS.map((m) => <option key={m.key} value={m.key}>{m.label}</option>)}
        </select>
      </Field>

      <Field label="Öncelik Ücreti (SOL)">
        <input aria-label="Öncelik Ücreti (SOL)" type="number" className={inputCls} style={{ fontSize: 12 }} value={p.priorityFee} onChange={(e) => set({ priorityFee: Number(e.target.value) })} />
      </Field>
      <Err f="priorityFee" />

      <Field label="Gecikme Modeli">
        <select aria-label="Gecikme Modeli" className={selectCls} style={{ fontSize: 12 }} value={p.latencyModel} onChange={(e) => set({ latencyModel: e.target.value })}>
          {LATENCY_MODELS.map((m) => <option key={m.key} value={m.key}>{m.label}</option>)}
        </select>
      </Field>

      <Field label="Likidite Modeli">
        <select aria-label="Likidite Modeli" className={selectCls} style={{ fontSize: 12 }} value={p.liquidityModel} onChange={(e) => set({ liquidityModel: e.target.value })}>
          {LIQUIDITY_MODELS.map((m) => <option key={m.key} value={m.key}>{m.label}</option>)}
        </select>
      </Field>

      <Field label="Min. Creator Skoru">
        <input aria-label="Min. Creator Skoru" type="number" className={inputCls} style={{ fontSize: 12 }} value={p.minCreatorScore} onChange={(e) => set({ minCreatorScore: Number(e.target.value) })} />
      </Field>
      <Err f="minCreatorScore" />

      <Field label="Min. Token Güvenliği">
        <input aria-label="Min. Token Güvenliği" type="number" className={inputCls} style={{ fontSize: 12 }} value={p.minTokenSafety} onChange={(e) => set({ minTokenSafety: Number(e.target.value) })} />
      </Field>
      <Err f="minTokenSafety" />

      <button onClick={run} className="mt-1 rounded-md px-3 py-2" style={{ fontSize: 13, fontWeight: 600, backgroundColor: "#2FD98B", color: "#08210F" }}>
        Çalıştır
      </button>
    </div>
  );
}
