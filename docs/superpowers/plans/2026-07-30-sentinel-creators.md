# SENTINEL Frontend — Increment 5 Implementation Plan
### Creators (Creator Profile + Liste)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`).

**Goal:** `/creators` liste + `/creators/[address]` tam creator profili (header + reputation + 8 metrik + token geçmişi tablosu + davranış paterni) + Wallet Graph creator node linki — mock seam, mevcut bileşenler yeniden kullanılarak.

**Architecture:** Seam genişler (`getCreators`/`getCreator`). Reputation için Token Detail'in `ScoreCard`+`ExplainableScore` reuse; `MetricTile` paylaşılan `components/sentinel/`'e taşınır. Config-driven rozetler (OCP), seam (DIP), dar prop'lar (ISP).

**Tech Stack:** Next 16 App Router, TS strict, Tailwind v4, shadcn/ui, TanStack Query, Vitest + RTL.

## Global Constraints
- `SENTINEL/apps/web/`; npm; branch `feat/creators`.
- **Dark-only**, **UI Türkçe** (teknik token/simge/adres hariç). Mono yalnız sayı/adres.
- **Veri kuralı:** hiçbir bileşen `lib/api/mock`'u import etmez → `useCreators`/`useCreator`/`getApi()`.
- **Clean Code & SOLID + reuse ölçütü:** reputation `ScoreCard`+`ExplainableScore` reuse; `MetricTile` paylaşılır; rozetler config (OCP); mantık lib'de; küçük dosyalar; TDD.
- `RiskLevel`/`riskMeta`/`ScoreDetail`/`SCORE_DEFS` mevcut modüllerden reuse.
- Commit: her task sonunda; gövde şununla biter: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.

---

### Task 1: MetricTile'ı paylaşılan konuma taşı + outcome config (TDD)

**Files:**
- Move: `apps/web/components/token/MetricTile.tsx` → `apps/web/components/sentinel/MetricTile.tsx`
- Modify: `apps/web/components/token/tabs/OverviewTab.tsx` (import güncelle)
- Create: `apps/web/lib/creator/outcome-defs.ts`
- Test: `apps/web/lib/creator/outcome-defs.test.ts`

**Interfaces:**
- Produces: `MetricTile` at `@/components/sentinel/MetricTile`; `OUTCOME_DEFS`, `LIQUIDITY_DEFS`, `CreatorOutcome`, `LiquidityStatus` (types in api/types.ts, defs in outcome-defs).

- [ ] **Step 1: `lib/api/types.ts`'e ekle** — `CreatorOutcome`, `LiquidityStatus` type'ları (bu task'ta yalnız bunlar; profil tipleri Task 2'de):

```ts
export type CreatorOutcome = "active" | "graduated" | "dumped" | "rug" | "dead";
export type LiquidityStatus = "locked" | "unlocked" | "removed";
```

- [ ] **Step 2: `MetricTile`'ı taşı** — `git mv components/token/MetricTile.tsx components/sentinel/MetricTile.tsx` (içerik aynı; export aynı). `components/token/tabs/OverviewTab.tsx`'te `import { MetricTile } from "../MetricTile"` → `import { MetricTile } from "@/components/sentinel/MetricTile"`. Başka importçu yoksa doğrula (`grep -rn "token/MetricTile" apps/web`).

- [ ] **Step 3: `lib/creator/outcome-defs.ts` yaz**

```ts
import type { CreatorOutcome, LiquidityStatus } from "@/lib/api/types";

export const OUTCOME_DEFS: Record<CreatorOutcome, { label: string; color: string }> = {
  active: { label: "Aktif", color: "#3E9BFF" },
  graduated: { label: "Graduated", color: "#2FD98B" },
  dumped: { label: "Dump", color: "#FFB020" },
  rug: { label: "Rug", color: "#F0476B" },
  dead: { label: "Ölü", color: "#8A94A6" },
};

export const LIQUIDITY_DEFS: Record<LiquidityStatus, { label: string; color: string }> = {
  locked: { label: "Kilitli", color: "#2FD98B" },
  unlocked: { label: "Kilitsiz", color: "#FFB020" },
  removed: { label: "Çekildi", color: "#F0476B" },
};
```

- [ ] **Step 4: Testleri yaz** (`outcome-defs.test.ts`)

```ts
import { OUTCOME_DEFS, LIQUIDITY_DEFS } from "./outcome-defs";
test("every outcome and liquidity status has label+color", () => {
  for (const o of ["active", "graduated", "dumped", "rug", "dead"] as const) {
    expect(OUTCOME_DEFS[o].label).toBeTruthy(); expect(OUTCOME_DEFS[o].color).toMatch(/^#/);
  }
  for (const l of ["locked", "unlocked", "removed"] as const) {
    expect(LIQUIDITY_DEFS[l].label).toBeTruthy(); expect(LIQUIDITY_DEFS[l].color).toMatch(/^#/);
  }
});
```

- [ ] **Step 5: RED→GREEN + build** — Run: `npm run test`; the MetricTile move must keep ALL existing tests green (OverviewTab consumers). `npm run test -- outcome-defs`; PASS. `npm run build`; OK.
- [ ] **Step 6: Commit**

```bash
git add apps/web/components/sentinel/MetricTile.tsx apps/web/components/token/tabs/OverviewTab.tsx apps/web/lib/creator apps/web/lib/api/types.ts
git rm apps/web/components/token/MetricTile.tsx 2>/dev/null || true
git commit -m "refactor(web): share MetricTile; add creator outcome/liquidity config"
```

---

### Task 2: Creator data types + mock + hooks (TDD)

**Files:**
- Modify: `apps/web/lib/api/types.ts`, `contract.ts`, `mock.ts`, `http.ts`, `lib/get-query-client.ts`, `lib/hooks/queries.ts`
- Test: `apps/web/lib/api/creators.test.ts`, `apps/web/lib/hooks/creators.test.tsx`

**Interfaces:**
- Consumes: `ScoreDetail`, `RiskLevel`, `CreatorOutcome`, `LiquidityStatus`.
- Produces: `CreatorRow`, `CreatorTokenHistoryItem`, `CreatorBehavior`, `CreatorMetrics`, `CreatorProfile`; `getCreators`/`getCreator`; `qk.creators`/`qk.creator`; `useCreators`/`useCreator`.

- [ ] **Step 1: `lib/api/types.ts`'e ekle**

```ts
export interface CreatorRow {
  address: string; label?: string; reputationScore: number; riskLevel: RiskLevel;
  totalTokens: number; activeTokens: number; ruggedTokens: number; successRatePct: number; realizedPnlSol: number;
}
export interface CreatorTokenHistoryItem {
  id: string; symbol: string; mint: string; createdAt: string;
  peakMarketCap: number; currentMarketCap: number; maxDrawdownPct: number;
  liquidityStatus: LiquidityStatus; creatorSellPct: number; outcome: CreatorOutcome; riskFlags: string[];
}
export interface CreatorBehavior {
  deployFrequency: string; avgFirstSellMinutes: number; repeatedFunders: string[];
  similarMetadata: boolean; sameSocial: boolean; sameLiquidityPattern: boolean;
}
export interface CreatorMetrics {
  totalTokens: number; activeTokens: number; ruggedTokens: number; avgLifetimeHours: number;
  avgPeakMarketCap: number; realizedPnlSol: number; successRatePct: number; avgFirstSellMinutes: number;
}
export interface CreatorProfile {
  address: string; label?: string; walletAgeDays: number; firstSeen: string;
  reputation: ScoreDetail; riskLevel: RiskLevel; metrics: CreatorMetrics;
  history: CreatorTokenHistoryItem[]; behavior: CreatorBehavior;
}
```

- [ ] **Step 2: `contract.ts`** — `SentinelApi`'ye: `getCreators(): Promise<CreatorRow[]>;` ve `getCreator(address: string): Promise<CreatorProfile>;` + tip importları.

- [ ] **Step 3: Testi yaz** (`lib/api/creators.test.ts`)

```ts
import { mockApi } from "./mock";
test("getCreators returns rows", async () => {
  const rows = await mockApi.getCreators();
  expect(rows.length).toBeGreaterThan(3);
  expect(rows[0].address).toBeTruthy();
});
test("getCreator returns a full profile deterministically", async () => {
  const a = await mockApi.getCreator("CreAxz");
  expect(a.address).toBe("CreAxz");
  expect(a.reputation.key).toBe("creatorReputation");
  expect(a.reputation.breakdown.length).toBeGreaterThan(0);
  expect(a.history.length).toBeGreaterThan(0);
  const b = await mockApi.getCreator("CreAxz");
  expect(b.reputation.value).toBe(a.reputation.value); // deterministic
});
```

- [ ] **Step 4: RED** — Run: `npm run test -- api/creators`; FAIL.

- [ ] **Step 5: `mock.ts`'e ekle** (deterministik; `seedOf`/`clamp`/`delay`/`scoreToLevel`/`formatUsd` reuse; `tokens` seed sembolleri history'de kullanılabilir)

```ts
import type { CreatorRow, CreatorProfile, CreatorTokenHistoryItem, CreatorOutcome, LiquidityStatus } from "./types";

const OUTCOMES: CreatorOutcome[] = ["active", "graduated", "dumped", "rug", "dead"];
const LIQS: LiquidityStatus[] = ["locked", "unlocked", "removed"];
const CREATOR_ADDRS = ["CreAxz", "CreBmn", "CreCqw", "Dep7hn", "Dep9kf", "Dep2rt"];

function creatorRow(addr: string): CreatorRow {
  const seed = seedOf(addr);
  const rep = clamp(20 + (seed % 70));
  const total = 4 + (seed % 12);
  return {
    address: addr, reputationScore: rep, riskLevel: scoreToLevel(rep),
    totalTokens: total, activeTokens: (seed % 4), ruggedTokens: Math.min(total, (seed % 6)),
    successRatePct: clamp(rep - 10 + (seed % 20)), realizedPnlSol: (seed % 200) - 60,
  };
}

function creatorHistory(addr: string, n: number): CreatorTokenHistoryItem[] {
  const seed = seedOf(addr);
  return Array.from({ length: n }, (_, i) => {
    const s = seed + i * 13;
    const peak = 20000 + (s % 400) * 1000;
    const cur = Math.round(peak * ((s % 90) / 100));
    return {
      id: `${addr}-h${i}`, symbol: `T${(s % 90).toString(36).toUpperCase()}${i}`, mint: `${addr.slice(0,4)}...${i}`,
      createdAt: `${i + 1}g önce`, peakMarketCap: peak, currentMarketCap: cur,
      maxDrawdownPct: clamp(100 - (cur / peak) * 100), liquidityStatus: LIQS[s % LIQS.length],
      creatorSellPct: clamp(s % 60), outcome: OUTCOMES[s % OUTCOMES.length],
      riskFlags: s % 2 === 0 ? ["Mint authority aktif"] : [],
    };
  });
}

function buildCreator(addr: string): CreatorProfile {
  const seed = seedOf(addr);
  const rep = clamp(20 + (seed % 70));
  const total = 4 + (seed % 12);
  return {
    address: addr, walletAgeDays: 5 + (seed % 60), firstSeen: `${5 + (seed % 60)}g önce`,
    reputation: {
      key: "creatorReputation", value: rep, confidence: clamp(60 + (seed % 35)), updatedAt: "az önce",
      breakdown: [
        { label: "Geçmiş performans", weight: 40, detail: rep < 50 ? `Son ${total} tokenın çoğu 24s içinde büyük değer kaybetti` : "Geçmiş tokenlar makul performans gösterdi" },
        { label: "Bağlantılı cüzdanlar", weight: 25, detail: rep < 50 ? "Bağlantılı cüzdanlarda likidite çekme paterni" : "Anormallik yok" },
        { label: "Cüzdan yaşı ve fonlama", weight: 20, detail: `Cüzdan ${5 + (seed % 60)} günlük` },
        { label: "Tekrarlanan deploy", weight: 15, detail: total > 8 ? "Yüksek frekanslı deploy paterni" : "Düşük frekans" },
      ],
    },
    riskLevel: scoreToLevel(rep),
    metrics: {
      totalTokens: total, activeTokens: (seed % 4), ruggedTokens: Math.min(total, (seed % 6)),
      avgLifetimeHours: 2 + (seed % 72), avgPeakMarketCap: 30000 + (seed % 300) * 1000,
      realizedPnlSol: (seed % 200) - 60, successRatePct: clamp(rep - 10 + (seed % 20)), avgFirstSellMinutes: 3 + (seed % 90),
    },
    history: creatorHistory(addr, 4 + (seed % 4)),
    behavior: {
      deployFrequency: `${total} token / 30 gün`, avgFirstSellMinutes: 3 + (seed % 90),
      repeatedFunders: [`Fnd${seed % 9}...aa`, `Fnd${(seed + 3) % 9}...bb`],
      similarMetadata: seed % 2 === 0, sameSocial: seed % 3 === 0, sameLiquidityPattern: seed % 2 === 1,
    },
  };
}
const creators: CreatorRow[] = CREATOR_ADDRS.map(creatorRow);
```
Ve `mockApi`'ye:
```ts
  getCreators: () => delay(creators),
  getCreator: (address) => delay(buildCreator(address)),
```

- [ ] **Step 6: `http.ts`** — `getCreators: notReady, getCreator: notReady,`.
- [ ] **Step 7: `get-query-client.ts`** — `qk`'ye `creators: ["creators"] as const, creator: (address: string) => ["creator", address] as const,`.
- [ ] **Step 8: `queries.ts`** —

```ts
export function useCreators() { return useQuery({ queryKey: qk.creators, queryFn: () => getApi().getCreators() }); }
export function useCreator(address: string) { return useQuery({ queryKey: qk.creator(address), queryFn: () => getApi().getCreator(address) }); }
```

- [ ] **Step 9: Hook testi** (`lib/hooks/creators.test.tsx`)

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useCreator } from "./queries";
function Probe() { const { data } = useCreator("CreAxz"); return <div>a:{data?.address ?? "-"}</div>; }
test("useCreator loads a profile", async () => {
  render(<QueryClientProvider client={getQueryClient()}><Probe /></QueryClientProvider>);
  await waitFor(() => expect(screen.getByText("a:CreAxz")).toBeInTheDocument());
});
```

- [ ] **Step 10: GREEN + build** — Run: `npm run test -- creators`; PASS. Build OK.
- [ ] **Step 11: Commit**

```bash
git add apps/web/lib/api apps/web/lib/get-query-client.ts apps/web/lib/hooks
git commit -m "feat(web): add creators data seam (getCreators/getCreator) and hooks"
```

---

### Task 3: CreatorHeader + CreatorMetrics + CreatorBehaviorPanel (TDD)

**Files:**
- Create: `apps/web/components/creator/CreatorHeader.tsx`, `CreatorMetrics.tsx`, `CreatorBehaviorPanel.tsx`
- Test: `apps/web/components/creator/CreatorHeader.test.tsx`, `CreatorBehaviorPanel.test.tsx`

**Interfaces:**
- Consumes: `CreatorProfile`, `CreatorMetrics`, `CreatorBehavior`, `MetricTile` (`@/components/sentinel/MetricTile`), `WalletAddress`, `riskMeta`, `formatUsd`.
- Produces: `<CreatorHeader profile/>`, `<CreatorMetrics metrics/>`, `<CreatorBehaviorPanel behavior/>`.

- [ ] **Step 1: `CreatorHeader.tsx`** (`"use client"`; adres + yaş + risk + Watch/Telegram toast)

```tsx
"use client";
import { Star, Send } from "lucide-react";
import { toast } from "sonner";
import type { CreatorProfile } from "@/lib/api/types";
import { WalletAddress } from "@/components/sentinel/WalletAddress";
import { riskMeta } from "@/lib/format";

export function CreatorHeader({ profile }: { profile: CreatorProfile }) {
  const rm = riskMeta[profile.riskLevel];
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <h1 style={{ fontSize: 18 }}>Üretici</h1>
            <span className="rounded px-1.5 py-0.5" style={{ color: rm.color, backgroundColor: rm.bg, fontSize: 11, fontWeight: 600 }}>{rm.label}</span>
          </div>
          <WalletAddress address={profile.address} />
          <span className="text-muted-foreground" style={{ fontSize: 11 }}>Cüzdan yaşı: {profile.walletAgeDays} gün · İlk görülme: {profile.firstSeen}</span>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => toast("Üretici izleniyor")} className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 hover:bg-accent" style={{ fontSize: 12 }}><Star size={14} /> İzle</button>
          <button onClick={() => toast("Telegram alarmı oluşturuldu")} className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 hover:bg-accent" style={{ fontSize: 12 }}><Send size={14} /> Telegram Alarmı</button>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: `CreatorMetrics.tsx`** (config-driven 8 tile; MetricTile reuse)

```tsx
import type { CreatorMetrics as CM } from "@/lib/api/types";
import { MetricTile } from "@/components/sentinel/MetricTile";
import { formatUsd } from "@/lib/format";

export function CreatorMetrics({ metrics: m }: { metrics: CM }) {
  const tiles: { label: string; value: string }[] = [
    { label: "Toplam Token", value: `${m.totalTokens}` },
    { label: "Aktif Token", value: `${m.activeTokens}` },
    { label: "Rug Token", value: `${m.ruggedTokens}` },
    { label: "Ort. Ömür", value: `${m.avgLifetimeHours}s` },
    { label: "Ort. Peak MC", value: formatUsd(m.avgPeakMarketCap) },
    { label: "Realized PnL", value: `${m.realizedPnlSol} SOL` },
    { label: "Başarı Oranı", value: `%${m.successRatePct}` },
    { label: "Ort. İlk Satış", value: `${m.avgFirstSellMinutes} dk` },
  ];
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
      {tiles.map((t) => <MetricTile key={t.label} label={t.label} value={t.value} />)}
    </div>
  );
}
```

- [ ] **Step 3: `CreatorBehaviorPanel.tsx`**

```tsx
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
```

- [ ] **Step 4: Testleri yaz**

```tsx
// CreatorHeader.test.tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { CreatorHeader } from "./CreatorHeader";
import type { CreatorProfile } from "@/lib/api/types";
vi.mock("sonner", () => ({ toast: vi.fn() }));
const p = { address: "CreAxz", walletAgeDays: 19, firstSeen: "19g önce", riskLevel: "high", reputation: {} as any, metrics: {} as any, history: [], behavior: {} as any } as CreatorProfile;
test("shows wallet age and Watch triggers toast", async () => {
  render(<CreatorHeader profile={p} />);
  expect(screen.getByText(/Cüzdan yaşı: 19 gün/)).toBeInTheDocument();
  await userEvent.click(screen.getByText("İzle"));
  expect(toast).toHaveBeenCalled();
});
```

```tsx
// CreatorBehaviorPanel.test.tsx
import { render, screen } from "@testing-library/react";
import { CreatorBehaviorPanel } from "./CreatorBehaviorPanel";
test("renders behavior fields", () => {
  render(<CreatorBehaviorPanel behavior={{ deployFrequency: "12 token / 30 gün", avgFirstSellMinutes: 8, repeatedFunders: ["Fnd1...aa"], similarMetadata: true, sameSocial: false, sameLiquidityPattern: true }} />);
  expect(screen.getByText("12 token / 30 gün")).toBeInTheDocument();
  expect(screen.getByText("Benzer metadata")).toBeInTheDocument();
});
```

- [ ] **Step 5: GREEN + build** — Run: `npm run test -- creator/CreatorHeader creator/CreatorBehaviorPanel`; PASS. Build OK.
- [ ] **Step 6: Commit**

```bash
git add apps/web/components/creator/CreatorHeader.tsx apps/web/components/creator/CreatorMetrics.tsx apps/web/components/creator/CreatorBehaviorPanel.tsx apps/web/components/creator/CreatorHeader.test.tsx apps/web/components/creator/CreatorBehaviorPanel.test.tsx
git commit -m "feat(web): add creator header, metrics, and behavior panel"
```

---

### Task 4: CreatorTokenHistoryTable + CreatorsList (TDD)

**Files:**
- Create: `apps/web/components/creator/CreatorTokenHistoryTable.tsx`, `CreatorsList.tsx`
- Test: `apps/web/components/creator/CreatorTokenHistoryTable.test.tsx`, `CreatorsList.test.tsx`

**Interfaces:**
- Consumes: `CreatorTokenHistoryItem`, `CreatorRow`, `OUTCOME_DEFS`/`LIQUIDITY_DEFS`, `useCreators`, `ScoreBadge`, `WalletAddress`, `formatUsd`, `next/link`.
- Produces: `<CreatorTokenHistoryTable history/>`, `<CreatorsList/>`.

- [ ] **Step 1: `CreatorTokenHistoryTable.tsx`** (outcome/liquidity rozetleri config'ten; token linkleri)

```tsx
import Link from "next/link";
import type { CreatorTokenHistoryItem } from "@/lib/api/types";
import { OUTCOME_DEFS, LIQUIDITY_DEFS } from "@/lib/creator/outcome-defs";
import { formatUsd } from "@/lib/format";

export function CreatorTokenHistoryTable({ history }: { history: CreatorTokenHistoryItem[] }) {
  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="border-b border-border px-4 py-3"><h3>Token Geçmişi</h3></div>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse" style={{ fontSize: 13 }}>
          <thead>
            <tr className="text-muted-foreground" style={{ fontSize: 11 }}>
              {["Token", "Oluşturma", "Peak MC", "Mevcut MC", "Max Düşüş", "Likidite", "Satış %", "Sonuç"].map((h) => (
                <th key={h} className="whitespace-nowrap px-3 py-2 text-left font-normal">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {history.map((t) => {
              const om = OUTCOME_DEFS[t.outcome]; const lm = LIQUIDITY_DEFS[t.liquidityStatus];
              return (
                <tr key={t.id} className="border-t border-border hover:bg-accent/40">
                  <td className="px-3 py-2"><Link href={`/tokens/${t.symbol}`} className="hover:underline" style={{ fontWeight: 500 }}>{t.symbol}</Link></td>
                  <td className="whitespace-nowrap px-3 py-2 text-muted-foreground">{t.createdAt}</td>
                  <td className="whitespace-nowrap px-3 py-2 font-mono tabular-nums">{formatUsd(t.peakMarketCap)}</td>
                  <td className="whitespace-nowrap px-3 py-2 font-mono tabular-nums">{formatUsd(t.currentMarketCap)}</td>
                  <td className="whitespace-nowrap px-3 py-2 font-mono tabular-nums" style={{ color: "#F0476B" }}>-%{t.maxDrawdownPct}</td>
                  <td className="px-3 py-2"><span className="rounded px-1.5 py-0.5" style={{ color: lm.color, backgroundColor: `${lm.color}1f`, fontSize: 11 }}>{lm.label}</span></td>
                  <td className="whitespace-nowrap px-3 py-2 font-mono tabular-nums">%{t.creatorSellPct}</td>
                  <td className="px-3 py-2"><span className="rounded px-1.5 py-0.5" style={{ color: om.color, backgroundColor: `${om.color}1f`, fontSize: 11, fontWeight: 600 }}>{om.label}</span></td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: `CreatorsList.tsx`** (`"use client"`; useCreators; satır → profil linki)

```tsx
"use client";
import Link from "next/link";
import { useCreators } from "@/lib/hooks/queries";
import { ScoreBadge } from "@/components/sentinel/ScoreBadge";
import { WalletAddress } from "@/components/sentinel/WalletAddress";
import { riskMeta } from "@/lib/format";

export function CreatorsList() {
  const { data } = useCreators();
  const rows = data ?? [];
  return (
    <div className="space-y-4">
      <h1>Üreticiler</h1>
      <div className="rounded-lg border border-border bg-card">
        <div className="overflow-x-auto">
          <table className="w-full border-collapse" style={{ fontSize: 13 }}>
            <thead>
              <tr className="text-muted-foreground" style={{ fontSize: 11 }}>
                {["Adres", "İtibar", "Toplam", "Aktif", "Rug", "Başarı", "Risk", ""].map((h) => (
                  <th key={h} className="whitespace-nowrap px-3 py-2 text-left font-normal">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((c) => {
                const rm = riskMeta[c.riskLevel];
                return (
                  <tr key={c.address} className="border-t border-border hover:bg-accent/40">
                    <td className="px-3 py-2"><Link href={`/creators/${c.address}`} className="hover:underline"><WalletAddress address={c.address} explorer={false} /></Link></td>
                    <td className="px-3 py-2"><ScoreBadge score={c.reputationScore} /></td>
                    <td className="px-3 py-2 font-mono tabular-nums">{c.totalTokens}</td>
                    <td className="px-3 py-2 font-mono tabular-nums">{c.activeTokens}</td>
                    <td className="px-3 py-2 font-mono tabular-nums">{c.ruggedTokens}</td>
                    <td className="px-3 py-2 font-mono tabular-nums">%{c.successRatePct}</td>
                    <td className="px-3 py-2"><span style={{ color: rm.color, fontSize: 11 }}>{rm.label}</span></td>
                    <td className="px-3 py-2"><Link href={`/creators/${c.address}`} className="text-primary" style={{ fontSize: 12 }}>Profil →</Link></td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Testleri yaz**

```tsx
// CreatorTokenHistoryTable.test.tsx
import { render, screen } from "@testing-library/react";
import { CreatorTokenHistoryTable } from "./CreatorTokenHistoryTable";
import type { CreatorTokenHistoryItem } from "@/lib/api/types";
const h: CreatorTokenHistoryItem = { id: "1", symbol: "PULSE", mint: "m", createdAt: "1g önce", peakMarketCap: 100000, currentMarketCap: 10000, maxDrawdownPct: 90, liquidityStatus: "removed", creatorSellPct: 40, outcome: "rug", riskFlags: [] };
test("renders row with outcome badge and token link", () => {
  render(<CreatorTokenHistoryTable history={[h]} />);
  expect(screen.getByText("Rug")).toBeInTheDocument();
  expect(screen.getByText("PULSE").closest("a")!.getAttribute("href")).toBe("/tokens/PULSE");
});
```

```tsx
// CreatorsList.test.tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { CreatorsList } from "./CreatorsList";
test("lists creators with profile links", async () => {
  render(<QueryClientProvider client={getQueryClient()}><CreatorsList /></QueryClientProvider>);
  await waitFor(() => expect(screen.getAllByText("Profil →").length).toBeGreaterThan(0));
  const link = screen.getAllByText("Profil →")[0].closest("a")!;
  expect(link.getAttribute("href")).toMatch(/^\/creators\//);
});
```

- [ ] **Step 4: GREEN + build** — Run: `npm run test -- creator/CreatorTokenHistoryTable creator/CreatorsList`; PASS. Build OK.
- [ ] **Step 5: Commit**

```bash
git add apps/web/components/creator/CreatorTokenHistoryTable.tsx apps/web/components/creator/CreatorsList.tsx apps/web/components/creator/CreatorTokenHistoryTable.test.tsx apps/web/components/creator/CreatorsList.test.tsx
git commit -m "feat(web): add creator token-history table and creators list"
```

---

### Task 5: CreatorProfileContent + sayfalar + Wallet Graph creator linki (entegrasyon, TDD)

**Files:**
- Create: `apps/web/components/creator/CreatorProfileContent.tsx`, `apps/web/app/(app)/creators/page.tsx`, `apps/web/app/(app)/creators/[address]/page.tsx`
- Modify: `apps/web/components/graph/NodeDetailPanel.tsx` (creator node → link)
- Test: `apps/web/components/creator/CreatorProfileContent.test.tsx`, `apps/web/components/graph/NodeDetailPanel.test.tsx` (güncelle)

**Interfaces:**
- Consumes: `useCreator`, `SCORE_DEFS`, `ScoreCard`, `ExplainableScore` (reuse), CreatorHeader/Metrics/HistoryTable/BehaviorPanel, `getQueryClient/qk/getApi`, `Skeleton`.
- Produces: `<CreatorProfileContent address/>`, `/creators` + `/creators/[address]` pages, graph creator link.

- [ ] **Step 1: `CreatorProfileContent.tsx`** (`"use client"`; reputation ScoreCard+ExplainableScore reuse)

```tsx
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
```

- [ ] **Step 2: `app/(app)/creators/page.tsx`** (server; getCreators prefetch)

```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { CreatorsList } from "@/components/creator/CreatorsList";

export default async function CreatorsPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.creators, queryFn: () => getApi().getCreators() });
  return <HydrationBoundary state={dehydrate(queryClient)}><CreatorsList /></HydrationBoundary>;
}
```

- [ ] **Step 3: `app/(app)/creators/[address]/page.tsx`** (server; getCreator prefetch, params Promise)

```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { CreatorProfileContent } from "@/components/creator/CreatorProfileContent";

export default async function CreatorProfilePage({ params }: { params: Promise<{ address: string }> }) {
  const { address } = await params;
  const queryClient = getQueryClient();
  try { await queryClient.prefetchQuery({ queryKey: qk.creator(address), queryFn: () => getApi().getCreator(address) }); } catch {}
  return <HydrationBoundary state={dehydrate(queryClient)}><CreatorProfileContent address={address} /></HydrationBoundary>;
}
```
(Not: `/creators` altındaki mevcut placeholder `page.tsx` bu gerçek sayfayla değişir.)

- [ ] **Step 4: `components/graph/NodeDetailPanel.tsx` — creator node linki** ekle. Token linkinin yanına: node `creator_wallet` tipindeyse `<Link href={`/creators/${node.address ?? node.id}`} ...>Creator Detayına Git</Link>`. (Mevcut token-linki koşulunu bozmadan ekle.)

- [ ] **Step 5: Testleri yaz/güncelle**

```tsx
// CreatorProfileContent.test.tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { CreatorProfileContent } from "./CreatorProfileContent";
test("renders reputation, metrics, history and behavior", async () => {
  render(<QueryClientProvider client={getQueryClient()}><CreatorProfileContent address="CreAxz" /></QueryClientProvider>);
  await waitFor(() => expect(screen.getByText("Token Geçmişi")).toBeInTheDocument());
  expect(screen.getByText("Davranış Paterni")).toBeInTheDocument();
  expect(screen.getByText("Toplam Token")).toBeInTheDocument();
});
```

```tsx
// NodeDetailPanel.test.tsx — creator node link ekle (mevcut testleri koru)
import type { GraphNode } from "@/lib/api/types";
const creator: GraphNode = { id: "C1", type: "creator_wallet", label: "Creator-A", address: "CreAxz", riskLevel: "high", firstSeen: "x", lastSeen: "y" };
test("creator node shows creator profile link", () => {
  render(<NodeDetailPanel node={creator} graph={graph} />);
  expect(screen.getByText("Creator Detayına Git").closest("a")!.getAttribute("href")).toBe("/creators/CreAxz");
});
```

- [ ] **Step 6: GREEN + build + manuel** — Run: `npm run test`; tüm suite PASS. `npm run build`; `/creators` + `/creators/[address]` derlenir. Manuel (dev): sidebar "Üreticiler" → liste → profil (reputation + metrik + geçmiş + davranış); Wallet Graph creator node → "Creator Detayına Git"; geçmişteki token → Token Detail.
- [ ] **Step 7: Commit**

```bash
git add apps/web/components/creator/CreatorProfileContent.tsx "apps/web/app/(app)/creators" apps/web/components/graph/NodeDetailPanel.tsx apps/web/components/creator/CreatorProfileContent.test.tsx apps/web/components/graph/NodeDetailPanel.test.tsx
git commit -m "feat(web): wire creators list + profile pages and graph creator link"
```

---

## Self-Review (yazar kontrolü)

**Spec coverage:** liste+profil rotaları (T5), seam+hook (T2), MetricTile paylaşımı (T1), outcome/liquidity config (T1), header/metrics/behavior (T3), history table + list (T4), reputation ScoreCard+ExplainableScore reuse (T5), Wallet Graph creator linki (T5), mock import yasağı (kabul #5). ✅

**SOLID + reuse:** DRY (ScoreCard/ExplainableScore/MetricTile/ScoreBadge/WalletAddress reuse), SRP (header/metrics/table/behavior/list ayrı), OCP (OUTCOME_DEFS/LIQUIDITY_DEFS + metrik config), DIP (useCreators/useCreator/getApi), ISP (dar prop'lar). ✅

**Placeholder taraması:** kod gerçek; /creators artık gerçek. ✅

**Tip tutarlılığı:** `CreatorProfile`/`CreatorRow`/`CreatorTokenHistoryItem`/`CreatorBehavior` tek kaynak (types.ts); reputation `ScoreDetail` key `creatorReputation` ∈ SCORE_DEFS (ScoreCard/ExplainableScore reuse sorunsuz); `qk.creators`/`qk.creator` prefetch (T5) + hook (T2) aynı; MetricTile taşındıktan sonra OverviewTab importu güncel. ✅
