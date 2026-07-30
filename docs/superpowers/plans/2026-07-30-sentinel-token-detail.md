# SENTINEL Frontend — Increment 2 Implementation Plan
### Token Detail (Header + 4 Skor + Overview + Risk Analizi + Açıklanabilir Skor)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Feed'den tıklanan tokenın `/tokens/[mint]` detay ekranını kur: header + aksiyonlar, 4 skor kartı, Overview sekmesi (Recharts), Risk Analizi sekmesi, Açıklanabilir Skor paneli — tümü mock seam üzerinden.

**Architecture:** Increment 1'in seam'i (`component → hook → getApi() → mock`) genişletilir (`getToken`). SOLID: skorlar/sekmeler/riskler **config-driven** (OCP), her bileşen tek sorumluluk (SRP), bileşenler `SentinelApi` soyutlamasına bağımlı (DIP), dar prop'lar (ISP). Türetme/hesap `lib/`'te saf fonksiyonlarda.

**Tech Stack:** Next 16 App Router, TS strict, Tailwind v4, shadcn/ui (Base UI), TanStack Query, Recharts, Vitest + RTL.

## Global Constraints
- Tüm iş `SENTINEL/apps/web/` altında; npm; branch `feat/token-detail`.
- **Dark-only**, **UI dili Türkçe** (teknik token/simge/adres hariç). Mono font yalnız sayı/adres/hash.
- **Veri kuralı:** hiçbir bileşen `lib/api/mock`'u doğrudan import etmez → `useToken`/`getApi()`.
- **Skor seviyeleri** (`lib/format.ts`): 0–24 Kritik · 25–49 Yüksek Risk · 50–69 Orta · 70–84 İyi · 85–100 Güçlü.
- **Clean Code & SOLID review ölçütü:** SRP/OCP/DIP/ISP; config-driven skor+sekme+risk; küçük odaklı dosyalar; anlamlı isimler; DRY (erken soyutlama yok); pristine test çıktısı; TDD.
- **shadcn Base UI:** ported bileşenlerde Radix `asChild` yerine `render` prop.
- Commit: her task sonunda; commit gövdesi şununla biter: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`. Scaffold `apps/web/AGENTS.md`/`CLAUDE.md` yok (silinmişti).
- Ekran görünen tüm metinler Türkçe olmalı (yeni bileşenler dahil).

---

### Task 1: `lib/format.ts` (risk severity + yüzde) ve `lib/token/score-defs.ts` (TDD)

**Files:**
- Modify: `apps/web/lib/format.ts`
- Create: `apps/web/lib/token/score-defs.ts`
- Test: `apps/web/lib/token/score-defs.test.ts`, `apps/web/lib/format.test.ts` (ekle)

**Interfaces:**
- Produces:
  - `formatPct(n: number): string` (ör. `12.3` → `"%12,3"` — Türkçe ondalık? Basit tut: `"%12.3"`)
  - `type RiskSeverity = "critical" | "high" | "medium" | "info"`
  - `riskSeverityMeta: Record<RiskSeverity, { label: string; color: string; bg: string; border: string }>`
  - `type ScoreKey = "opportunity" | "creatorReputation" | "tokenSafety" | "manipulationRisk"`
  - `SCORE_DEFS: { key: ScoreKey; label: string; higherIsBetter: boolean }[]`
  - `scoreDisplayLevel(value: number, higherIsBetter: boolean): RiskLevel` — renk seviyesi; `higherIsBetter` false ise `scoreToLevel(100 - value)` (Manipulation Risk ters).

- [ ] **Step 1: Testleri yaz** (`lib/token/score-defs.test.ts`)

```ts
import { SCORE_DEFS, scoreDisplayLevel } from "./score-defs";

test("SCORE_DEFS has the 4 scores with correct polarity", () => {
  const byKey = Object.fromEntries(SCORE_DEFS.map((d) => [d.key, d]));
  expect(SCORE_DEFS).toHaveLength(4);
  expect(byKey.opportunity.higherIsBetter).toBe(true);
  expect(byKey.creatorReputation.higherIsBetter).toBe(true);
  expect(byKey.tokenSafety.higherIsBetter).toBe(true);
  expect(byKey.manipulationRisk.higherIsBetter).toBe(false);
});

test("scoreDisplayLevel inverts when higher is worse", () => {
  expect(scoreDisplayLevel(90, true)).toBe("strong");   // yüksek iyi
  expect(scoreDisplayLevel(90, false)).toBe("critical"); // yüksek manipülasyon = kötü
  expect(scoreDisplayLevel(10, false)).toBe("strong");   // düşük manipülasyon = iyi
});
```

- [ ] **Step 2: format testini genişlet** (`lib/format.test.ts`'e ekle)

```ts
import { riskSeverityMeta, formatPct } from "./format";

test("riskSeverityMeta covers every severity with hex color", () => {
  for (const s of ["critical", "high", "medium", "info"] as const) {
    expect(riskSeverityMeta[s].label).toBeTruthy();
    expect(riskSeverityMeta[s].color).toMatch(/^#/);
  }
});

test("formatPct", () => {
  expect(formatPct(12.34)).toBe("%12.3");
  expect(formatPct(100)).toBe("%100.0");
});
```

- [ ] **Step 3: Testlerin başarısız olduğunu doğrula** — Run: `npm run test -- score-defs format`; Expected: FAIL.

- [ ] **Step 4: `lib/format.ts`'e ekle**

```ts
import type { RiskLevel } from "./format"; // (aynı dosya; tip zaten burada)

export type RiskSeverity = "critical" | "high" | "medium" | "info";

export const riskSeverityMeta: Record<RiskSeverity, { label: string; color: string; bg: string; border: string }> = {
  critical: { label: "Kritik", color: "#F0476B", bg: "rgba(240,71,107,0.12)", border: "rgba(240,71,107,0.35)" },
  high: { label: "Yüksek", color: "#FFB020", bg: "rgba(255,176,32,0.12)", border: "rgba(255,176,32,0.35)" },
  medium: { label: "Orta", color: "#3E9BFF", bg: "rgba(62,155,255,0.12)", border: "rgba(62,155,255,0.35)" },
  info: { label: "Bilgi", color: "#8A94A6", bg: "rgba(138,148,166,0.12)", border: "rgba(138,148,166,0.30)" },
};

export function formatPct(n: number): string {
  return `%${n.toFixed(1)}`;
}
```
(Not: `RiskLevel`/`scoreToLevel`/`riskMeta` zaten `format.ts`'te; yeni `import` satırı ekleme — aynı dosyadalar. Yukarıdaki import satırını KOYMA, sadece export'ları ekle.)

- [ ] **Step 5: `lib/token/score-defs.ts` yaz**

```ts
import { scoreToLevel, type RiskLevel } from "@/lib/format";

export type ScoreKey = "opportunity" | "creatorReputation" | "tokenSafety" | "manipulationRisk";

export const SCORE_DEFS: { key: ScoreKey; label: string; higherIsBetter: boolean }[] = [
  { key: "opportunity", label: "Fırsat Skoru", higherIsBetter: true },
  { key: "creatorReputation", label: "Üretici İtibarı", higherIsBetter: true },
  { key: "tokenSafety", label: "Token Güvenliği", higherIsBetter: true },
  { key: "manipulationRisk", label: "Manipülasyon Riski", higherIsBetter: false },
];

/** Renk seviyesi: yüksek-kötü skorlarda ters çevrilir (100 - value). */
export function scoreDisplayLevel(value: number, higherIsBetter: boolean): RiskLevel {
  return scoreToLevel(higherIsBetter ? value : 100 - value);
}
```

- [ ] **Step 6: Testleri doğrula** — Run: `npm run test -- score-defs format`; Expected: PASS.
- [ ] **Step 7: Commit**

```bash
git add apps/web/lib/format.ts apps/web/lib/format.test.ts apps/web/lib/token/score-defs.ts apps/web/lib/token/score-defs.test.ts
git commit -m "feat(web): add risk-severity meta, formatPct, and config-driven score defs"
```

---

### Task 2: Veri katmanı — `TokenDetail` tipleri + `getToken` mock + `useToken` (TDD)

**Files:**
- Modify: `apps/web/lib/api/types.ts`, `apps/web/lib/api/contract.ts`, `apps/web/lib/api/mock.ts`, `apps/web/lib/api/http.ts`, `apps/web/lib/get-query-client.ts`, `apps/web/lib/hooks/queries.ts`
- Test: `apps/web/lib/api/token.test.ts`, `apps/web/lib/hooks/token.test.tsx`

**Interfaces:**
- Consumes: `ScoreKey` (score-defs), `RiskSeverity` (format), existing `TokenRow`/`tokens` seed.
- Produces: `ScoreBreakdownItem, ScoreDetail, RiskItem, RiskGroups, SeriesPoint, TokenMetrics, TokenDetail` types; `SentinelApi.getToken(idOrMint)`; `qk.token(mint)`; `useToken(mint)`.

- [ ] **Step 1: `lib/api/types.ts`'e ekle**

```ts
import type { ScoreKey } from "@/lib/token/score-defs";
import type { RiskSeverity } from "@/lib/format";

export interface ScoreBreakdownItem { label: string; weight: number; detail: string; }
export interface ScoreDetail { key: ScoreKey; value: number; confidence: number; updatedAt: string; breakdown: ScoreBreakdownItem[]; }
export interface RiskItem { id: string; title: string; severity: RiskSeverity; description: string; evidence?: string; firstSeen: string; lastSeen: string; }
export interface RiskGroups { contract: RiskItem[]; market: RiskItem[]; creator: RiskItem[]; }
export interface SeriesPoint { t: number; v: number; }
export interface TokenMetrics {
  holders: number; uniqueBuyers: number; buyRatio: number; sellRatio: number;
  creatorHoldingPct: number; top10HolderPct: number; sniperPct: number; botActivityPct: number;
}
export interface TokenDetail {
  id: string; name: string; symbol: string; mint: string;
  ageSeconds: number; price: number; priceChange24h: number;
  marketCap: number; liquidity: number; volume24h: number;
  scores: Record<ScoreKey, ScoreDetail>;
  metrics: TokenMetrics;
  series: { price: SeriesPoint[]; liquidity: SeriesPoint[]; volume: SeriesPoint[]; holders: SeriesPoint[] };
  risks: RiskGroups;
}
```

- [ ] **Step 2: `lib/api/contract.ts`'e ekle** — `SentinelApi` interface'ine:

```ts
  getToken(idOrMint: string): Promise<TokenDetail>;
```
ve `import type { ..., TokenDetail } from "./types";` güncelle.

- [ ] **Step 3: Testleri yaz** (`lib/api/token.test.ts`)

```ts
import { mockApi } from "./mock";
import { SCORE_DEFS } from "@/lib/token/score-defs";

test("getToken returns a full detail for a known symbol", async () => {
  const d = await mockApi.getToken("PULSE");
  expect(d.symbol).toBe("PULSE");
  for (const { key } of SCORE_DEFS) {
    expect(d.scores[key].value).toBeGreaterThanOrEqual(0);
    expect(d.scores[key].value).toBeLessThanOrEqual(100);
    expect(d.scores[key].breakdown.length).toBeGreaterThan(0);
  }
  expect(d.series.price.length).toBeGreaterThan(0);
  expect(d.risks.contract.length + d.risks.market.length + d.risks.creator.length).toBeGreaterThan(0);
});

test("getToken is case-insensitive and rejects unknown", async () => {
  expect((await mockApi.getToken("pulse")).symbol).toBe("PULSE");
  await expect(mockApi.getToken("NOPE")).rejects.toThrow();
});
```

- [ ] **Step 4: Testin başarısız olduğunu doğrula** — Run: `npm run test -- api/token`; Expected: FAIL (getToken yok).

- [ ] **Step 5: `lib/api/mock.ts`'e `getToken` üretimini ekle** (mevcut `spark`/`tokens` yeniden kullanılır)

```ts
import type { ScoreKey } from "@/lib/token/score-defs";
import type { RiskSeverity } from "@/lib/format";
import type { ScoreDetail, RiskItem, RiskGroups, SeriesPoint, TokenDetail } from "./types";

const clamp = (n: number) => Math.max(0, Math.min(100, Math.round(n)));
const seedOf = (s: string) => s.split("").reduce((a, c) => a + c.charCodeAt(0), 0);
const toSeries = (seed: number, len = 24): SeriesPoint[] => spark(seed, len).map((v, t) => ({ t, v }));

function scoreDetail(key: ScoreKey, value: number, seed: number, breakdown: ScoreDetail["breakdown"]): ScoreDetail {
  return { key, value: clamp(value), confidence: clamp(60 + (seed % 35)), updatedAt: "az önce", breakdown };
}

function buildDetail(row: TokenRow): TokenDetail {
  const seed = seedOf(row.symbol);
  const creatorRep = row.creatorScore;
  const safety = row.safetyScore;
  const manip = clamp(100 - safety * 0.7 - creatorRep * 0.3 + 10);
  const opp = clamp(0.35 * row.momentum + 0.35 * creatorRep + 0.3 * safety);

  const scores: Record<ScoreKey, ScoreDetail> = {
    opportunity: scoreDetail("opportunity", opp, seed + 1, [
      { label: "Momentum", weight: 35, detail: `Momentum skoru ${row.momentum}/100` },
      { label: "Üretici itibarı", weight: 35, detail: `Üretici skoru ${creatorRep}/100` },
      { label: "Token güvenliği", weight: 30, detail: `Güvenlik skoru ${safety}/100` },
    ]),
    creatorReputation: scoreDetail("creatorReputation", creatorRep, seed + 2, [
      { label: "Geçmiş performans", weight: 40, detail: creatorRep < 50 ? "Son tokenların çoğu 24s içinde büyük değer kaybetti" : "Geçmiş tokenlar makul performans gösterdi" },
      { label: "Bağlantılı cüzdanlar", weight: 35, detail: creatorRep < 50 ? "Bağlantılı cüzdanlarda likidite çekme paterni" : "Bağlantılı cüzdanlarda anormallik yok" },
      { label: "Cüzdan yaşı", weight: 25, detail: "Fonlayan cüzdan geçmişi incelendi" },
    ]),
    tokenSafety: scoreDetail("tokenSafety", safety, seed + 3, [
      { label: "Kontrat yetkileri", weight: 45, detail: safety < 60 ? "Mint/freeze authority aktif" : "Kritik yetkiler devre dışı" },
      { label: "Metadata", weight: 30, detail: safety < 60 ? "Değiştirilebilir metadata" : "Metadata kilitli" },
      { label: "Likidite kilidi", weight: 25, detail: row.liquidity < 20000 ? "Düşük/kilitsiz likidite" : "Yeterli likidite" },
    ]),
    manipulationRisk: scoreDetail("manipulationRisk", manip, seed + 4, [
      { label: "Holder yoğunluğu", weight: 40, detail: "Top-10 holder oranı izleniyor" },
      { label: "Wash trading", weight: 35, detail: manip > 60 ? "Şüpheli koordineli işlem paterni" : "Belirgin wash trading yok" },
      { label: "Sniper/bot", weight: 25, detail: manip > 60 ? "İlk alıcılarda yüksek bot oranı" : "Bot aktivitesi normal" },
    ]),
  };

  const metrics = {
    holders: row.holders,
    uniqueBuyers: Math.round(row.holders * 0.7),
    buyRatio: clamp(45 + (seed % 30)),
    sellRatio: clamp(55 - (seed % 30)),
    creatorHoldingPct: clamp(2 + (seed % 18)),
    top10HolderPct: clamp(18 + (seed % 40)),
    sniperPct: clamp(manip / 3),
    botActivityPct: clamp(manip / 2.5),
  };

  const risks: RiskGroups = { contract: [], market: [], creator: [] };
  const push = (g: RiskItem[], id: string, title: string, severity: RiskSeverity, description: string, evidence?: string) =>
    g.push({ id, title, severity, description, evidence, firstSeen: "12dk önce", lastSeen: "az önce" });
  if (safety < 70) push(risks.contract, `${row.id}-c1`, "Mint authority aktif", safety < 40 ? "critical" : "high", "Yeni token basılabilir; arz kontrol edilemiyor.", "Mint authority: aktif");
  if (safety < 60) push(risks.contract, `${row.id}-c2`, "Değiştirilebilir metadata", "medium", "Metadata update authority hâlâ açık.", "Update authority: aktif");
  if (row.liquidity < 20000) push(risks.market, `${row.id}-m1`, "Düşük likidite", "high", "Havuz sığ; yüksek price impact riski.", `Likidite: ${formatUsd(row.liquidity)}`);
  if (metrics.top10HolderPct > 45) push(risks.market, `${row.id}-m2`, "Konsantre holder dağılımı", "high", "Arzın büyük kısmı az sayıda cüzdanda.", `Top-10: %${metrics.top10HolderPct}`);
  if (creatorRep < 50) push(risks.creator, `${row.id}-cr1`, "Geçmiş rug bağlantıları", creatorRep < 30 ? "critical" : "high", "Üreticinin geçmiş tokenlarında likidite çekme paterni.", "Bağlantılı cüzdan kümesi tespit edildi");
  if (metrics.creatorHoldingPct > 12) push(risks.creator, `${row.id}-cr2`, "Yüksek üretici payı", "medium", "Üretici arzın önemli kısmını elinde tutuyor.", `Üretici payı: %${metrics.creatorHoldingPct}`);
  if (!risks.contract.length && !risks.market.length && !risks.creator.length)
    push(risks.market, `${row.id}-ok`, "Belirgin risk tespit edilmedi", "info", "Otomatik kontrollerde kritik bir bulgu yok.");

  return {
    id: row.id, name: row.name, symbol: row.symbol, mint: row.mint,
    ageSeconds: row.ageSeconds, price: row.price, priceChange24h: (seed % 40) - 15,
    marketCap: row.liquidity * 4, liquidity: row.liquidity, volume24h: row.vol5m * 12,
    scores, metrics,
    series: { price: toSeries(seed + 5), liquidity: toSeries(seed + 6), volume: toSeries(seed + 7), holders: toSeries(seed + 8) },
    risks,
  };
}
```
Ve `mockApi` nesnesine metod ekle:
```ts
  getToken(idOrMint) {
    const q = idOrMint.toLowerCase();
    const row = tokens.find((t) => t.symbol.toLowerCase() === q || t.id.toLowerCase() === q || t.mint.toLowerCase() === q);
    if (!row) return Promise.reject(new Error(`Token bulunamadı: ${idOrMint}`));
    return delay(buildDetail(row));
  },
```

- [ ] **Step 6: `lib/api/http.ts`'e stub ekle** — `getToken: notReady,`.

- [ ] **Step 7: `lib/get-query-client.ts`'e key ekle** — `qk` nesnesine: `token: (mint: string) => ["token", mint] as const,`.

- [ ] **Step 8: `lib/hooks/queries.ts`'e hook ekle**

```ts
export function useToken(mint: string) {
  return useQuery({ queryKey: qk.token(mint), queryFn: () => getApi().getToken(mint) });
}
```

- [ ] **Step 9: Hook testini yaz** (`lib/hooks/token.test.tsx`)

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useToken } from "./queries";

function Probe() { const { data } = useToken("PULSE"); return <div>sym:{data?.symbol ?? "-"}</div>; }

test("useToken loads a token detail", async () => {
  render(<QueryClientProvider client={getQueryClient()}><Probe /></QueryClientProvider>);
  await waitFor(() => expect(screen.getByText("sym:PULSE")).toBeInTheDocument());
});
```

- [ ] **Step 10: Testleri doğrula** — Run: `npm run test -- api/token hooks/token`; Expected: PASS. Run `npm run build`; Expected: OK.
- [ ] **Step 11: Commit**

```bash
git add apps/web/lib/api apps/web/lib/get-query-client.ts apps/web/lib/hooks/queries.ts apps/web/lib/hooks/token.test.tsx
git commit -m "feat(web): add TokenDetail types, mock getToken, and useToken hook"
```

---

### Task 3: Skor bileşenleri — MetricTile, ScoreCard, ScoreRow, ExplainableScore (TDD)

**Files:**
- Create: `apps/web/components/token/MetricTile.tsx`, `ScoreCard.tsx`, `ScoreRow.tsx`, `ExplainableScore.tsx`
- Test: `apps/web/components/token/ScoreCard.test.tsx`, `ExplainableScore.test.tsx`

**Interfaces:**
- Consumes: `ScoreDetail`, `SCORE_DEFS`, `scoreDisplayLevel`, `riskMeta`, `formatPct`.
- Produces: `<MetricTile label value hint?/>`, `<ScoreCard def score selected onExplain/>`, `<ScoreRow scores selectedKey onSelect/>`, `<ExplainableScore def score/>`.

- [ ] **Step 1: `MetricTile.tsx`** (SRP: tek metrik gösterimi)

```tsx
export function MetricTile({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="rounded-lg border border-border bg-surface-2 p-3">
      <div className="text-muted-foreground" style={{ fontSize: 11 }}>{label}</div>
      <div className="mt-1 font-mono tabular-nums" style={{ fontSize: 18, fontWeight: 600 }}>{value}</div>
      {hint && <div className="text-muted-foreground" style={{ fontSize: 10 }}>{hint}</div>}
    </div>
  );
}
```

- [ ] **Step 2: `ScoreCard.tsx`** (`"use client"`; ring meter; renk `scoreDisplayLevel` — Manipulation ters)

```tsx
"use client";
import { scoreDisplayLevel, type ScoreKey } from "@/lib/token/score-defs";
import { riskMeta } from "@/lib/format";
import type { ScoreDetail } from "@/lib/api/types";

interface Props {
  def: { key: ScoreKey; label: string; higherIsBetter: boolean };
  score: ScoreDetail;
  selected: boolean;
  onExplain: () => void;
}

export function ScoreCard({ def, score, selected, onExplain }: Props) {
  const level = scoreDisplayLevel(score.value, def.higherIsBetter);
  const meta = riskMeta[level];
  const R = 26, C = 2 * Math.PI * R;
  return (
    <div className={`rounded-lg border bg-card p-4 transition-colors ${selected ? "border-primary" : "border-border hover:border-white/15"}`}>
      <div className="flex items-center justify-between gap-2">
        <span className="text-muted-foreground" style={{ fontSize: 12 }}>{def.label}</span>
        <span className="rounded px-1.5 py-0.5" style={{ color: meta.color, backgroundColor: meta.bg, fontSize: 10, fontWeight: 600 }}>{meta.label}</span>
      </div>
      <div className="mt-2 flex items-center gap-3">
        <svg width="64" height="64" className="shrink-0 -rotate-90">
          <circle cx="32" cy="32" r={R} fill="none" stroke="var(--sentinel-surface-3)" strokeWidth="6" />
          <circle cx="32" cy="32" r={R} fill="none" stroke={meta.color} strokeWidth="6" strokeLinecap="round"
            strokeDasharray={C} strokeDashoffset={C * (1 - score.value / 100)} />
        </svg>
        <div className="flex flex-col">
          <span className="font-mono tabular-nums" style={{ fontSize: 24, fontWeight: 700, color: meta.color }}>{score.value}</span>
          <span className="text-muted-foreground" style={{ fontSize: 10 }}>Güven %{score.confidence} · {score.updatedAt}</span>
        </div>
      </div>
      <button onClick={onExplain} className="mt-2 text-primary hover:underline" style={{ fontSize: 11 }}>Neden bu skor?</button>
    </div>
  );
}
```

- [ ] **Step 3: `ScoreRow.tsx`** (OCP: SCORE_DEFS üzerinden map)

```tsx
"use client";
import { SCORE_DEFS, type ScoreKey } from "@/lib/token/score-defs";
import type { TokenDetail } from "@/lib/api/types";
import { ScoreCard } from "./ScoreCard";

export function ScoreRow({ scores, selectedKey, onSelect }: { scores: TokenDetail["scores"]; selectedKey: ScoreKey; onSelect: (k: ScoreKey) => void }) {
  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      {SCORE_DEFS.map((def) => (
        <ScoreCard key={def.key} def={def} score={scores[def.key]} selected={selectedKey === def.key} onExplain={() => onSelect(def.key)} />
      ))}
    </div>
  );
}
```

- [ ] **Step 4: `ExplainableScore.tsx`** (SRP: breakdown paneli)

```tsx
"use client";
import { scoreDisplayLevel, type ScoreKey } from "@/lib/token/score-defs";
import { riskMeta } from "@/lib/format";
import type { ScoreDetail } from "@/lib/api/types";

export function ExplainableScore({ def, score }: { def: { key: ScoreKey; label: string; higherIsBetter: boolean }; score: ScoreDetail }) {
  const meta = riskMeta[scoreDisplayLevel(score.value, def.higherIsBetter)];
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-center justify-between border-b border-border pb-2">
        <h3>{def.label} — neden {score.value}/100?</h3>
        <span style={{ color: meta.color, fontSize: 12, fontWeight: 600 }}>{meta.label}</span>
      </div>
      <ul className="mt-3 space-y-2">
        {score.breakdown.map((b, i) => (
          <li key={i} className="flex items-start gap-3">
            <span className="mt-0.5 shrink-0 rounded bg-surface-2 px-1.5 py-0.5 font-mono text-muted-foreground" style={{ fontSize: 10 }}>%{b.weight}</span>
            <div>
              <div style={{ fontSize: 13, fontWeight: 500 }}>{b.label}</div>
              <div className="text-muted-foreground" style={{ fontSize: 12 }}>{b.detail}</div>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
```

- [ ] **Step 5: Testleri yaz** (`ScoreCard.test.tsx`, `ExplainableScore.test.tsx`)

```tsx
// ScoreCard.test.tsx — Manipulation Risk ters mantık + Neden callback
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ScoreCard } from "./ScoreCard";
import type { ScoreDetail } from "@/lib/api/types";

const mk = (key: any, value: number): ScoreDetail => ({ key, value, confidence: 80, updatedAt: "az önce", breakdown: [{ label: "x", weight: 100, detail: "d" }] });

test("high manipulation score shows Kritik (inverted)", () => {
  render(<ScoreCard def={{ key: "manipulationRisk", label: "Manipülasyon Riski", higherIsBetter: false }} score={mk("manipulationRisk", 90)} selected={false} onExplain={() => {}} />);
  expect(screen.getByText("90")).toBeInTheDocument();
  expect(screen.getByText("Kritik")).toBeInTheDocument();
});

test("Neden bu skor triggers onExplain", async () => {
  const onExplain = vi.fn();
  render(<ScoreCard def={{ key: "tokenSafety", label: "Token Güvenliği", higherIsBetter: true }} score={mk("tokenSafety", 80)} selected={false} onExplain={onExplain} />);
  await userEvent.click(screen.getByText("Neden bu skor?"));
  expect(onExplain).toHaveBeenCalled();
});
```

```tsx
// ExplainableScore.test.tsx
import { render, screen } from "@testing-library/react";
import { ExplainableScore } from "./ExplainableScore";

test("renders weighted breakdown items", () => {
  render(<ExplainableScore def={{ key: "creatorReputation", label: "Üretici İtibarı", higherIsBetter: true }}
    score={{ key: "creatorReputation", value: 27, confidence: 70, updatedAt: "az önce", breakdown: [{ label: "Geçmiş performans", weight: 40, detail: "kötü" }] }} />);
  expect(screen.getByText("Geçmiş performans")).toBeInTheDocument();
  expect(screen.getByText("%40")).toBeInTheDocument();
});
```

- [ ] **Step 6: Testleri doğrula** — Run: `npm run test -- token/ScoreCard token/ExplainableScore`; Expected: PASS.
- [ ] **Step 7: Commit**

```bash
git add apps/web/components/token/MetricTile.tsx apps/web/components/token/ScoreCard.tsx apps/web/components/token/ScoreRow.tsx apps/web/components/token/ExplainableScore.tsx apps/web/components/token/ScoreCard.test.tsx apps/web/components/token/ExplainableScore.test.tsx
git commit -m "feat(web): add score card/row and explainable-score panel (config-driven)"
```

---

### Task 4: TokenHeader + TokenActions (TDD)

**Files:**
- Create: `apps/web/components/token/TokenActions.tsx`, `apps/web/components/token/TokenHeader.tsx`
- Test: `apps/web/components/token/TokenActions.test.tsx`, `TokenHeader.test.tsx`

**Interfaces:**
- Consumes: `TokenDetail`, `useSessionStore` (trading mode), `TokenAvatar`, `WalletAddress`, `formatAge/formatPrice/formatUsd`, `sonner`.
- Produces: `<TokenActions symbol tradingMode/>` (İzle/Telegram/Simüle/Al/Sat), `<TokenHeader token/>`.

- [ ] **Step 1: `TokenActions.tsx`** (`"use client"`; SRP: aksiyon grubu; Live modda uyarı)

```tsx
"use client";
import { Star, Send, FlaskConical } from "lucide-react";
import { toast } from "sonner";
import { useSessionStore } from "@/lib/store/session";

export function TokenActions({ symbol }: { symbol: string }) {
  const mode = useSessionStore((s) => s.tradingMode);
  const trade = (side: "Al" | "Sat") => {
    if (mode === "live") toast.warning(`CANLI mod — ${side} ${symbol}`, { description: "Gerçek para. Bu demoda emir gönderilmez." });
    else toast(`${side} ${symbol}`, { description: `${mode === "paper" ? "Kağıt" : "Gölge"} modda simüle edilir.` });
  };
  return (
    <div className="flex flex-wrap items-center gap-2">
      <button onClick={() => toast("İzleme listesine eklendi")} className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 hover:bg-accent" style={{ fontSize: 12 }}><Star size={14} /> İzle</button>
      <button onClick={() => toast("Telegram alarmı oluşturuldu")} className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 hover:bg-accent" style={{ fontSize: 12 }}><Send size={14} /> Telegram Alarmı</button>
      <button onClick={() => toast(`${symbol} işlemi simüle ediliyor…`)} className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 hover:bg-accent" style={{ fontSize: 12 }}><FlaskConical size={14} /> Simüle Et</button>
      <button onClick={() => trade("Al")} className="rounded-md px-4 py-1.5 text-primary-foreground" style={{ backgroundColor: "#2FD98B", fontSize: 12, fontWeight: 600, color: "#08210F" }}>Al</button>
      <button onClick={() => trade("Sat")} className="rounded-md px-4 py-1.5" style={{ backgroundColor: "rgba(240,71,107,0.15)", border: "1px solid rgba(240,71,107,0.4)", color: "#F0476B", fontSize: 12, fontWeight: 600 }}>Sat</button>
    </div>
  );
}
```

- [ ] **Step 2: `TokenHeader.tsx`** (SRP: kimlik + metrik; aksiyonları TokenActions'a delege eder)

```tsx
import { TokenAvatar } from "@/components/sentinel/TokenAvatar";
import { WalletAddress } from "@/components/sentinel/WalletAddress";
import { TokenActions } from "./TokenActions";
import { formatAge, formatPrice, formatUsd } from "@/lib/format";
import type { TokenDetail } from "@/lib/api/types";

function Stat({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div className="flex flex-col">
      <span className="text-muted-foreground" style={{ fontSize: 11 }}>{label}</span>
      <span className="font-mono tabular-nums" style={{ fontSize: 14, fontWeight: 600, color }}>{value}</span>
    </div>
  );
}

export function TokenHeader({ token }: { token: TokenDetail }) {
  const up = token.priceChange24h >= 0;
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <TokenAvatar symbol={token.symbol} size={44} />
          <div className="flex flex-col">
            <div className="flex items-center gap-2">
              <span style={{ fontSize: 18, fontWeight: 600 }}>{token.name}</span>
              <span className="text-muted-foreground">{token.symbol}</span>
            </div>
            <WalletAddress address={token.mint} />
          </div>
        </div>
        <TokenActions symbol={token.symbol} />
      </div>
      <div className="mt-4 grid grid-cols-3 gap-4 md:grid-cols-6">
        <Stat label="Fiyat" value={formatPrice(token.price)} />
        <Stat label="24s Değişim" value={`${up ? "+" : ""}${token.priceChange24h.toFixed(1)}%`} color={up ? "#2FD98B" : "#F0476B"} />
        <Stat label="Market Cap" value={formatUsd(token.marketCap)} />
        <Stat label="Likidite" value={formatUsd(token.liquidity)} />
        <Stat label="24s Hacim" value={formatUsd(token.volume24h)} />
        <Stat label="Yaş" value={formatAge(token.ageSeconds)} />
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Testleri yaz**

```tsx
// TokenActions.test.tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { TokenActions } from "./TokenActions";
import { useSessionStore } from "@/lib/store/session";
vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { warning: vi.fn() }) }));

test("Al in live mode warns about real funds", async () => {
  useSessionStore.setState({ tradingMode: "live" });
  render(<TokenActions symbol="PULSE" />);
  await userEvent.click(screen.getByText("Al"));
  expect((toast as any).warning).toHaveBeenCalled();
});
```

```tsx
// TokenHeader.test.tsx
import { render, screen } from "@testing-library/react";
import { TokenHeader } from "./TokenHeader";
import type { TokenDetail } from "@/lib/api/types";
const token = { id: "t1", name: "SolPulse", symbol: "PULSE", mint: "9xQeWv...4Fk2", ageSeconds: 38, price: 0.0042, priceChange24h: 6.2, marketCap: 320000, liquidity: 82400, volume24h: 494400, scores: {} as any, metrics: {} as any, series: {} as any, risks: {} as any } as TokenDetail;

test("header shows identity and price", () => {
  render(<TokenHeader token={token} />);
  expect(screen.getByText("SolPulse")).toBeInTheDocument();
  expect(screen.getByText("PULSE")).toBeInTheDocument();
  expect(screen.getByText("$0.0042")).toBeInTheDocument();
});
```

- [ ] **Step 4: Testleri doğrula** — Run: `npm run test -- token/TokenActions token/TokenHeader`; Expected: PASS.
- [ ] **Step 5: Commit**

```bash
git add apps/web/components/token/TokenActions.tsx apps/web/components/token/TokenHeader.tsx apps/web/components/token/TokenActions.test.tsx apps/web/components/token/TokenHeader.test.tsx
git commit -m "feat(web): add token header and action group (paper/shadow/live-aware)"
```

---

### Task 5: Sekmeler — tab-defs, TokenTabs, OverviewTab, RiskAnalysisTab (TDD)

**Files:**
- Create: `apps/web/components/token/tab-defs.ts`, `TokenTabs.tsx`, `tabs/OverviewTab.tsx`, `tabs/RiskAnalysisTab.tsx`
- Add shadcn: `tabs`
- Test: `apps/web/components/token/tabs/RiskAnalysisTab.test.tsx`, `apps/web/components/token/TokenTabs.test.tsx`

**Interfaces:**
- Consumes: `TokenDetail`, `riskSeverityMeta`, `formatUsd/formatPct`, Recharts, shadcn `Tabs`.
- Produces: `TAB_DEFS` registry; `<TokenTabs token/>`; `<OverviewTab token/>`; `<RiskAnalysisTab risks/>`.

- [ ] **Step 1: shadcn tabs ekle** — `npx shadcn@latest add tabs` (Base UI variant üretebilir; import `@/components/ui/tabs`).

- [ ] **Step 2: `tab-defs.ts`** (OCP registry — `built` false olanlar placeholder)

```ts
export interface TabDef { key: string; label: string; built: boolean; }
export const TAB_DEFS: TabDef[] = [
  { key: "overview", label: "Genel Bakış", built: true },
  { key: "risk", label: "Risk Analizi", built: true },
  { key: "market", label: "Piyasa", built: false },
  { key: "holders", label: "Sahipler", built: false },
  { key: "creator", label: "Üretici", built: false },
  { key: "wallet-graph", label: "Cüzdan Grafiği", built: false },
  { key: "transactions", label: "İşlemler", built: false },
  { key: "social", label: "Sosyal", built: false },
  { key: "signals", label: "Strateji Sinyalleri", built: false },
  { key: "audit", label: "Denetim Günlüğü", built: false },
];
```

- [ ] **Step 3: `tabs/OverviewTab.tsx`** (`"use client"`; Recharts + MetricTile)

```tsx
"use client";
import { AreaChart, Area, XAxis, YAxis, ResponsiveContainer, Tooltip } from "recharts";
import type { TokenDetail, SeriesPoint } from "@/lib/api/types";
import { MetricTile } from "../MetricTile";
import { formatPct } from "@/lib/format";

function MiniChart({ title, data, color }: { title: string; data: SeriesPoint[]; color: string }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>{title}</div>
      <div style={{ height: 140 }}>
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
            <defs><linearGradient id={`g-${color.slice(1)}`} x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor={color} stopOpacity={0.3} /><stop offset="100%" stopColor={color} stopOpacity={0} /></linearGradient></defs>
            <XAxis dataKey="t" hide /><YAxis hide domain={["dataMin", "dataMax"]} />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} labelFormatter={() => ""} />
            <Area type="monotone" dataKey="v" stroke={color} strokeWidth={1.5} fill={`url(#g-${color.slice(1)})`} />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

export function OverviewTab({ token }: { token: TokenDetail }) {
  const m = token.metrics;
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        <MiniChart title="Fiyat" data={token.series.price} color="#7C5CFF" />
        <MiniChart title="Likidite" data={token.series.liquidity} color="#3E9BFF" />
        <MiniChart title="Hacim" data={token.series.volume} color="#2FD98B" />
        <MiniChart title="Holder Büyümesi" data={token.series.holders} color="#FFB020" />
      </div>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-6">
        <MetricTile label="Benzersiz Alıcı" value={`${m.uniqueBuyers}`} />
        <MetricTile label="Al/Sat" value={`${m.buyRatio}/${m.sellRatio}`} />
        <MetricTile label="Üretici Payı" value={formatPct(m.creatorHoldingPct)} />
        <MetricTile label="Top-10 Holder" value={formatPct(m.top10HolderPct)} />
        <MetricTile label="Sniper" value={formatPct(m.sniperPct)} />
        <MetricTile label="Bot Aktivitesi" value={formatPct(m.botActivityPct)} />
      </div>
    </div>
  );
}
```

- [ ] **Step 4: `tabs/RiskAnalysisTab.tsx`** (config-driven kategoriler)

```tsx
import type { RiskGroups, RiskItem } from "@/lib/api/types";
import { riskSeverityMeta } from "@/lib/format";

const CATS: { key: keyof RiskGroups; label: string }[] = [
  { key: "contract", label: "Kontrat Riski" },
  { key: "market", label: "Piyasa Riski" },
  { key: "creator", label: "Üretici Riski" },
];

function RiskRow({ r }: { r: RiskItem }) {
  const meta = riskSeverityMeta[r.severity];
  return (
    <li className="rounded-md border border-border bg-surface-2 p-3">
      <div className="flex items-center justify-between gap-2">
        <span style={{ fontSize: 13, fontWeight: 500 }}>{r.title}</span>
        <span className="rounded px-1.5 py-0.5" style={{ color: meta.color, backgroundColor: meta.bg, fontSize: 10, fontWeight: 600 }}>{meta.label}</span>
      </div>
      <div className="mt-1 text-muted-foreground" style={{ fontSize: 12 }}>{r.description}</div>
      {r.evidence && <div className="mt-1 font-mono text-muted-foreground" style={{ fontSize: 11 }}>Kanıt: {r.evidence}</div>}
      <div className="mt-1 text-muted-foreground" style={{ fontSize: 10 }}>İlk: {r.firstSeen} · Son: {r.lastSeen}</div>
    </li>
  );
}

export function RiskAnalysisTab({ risks }: { risks: RiskGroups }) {
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      {CATS.map((c) => (
        <div key={c.key}>
          <h3 className="mb-2">{c.label}</h3>
          {risks[c.key].length ? (
            <ul className="space-y-2">{risks[c.key].map((r) => <RiskRow key={r.id} r={r} />)}</ul>
          ) : (
            <div className="rounded-md border border-dashed border-border p-4 text-center text-muted-foreground" style={{ fontSize: 12 }}>Bu kategoride risk yok</div>
          )}
        </div>
      ))}
    </div>
  );
}
```

- [ ] **Step 5: `TokenTabs.tsx`** (`"use client"`; TAB_DEFS registry; Overview+Risk dolu)

```tsx
"use client";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { TAB_DEFS } from "./tab-defs";
import { OverviewTab } from "./tabs/OverviewTab";
import { RiskAnalysisTab } from "./tabs/RiskAnalysisTab";
import type { TokenDetail } from "@/lib/api/types";

export function TokenTabs({ token }: { token: TokenDetail }) {
  return (
    <Tabs defaultValue="overview" className="w-full">
      <TabsList className="flex flex-wrap">
        {TAB_DEFS.map((t) => <TabsTrigger key={t.key} value={t.key}>{t.label}</TabsTrigger>)}
      </TabsList>
      <TabsContent value="overview" className="mt-4"><OverviewTab token={token} /></TabsContent>
      <TabsContent value="risk" className="mt-4"><RiskAnalysisTab risks={token.risks} /></TabsContent>
      {TAB_DEFS.filter((t) => !t.built).map((t) => (
        <TabsContent key={t.key} value={t.key} className="mt-4">
          <div className="rounded-lg border border-dashed border-border bg-card py-16 text-center text-muted-foreground" style={{ fontSize: 13 }}>{t.label} — yakında</div>
        </TabsContent>
      ))}
    </Tabs>
  );
}
```
(Not: shadcn Tabs API'si Base UI ise `Tabs`/`TabsList`/`TabsTrigger`/`TabsContent` isimleri farklı olabilir; üretilen `@/components/ui/tabs` export'larına uy. Prop adları farklıysa uyarlama yap — davranış aynı kalmalı.)

- [ ] **Step 6: Testleri yaz**

```tsx
// tabs/RiskAnalysisTab.test.tsx
import { render, screen } from "@testing-library/react";
import { RiskAnalysisTab } from "./RiskAnalysisTab";
import type { RiskGroups } from "@/lib/api/types";
const risks: RiskGroups = {
  contract: [{ id: "c1", title: "Mint authority aktif", severity: "critical", description: "d", firstSeen: "12dk önce", lastSeen: "az önce" }],
  market: [], creator: [],
};
test("renders categories, severity, and empty state", () => {
  render(<RiskAnalysisTab risks={risks} />);
  expect(screen.getByText("Kontrat Riski")).toBeInTheDocument();
  expect(screen.getByText("Mint authority aktif")).toBeInTheDocument();
  expect(screen.getByText("Kritik")).toBeInTheDocument();
  expect(screen.getAllByText("Bu kategoride risk yok").length).toBe(2); // market + creator
});
```

```tsx
// TokenTabs.test.tsx — Overview default + a placeholder tab exists
import { render, screen } from "@testing-library/react";
import { TokenTabs } from "./TokenTabs";
const token = { series: { price: [{t:0,v:1}], liquidity: [{t:0,v:1}], volume: [{t:0,v:1}], holders: [{t:0,v:1}] }, metrics: { holders:1, uniqueBuyers:1, buyRatio:50, sellRatio:50, creatorHoldingPct:5, top10HolderPct:20, sniperPct:5, botActivityPct:5 }, risks: { contract: [], market: [], creator: [] } } as any;
test("shows tabs incl. built and placeholder", () => {
  render(<TokenTabs token={token} />);
  expect(screen.getByText("Genel Bakış")).toBeInTheDocument();
  expect(screen.getByText("Risk Analizi")).toBeInTheDocument();
  expect(screen.getByText("Cüzdan Grafiği")).toBeInTheDocument();
});
```

- [ ] **Step 7: Testleri ve build'i doğrula** — Run: `npm run test -- token/tabs token/TokenTabs`; Expected: PASS. Run `npm run build`; Expected: OK.
- [ ] **Step 8: Commit**

```bash
git add apps/web/components/token/tab-defs.ts apps/web/components/token/TokenTabs.tsx apps/web/components/token/tabs apps/web/components/token/tabs/RiskAnalysisTab.test.tsx apps/web/components/token/TokenTabs.test.tsx apps/web/components/ui/tabs.tsx
git commit -m "feat(web): add token tabs registry, Overview and Risk-analysis tabs"
```

---

### Task 6: TokenDetailContent + route sayfası + feed bağlama (entegrasyon, TDD)

**Files:**
- Create: `apps/web/components/token/TokenDetailContent.tsx`, `apps/web/app/(app)/tokens/[mint]/page.tsx`
- Modify: `apps/web/components/dashboard/LiveTokenFeed.tsx` (satır → link)
- Test: `apps/web/components/dashboard/LiveTokenFeed.test.tsx` (link href)

**Interfaces:**
- Consumes: `useToken`, `getQueryClient/qk`, `getApi`, tüm token bileşenleri, `SCORE_DEFS`.
- Produces: `<TokenDetailContent mint/>`; `/tokens/[mint]` sayfası; feed satırından navigasyon.

- [ ] **Step 1: `TokenDetailContent.tsx`** (`"use client"`; kompozisyon + seçili skor state + loading/empty)

```tsx
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
```

- [ ] **Step 2: `app/(app)/tokens/[mint]/page.tsx`** (server; RSC prefetch + hydration)

```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { TokenDetailContent } from "@/components/token/TokenDetailContent";

export default async function TokenDetailPage({ params }: { params: Promise<{ mint: string }> }) {
  const { mint } = await params;
  const queryClient = getQueryClient();
  try {
    await queryClient.prefetchQuery({ queryKey: qk.token(mint), queryFn: () => getApi().getToken(mint) });
  } catch {
    // bilinmeyen token — client tarafında isError ile ele alınır
  }
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <TokenDetailContent mint={mint} />
    </HydrationBoundary>
  );
}
```
(Not: Next 16'da dinamik route `params` bir Promise'tir — `await params` kullan. Build sırasında doğrula.)

- [ ] **Step 3: `LiveTokenFeed.tsx` — token hücresini linke çevir** ve Analyze butonunu navigasyona bağla. Import `next/link`, import `useRouter` from `next/navigation`. Token adı/symbol bloğunu `<Link href={\`/tokens/${t.symbol}\`}>` ile sar (satır tıklanınca detay). Analyze butonu: `onClick={() => router.push(\`/tokens/${t.symbol}\`)}`. (Watchlist/Trade davranışı aynı kalır.)

- [ ] **Step 4: `LiveTokenFeed.test.tsx`'e link testi ekle**

```tsx
test("token links to its detail page", async () => {
  wrap();
  await waitFor(() => expect(screen.getByText("SolPulse")).toBeInTheDocument());
  const link = screen.getByText("SolPulse").closest("a")!;
  expect(link.getAttribute("href")).toBe("/tokens/PULSE");
});
```
(Mevcut `wrap()` helper'ını kullan; `LiveTokenFeed` `next/link` + `next/navigation` kullandığından `usePathname`/`useRouter` mock'u gerekiyorsa test setup'ına ekle: `vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn() }), usePathname: () => "/" }))`.)

- [ ] **Step 5: Testleri ve build'i doğrula** — Run: `npm run test`; Expected: tüm suite PASS. Run: `npm run build`; Expected: `/tokens/[mint]` dahil derlenir.
- [ ] **Step 6: Manuel kabul (dev)** — `npm run dev`, `/` → bir tokena tıkla → `/tokens/<symbol>` açılır: header + aksiyonlar, 4 skor (Manipülasyon Riski ters renk), "Neden bu skor?" paneli değiştirir, Genel Bakış sekmesi grafik+tile, Risk Analizi kategorileri; placeholder sekmeler "yakında".
- [ ] **Step 7: Commit**

```bash
git add apps/web/components/token/TokenDetailContent.tsx "apps/web/app/(app)/tokens" apps/web/components/dashboard/LiveTokenFeed.tsx apps/web/components/dashboard/LiveTokenFeed.test.tsx
git commit -m "feat(web): wire /tokens/[mint] detail page and feed navigation"
```

---

## Self-Review (yazar kontrolü)

**Spec coverage:**
- Rota `/tokens/[mint]` + RSC prefetch/hydration → Task 6. ✅
- Feed → detail navigasyon → Task 6. ✅
- Veri seam genişlemesi (types/getToken/http/useToken/qk), mock import yasağı → Task 2. ✅
- 4 skor + Manipulation ters renk (config `higherIsBetter`) → Task 1 (defs) + Task 3 (ScoreCard). ✅
- ExplainableScore breakdown → Task 3. ✅
- Header + aksiyonlar (Al/Sat toast + Live uyarısı) → Task 4. ✅
- Overview sekmesi (Recharts + metrik tile) → Task 5. ✅
- Risk Analizi sekmesi (kategori + severity + empty) → Task 5. ✅
- Placeholder sekmeler (TAB_DEFS registry) → Task 5. ✅
- Test stratejisi → her task. ✅

**SOLID kontrolü:**
- SRP: MetricTile/ScoreCard/ScoreRow/ExplainableScore/TokenHeader/TokenActions/OverviewTab/RiskAnalysisTab/TokenTabs ayrı dosyalar, tek iş. Hesap `lib/`'te. ✅
- OCP: `SCORE_DEFS`, `TAB_DEFS`, risk `CATS`, `CATS`-map — yeni ekleme mevcut kodu değiştirmez. ✅
- DIP: bileşenler `useToken`/`getApi()` soyutlamasına bağımlı; mock import yok (kabul kriteri #7 + Task 6 testte href, veri hook'tan). ✅
- ISP: ScoreCard `ScoreDetail`, RiskAnalysisTab `RiskGroups`, MetricTile dar props. ✅

**Placeholder taraması:** kod adımları gerçek; "yakında" yalnız bilinçli placeholder sekme içeriği. ✅

**Tip tutarlılığı:** `ScoreKey`/`RiskSeverity` tek kaynaktan; `TokenDetail` alanları Task 2'de tanımlı, sonraki task'larda aynı isimlerle kullanılıyor; `qk.token(mint)` prefetch (Task 6) ve hook (Task 2) aynı. shadcn Tabs export adları üretime göre uyarlanır (Task 5 Step 5 notu). ✅
