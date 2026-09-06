"use client";
import { useState } from "react";
import { toast } from "sonner";
import type { AlertTriggerType, DeliveryChannel } from "@/lib/api/types";
import type { RiskLevel } from "@/lib/format";
import { riskMeta } from "@/lib/format";
import {
  ALERT_TRIGGER_DEFS,
  DELIVERY_CHANNEL_DEFS,
  validateAlertRule,
  type AlertRuleDraft,
} from "@/lib/alerts/alert-defs";

const TRIGGERS = Object.keys(ALERT_TRIGGER_DEFS) as AlertTriggerType[];
const CHANNELS = Object.keys(DELIVERY_CHANNEL_DEFS) as DeliveryChannel[];
const MAX_RISKS: RiskLevel[] = ["medium", "high", "critical"];

const EMPTY: AlertRuleDraft = {
  name: "",
  trigger: "new_mint",
  scope: "Tüm tokenlar",
  minLiquidity: 0,
  minCreatorScore: 0,
  maxRisk: "medium",
  channels: ["slack"],
};

// AlertRuleForm, kontrollü kural oluşturma formu. Submit SIMÜLE (toast) — SentinelApi'de
// mutation yok; gerçek persist Backend Alt-proje 3.
export default function AlertRuleForm({ onDone }: { onDone?: () => void }) {
  const [draft, setDraft] = useState<AlertRuleDraft>(EMPTY);
  const [errors, setErrors] = useState<{ field: string; msg: string }[]>([]);

  const errOf = (field: string) => errors.find((e) => e.field === field)?.msg;
  const set = <K extends keyof AlertRuleDraft>(k: K, v: AlertRuleDraft[K]) =>
    setDraft((d) => ({ ...d, [k]: v }));
  const toggleChannel = (c: DeliveryChannel) =>
    setDraft((d) => ({
      ...d,
      channels: d.channels.includes(c) ? d.channels.filter((x) => x !== c) : [...d.channels, c],
    }));

  const submit = () => {
    const errs = validateAlertRule(draft);
    setErrors(errs);
    if (errs.length > 0) return;
    toast.success("Kural kaydedildi (simüle — kalıcı değil)");
    onDone?.();
  };

  return (
    <div className="space-y-4">
      <Field label="İsim" error={errOf("name")}>
        <input
          aria-label="İsim"
          value={draft.name}
          onChange={(e) => set("name", e.target.value)}
          className="w-full rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-sm"
          placeholder="ör. Balina alımı — tüm tokenlar"
        />
      </Field>

      <Field label="Tetikleyici">
        <select
          aria-label="Tetikleyici"
          value={draft.trigger}
          onChange={(e) => set("trigger", e.target.value as AlertTriggerType)}
          className="w-full rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-sm"
        >
          {TRIGGERS.map((t) => (
            <option key={t} value={t}>
              {ALERT_TRIGGER_DEFS[t].label}
            </option>
          ))}
        </select>
      </Field>

      <Field label="Kapsam">
        <input
          aria-label="Kapsam"
          value={draft.scope}
          onChange={(e) => set("scope", e.target.value)}
          className="w-full rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-sm"
        />
      </Field>

      <div className="grid grid-cols-2 gap-3">
        <Field label="Min. Likidite ($)" error={errOf("minLiquidity")}>
          <input
            type="number"
            aria-label="Min. Likidite"
            value={draft.minLiquidity}
            onChange={(e) => set("minLiquidity", Number(e.target.value))}
            className="w-full rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-sm"
          />
        </Field>
        <Field label="Min. Üretici Skoru" error={errOf("minCreatorScore")}>
          <input
            type="number"
            aria-label="Min. Üretici Skoru"
            value={draft.minCreatorScore}
            onChange={(e) => set("minCreatorScore", Number(e.target.value))}
            className="w-full rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-sm"
          />
        </Field>
      </div>

      <Field label="Maks. Risk">
        <select
          aria-label="Maks. Risk"
          value={draft.maxRisk}
          onChange={(e) => set("maxRisk", e.target.value as RiskLevel)}
          className="w-full rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-sm"
        >
          {MAX_RISKS.map((r) => (
            <option key={r} value={r}>
              {riskMeta[r].label}
            </option>
          ))}
        </select>
      </Field>

      <Field label="Teslimat Kanalları" error={errOf("channels")}>
        <div className="flex flex-wrap gap-2">
          {CHANNELS.map((c) => {
            const ch = DELIVERY_CHANNEL_DEFS[c];
            const active = draft.channels.includes(c);
            return (
              <label
                key={c}
                className={`inline-flex cursor-pointer items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs ${
                  active ? "border-primary/50 bg-primary/10 text-foreground" : "border-border/60 text-foreground/60"
                }`}
              >
                <input
                  type="checkbox"
                  className="sr-only"
                  checked={active}
                  onChange={() => toggleChannel(c)}
                  aria-label={ch.label}
                />
                {ch.label}
              </label>
            );
          })}
        </div>
      </Field>

      <div className="flex justify-end gap-2 pt-2">
        <button
          type="button"
          onClick={submit}
          className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          Kaydet
        </button>
      </div>
    </div>
  );
}

function Field({
  label,
  error,
  children,
}: {
  label: string;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <label className="mb-1 block text-xs text-foreground/60">{label}</label>
      {children}
      {error && <span className="mt-1 block text-xs text-critical">{error}</span>}
    </div>
  );
}
