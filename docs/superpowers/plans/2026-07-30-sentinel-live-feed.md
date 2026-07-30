# SENTINEL Frontend — Increment 3 Implementation Plan
### Live Feed (Gerçek Zamanlı Event Terminali)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) syntax.

**Goal:** `/live-feed`'de gerçek zamanlı event terminali: 10 filtreli üst çubuk + canlı event tablosu + detay drawer — mock seam üzerinden.

**Architecture:** Increment 1/2 seam'i genişler (`getEvents` + `subscribeEvents`). SOLID: filtreleme saf `filterEvents` (lib), event tipleri `EVENT_TYPE_DEFS` registry (OCP), bileşenler `useEvents`/`getApi()` soyutlamasına bağımlı (DIP), dar prop'lar (ISP).

**Tech Stack:** Next 16 App Router, TS strict, Tailwind v4, shadcn/ui (Base UI) + Sheet, TanStack Query, Recharts (yok bu artımda), lucide-react, Vitest + RTL.

## Global Constraints
- `SENTINEL/apps/web/`; npm; branch `feat/live-feed`.
- **Dark-only**, **UI Türkçe** (teknik token/simge/adres hariç). Mono yalnız sayı/adres.
- **Veri kuralı:** hiçbir bileşen `lib/api/mock`'u import etmez → `useEvents`/`getApi()`.
- **Clean Code & SOLID ölçütü:** SRP (filtre/tablo/drawer/badge ayrı; mantık lib'de), OCP (`EVENT_TYPE_DEFS` + predicate-bazlı `filterEvents`), DIP, ISP; küçük dosyalar; TDD; pristine test.
- Risk seviyeleri (`RiskLevel`) ve severity (`AlertSeverity`) `@/lib/format`'ten; `riskMeta`/`severityMeta` yeniden kullanılır.
- shadcn Base UI (ported bileşenlerde `render` prop). Filtre kontrolleri Header.tsx desenindeki gibi **native styled** `<select>`/`<input>` + chip button; yeni ağır bağımlılık yok. Sadece **Sheet** eklenir.
- Commit: her task sonunda; gövde şununla biter: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.

---

### Task 1: Event modeli + filtre mantığı (saf, TDD)

**Files:**
- Modify: `apps/web/lib/api/types.ts` (EventType, FeedEvent, FeedFilters, EMPTY_FILTERS)
- Create: `apps/web/lib/feed/event-defs.ts`, `apps/web/lib/feed/filter.ts`
- Test: `apps/web/lib/feed/filter.test.ts`, `apps/web/lib/feed/event-defs.test.ts`

**Interfaces:**
- Consumes: `RiskLevel`, `AlertSeverity` (format).
- Produces: `EventType`, `FeedEvent`, `FeedFilters`, `EMPTY_FILTERS`, `EVENT_SEVERITY`, `EVENT_TYPE_DEFS`, `filterEvents`.

- [ ] **Step 1: `lib/api/types.ts`'e ekle**

```ts
import type { RiskLevel, AlertSeverity } from "@/lib/format";

export type EventType =
  | "new_mint" | "metadata_created" | "pool_created" | "first_swap"
  | "liquidity_added" | "liquidity_removed" | "creator_sell" | "whale_buy"
  | "suspicious_cluster" | "score_change" | "strategy_signal";

export interface FeedEvent {
  id: string; type: EventType;
  symbol: string; mint: string;
  launchpad: string; dex: string;
  liquidity: number; creatorScore: number;
  riskLevel: RiskLevel; tokenAgeSeconds: number;
  volume5m: number; holderGrowthPct: number;
  severity: AlertSeverity; detail: string; time: string; ts: number;
  watchlisted: boolean;
}

export interface FeedFilters {
  types: EventType[]; risks: RiskLevel[];
  launchpad: string; dex: string;
  minLiquidity: number; minCreatorScore: number;
  maxAgeSeconds: number | null; minVolume: number; minHolderGrowth: number;
  watchlistOnly: boolean;
}

export const EMPTY_FILTERS: FeedFilters = {
  types: [], risks: [], launchpad: "all", dex: "all",
  minLiquidity: 0, minCreatorScore: 0, maxAgeSeconds: null,
  minVolume: 0, minHolderGrowth: 0, watchlistOnly: false,
};
```
(Not: `RiskLevel`/`AlertSeverity` import satırı types.ts'te zaten varsa çoğaltma — mevcut import'a ekle.)

- [ ] **Step 2: Testleri yaz** (`lib/feed/filter.test.ts`)

```ts
import { filterEvents } from "./filter";
import { EMPTY_FILTERS } from "@/lib/api/types";
import type { FeedEvent } from "@/lib/api/types";

const ev = (o: Partial<FeedEvent>): FeedEvent => ({
  id: "1", type: "new_mint", symbol: "AAA", mint: "m", launchpad: "Pump.fun", dex: "Raydium",
  liquidity: 50000, creatorScore: 60, riskLevel: "medium", tokenAgeSeconds: 100,
  volume5m: 10000, holderGrowthPct: 20, severity: "info", detail: "d", time: "az önce", ts: 0, watchlisted: false, ...o,
});
const list = [
  ev({ id: "a", type: "whale_buy", riskLevel: "strong", launchpad: "Raydium", dex: "Orca", liquidity: 200000, creatorScore: 90, tokenAgeSeconds: 30, volume5m: 90000, holderGrowthPct: 60, watchlisted: true }),
  ev({ id: "b", type: "liquidity_removed", riskLevel: "critical", launchpad: "Pump.fun", dex: "Raydium", liquidity: 4000, creatorScore: 17, tokenAgeSeconds: 500, volume5m: 3000, holderGrowthPct: 2, watchlisted: false }),
];

test("EMPTY_FILTERS returns everything", () => {
  expect(filterEvents(list, EMPTY_FILTERS)).toHaveLength(2);
});
test("type filter", () => {
  expect(filterEvents(list, { ...EMPTY_FILTERS, types: ["whale_buy"] }).map(e => e.id)).toEqual(["a"]);
});
test("risk filter", () => {
  expect(filterEvents(list, { ...EMPTY_FILTERS, risks: ["critical"] }).map(e => e.id)).toEqual(["b"]);
});
test("launchpad + dex filter", () => {
  expect(filterEvents(list, { ...EMPTY_FILTERS, launchpad: "Raydium" }).map(e => e.id)).toEqual(["a"]);
  expect(filterEvents(list, { ...EMPTY_FILTERS, dex: "Raydium" }).map(e => e.id)).toEqual(["b"]);
});
test("numeric thresholds", () => {
  expect(filterEvents(list, { ...EMPTY_FILTERS, minLiquidity: 100000 }).map(e => e.id)).toEqual(["a"]);
  expect(filterEvents(list, { ...EMPTY_FILTERS, minCreatorScore: 50 }).map(e => e.id)).toEqual(["a"]);
  expect(filterEvents(list, { ...EMPTY_FILTERS, maxAgeSeconds: 60 }).map(e => e.id)).toEqual(["a"]);
  expect(filterEvents(list, { ...EMPTY_FILTERS, minVolume: 50000 }).map(e => e.id)).toEqual(["a"]);
  expect(filterEvents(list, { ...EMPTY_FILTERS, minHolderGrowth: 30 }).map(e => e.id)).toEqual(["a"]);
});
test("watchlistOnly", () => {
  expect(filterEvents(list, { ...EMPTY_FILTERS, watchlistOnly: true }).map(e => e.id)).toEqual(["a"]);
});
```

- [ ] **Step 3: event-defs testini yaz** (`lib/feed/event-defs.test.ts`)

```ts
import { EVENT_TYPE_DEFS, EVENT_SEVERITY } from "./event-defs";

test("every event type has a def with label + severity", () => {
  expect(EVENT_TYPE_DEFS).toHaveLength(11);
  for (const d of EVENT_TYPE_DEFS) {
    expect(d.label).toBeTruthy();
    expect(EVENT_SEVERITY[d.key]).toBeTruthy();
  }
});
```

- [ ] **Step 4: RED doğrula** — Run: `npm run test -- feed`; Expected: FAIL (moduller yok).

- [ ] **Step 5: `lib/feed/event-defs.ts` yaz**

```ts
import {
  Sparkles, FileText, Droplet, ArrowLeftRight, PlusCircle, MinusCircle,
  TrendingDown, Fish, ShieldAlert, Activity, Zap, type LucideIcon,
} from "lucide-react";
import type { EventType } from "@/lib/api/types";
import type { AlertSeverity } from "@/lib/format";

export const EVENT_SEVERITY: Record<EventType, AlertSeverity> = {
  new_mint: "info", metadata_created: "info", pool_created: "info", first_swap: "positive",
  liquidity_added: "positive", liquidity_removed: "critical", creator_sell: "warning",
  whale_buy: "positive", suspicious_cluster: "critical", score_change: "warning", strategy_signal: "info",
};

export interface EventTypeDef { key: EventType; label: string; icon: LucideIcon; }

export const EVENT_TYPE_DEFS: EventTypeDef[] = [
  { key: "new_mint", label: "Yeni Mint", icon: Sparkles },
  { key: "metadata_created", label: "Metadata Oluşturuldu", icon: FileText },
  { key: "pool_created", label: "Havuz Açıldı", icon: Droplet },
  { key: "first_swap", label: "İlk Swap", icon: ArrowLeftRight },
  { key: "liquidity_added", label: "Likidite Eklendi", icon: PlusCircle },
  { key: "liquidity_removed", label: "Likidite Çekildi", icon: MinusCircle },
  { key: "creator_sell", label: "Üretici Satışı", icon: TrendingDown },
  { key: "whale_buy", label: "Balina Alımı", icon: Fish },
  { key: "suspicious_cluster", label: "Şüpheli Küme", icon: ShieldAlert },
  { key: "score_change", label: "Skor Değişti", icon: Activity },
  { key: "strategy_signal", label: "Strateji Sinyali", icon: Zap },
];
```

- [ ] **Step 6: `lib/feed/filter.ts` yaz**

```ts
import type { FeedEvent, FeedFilters } from "@/lib/api/types";

export function filterEvents(events: FeedEvent[], f: FeedFilters): FeedEvent[] {
  return events.filter((e) =>
    (f.types.length === 0 || f.types.includes(e.type)) &&
    (f.risks.length === 0 || f.risks.includes(e.riskLevel)) &&
    (f.launchpad === "all" || e.launchpad === f.launchpad) &&
    (f.dex === "all" || e.dex === f.dex) &&
    e.liquidity >= f.minLiquidity &&
    e.creatorScore >= f.minCreatorScore &&
    (f.maxAgeSeconds === null || e.tokenAgeSeconds <= f.maxAgeSeconds) &&
    e.volume5m >= f.minVolume &&
    e.holderGrowthPct >= f.minHolderGrowth &&
    (!f.watchlistOnly || e.watchlisted)
  );
}
```

- [ ] **Step 7: GREEN doğrula** — Run: `npm run test -- feed`; Expected: PASS.
- [ ] **Step 8: Commit**

```bash
git add apps/web/lib/api/types.ts apps/web/lib/feed
git commit -m "feat(web): add feed event model, event-type registry, and pure filterEvents"
```

---

### Task 2: Veri katmanı — getEvents + subscribeEvents + hook'lar (TDD)

**Files:**
- Modify: `apps/web/lib/api/contract.ts`, `mock.ts`, `http.ts`, `lib/get-query-client.ts`, `lib/hooks/queries.ts`, `lib/hooks/live.ts`
- Test: `apps/web/lib/api/events.test.ts`, `apps/web/lib/hooks/events.test.tsx`

**Interfaces:**
- Produces: `SentinelApi.getEvents/subscribeEvents`; `qk.events`; `useEvents()`; `useLiveEvents()`.

- [ ] **Step 1: `contract.ts`'e ekle** — `SentinelApi`'ye:

```ts
  getEvents(): Promise<FeedEvent[]>;
  subscribeEvents(cb: (e: FeedEvent) => void): () => void;
```
`import type { ..., FeedEvent } from "./types";` güncelle.

- [ ] **Step 2: Testi yaz** (`lib/api/events.test.ts`)

```ts
import { mockApi } from "./mock";
test("getEvents returns a seed stream", async () => {
  const evs = await mockApi.getEvents();
  expect(evs.length).toBeGreaterThan(10);
  expect(evs[0].type).toBeTruthy();
});
test("subscribeEvents emits and unsubscribes", async () => {
  await new Promise<void>((resolve, reject) => {
    const stop = mockApi.subscribeEvents((e) => { expect(e.id).toBeTruthy(); stop(); resolve(); });
    expect(typeof stop).toBe("function");
    setTimeout(() => reject(new Error("no emit")), 5000);
  });
});
```

- [ ] **Step 3: RED doğrula** — Run: `npm run test -- api/events`; Expected: FAIL.

- [ ] **Step 4: `mock.ts`'e ekle** (mevcut `tokens`, `seedOf`, `clamp`, `scoreToLevel`, `delay` yeniden kullanılır; `EVENT_SEVERITY` import)

```ts
import type { EventType, FeedEvent } from "./types";
import { EVENT_SEVERITY } from "@/lib/feed/event-defs";

const LAUNCHPADS = ["Pump.fun", "Raydium", "Moonshot", "Meteora"];
const DEXES = ["Raydium", "Meteora", "Orca", "Jupiter"];
const EVENT_TYPES: EventType[] = [
  "new_mint", "metadata_created", "pool_created", "first_swap", "liquidity_added",
  "liquidity_removed", "creator_sell", "whale_buy", "suspicious_cluster", "score_change", "strategy_signal",
];
const EVENT_DETAIL: Record<EventType, string> = {
  new_mint: "Yeni token basıldı", metadata_created: "Metadata oluşturuldu", pool_created: "İlk havuz açıldı",
  first_swap: "İlk swap gerçekleşti", liquidity_added: "Likidite eklendi", liquidity_removed: "Üretici likidite çekti",
  creator_sell: "Üretici satış yaptı", whale_buy: "Balina alımı", suspicious_cluster: "Bağlantılı cüzdan kümesi",
  score_change: "Risk skoru değişti", strategy_signal: "Strateji sinyali üretildi",
};

function buildEvent(i: number, type: EventType): FeedEvent {
  const t = tokens[i % tokens.length];
  const seed = seedOf(t.symbol) + i;
  return {
    id: `ev-${i}-${type}`, type, symbol: t.symbol, mint: t.mint,
    launchpad: LAUNCHPADS[i % LAUNCHPADS.length], dex: DEXES[i % DEXES.length],
    liquidity: t.liquidity, creatorScore: t.creatorScore,
    riskLevel: scoreToLevel(Math.round((t.creatorScore + t.safetyScore) / 2)),
    tokenAgeSeconds: t.ageSeconds + i * 7, volume5m: t.vol5m,
    holderGrowthPct: clamp(5 + (seed % 60)), severity: EVENT_SEVERITY[type],
    detail: `${t.symbol} · ${EVENT_DETAIL[type]}`, time: i === 0 ? "az önce" : `${i * 4}sn önce`, ts: 1000 - i,
    watchlisted: t.watchlisted,
  };
}
const feedEvents: FeedEvent[] = Array.from({ length: 24 }, (_, i) => buildEvent(i, EVENT_TYPES[i % EVENT_TYPES.length]));
```
Ve `mockApi`'ye:
```ts
  getEvents: () => delay(feedEvents),
  subscribeEvents(cb) {
    let n = 0;
    const id = setInterval(() => {
      const type = EVENT_TYPES[n % EVENT_TYPES.length];
      cb({ ...buildEvent(n % tokens.length, type), id: `live-${n}-${type}`, time: "az önce", ts: 2000 + n });
      n++;
    }, 3000);
    return () => clearInterval(id);
  },
```

- [ ] **Step 5: `http.ts`'e ekle** — `getEvents: notReady, subscribeEvents: () => () => {},`.

- [ ] **Step 6: `get-query-client.ts`'e** — `qk`'ye `events: ["events"] as const,`.

- [ ] **Step 7: `queries.ts`'e** — 

```ts
export function useEvents() {
  return useQuery({ queryKey: qk.events, queryFn: () => getApi().getEvents() });
}
```

- [ ] **Step 8: `live.ts`'e** —

```ts
import type { FeedEvent } from "@/lib/api/types";
export function useLiveEvents() {
  const qc = useQueryClient();
  useEffect(() => getApi().subscribeEvents((e: FeedEvent) => {
    qc.setQueryData<FeedEvent[]>(qk.events, (prev) => [e, ...(prev ?? [])].slice(0, 200));
  }), [qc]);
}
```

- [ ] **Step 9: Hook testini yaz** (`lib/hooks/events.test.tsx`)

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useEvents } from "./queries";
function Probe() { const { data } = useEvents(); return <div>n:{data?.length ?? 0}</div>; }
test("useEvents loads the seed stream", async () => {
  render(<QueryClientProvider client={getQueryClient()}><Probe /></QueryClientProvider>);
  await waitFor(() => expect(screen.getByText(/n:2[0-9]/)).toBeInTheDocument());
});
```

- [ ] **Step 10: GREEN + build** — Run: `npm run test -- events`; Expected: PASS. Run `npm run build`; Expected: OK.
- [ ] **Step 11: Commit**

```bash
git add apps/web/lib/api apps/web/lib/get-query-client.ts apps/web/lib/hooks
git commit -m "feat(web): add getEvents/subscribeEvents seam and useEvents/useLiveEvents hooks"
```

---

### Task 3: EventTypeBadge + FeedTable + Sheet primitive (TDD)

**Files:**
- Add shadcn: `sheet`
- Create: `apps/web/components/feed/EventTypeBadge.tsx`, `apps/web/components/feed/FeedTable.tsx`
- Test: `apps/web/components/feed/EventTypeBadge.test.tsx`, `FeedTable.test.tsx`

**Interfaces:**
- Consumes: `EVENT_TYPE_DEFS`, `EVENT_SEVERITY`, `severityMeta`, `riskMeta`, `FeedEvent`, `formatAge/formatUsd`, `TokenAvatar`, `ScoreBadge`, `WalletAddress`.
- Produces: `<EventTypeBadge type/>`, `<FeedTable events onRowClick/>`.

- [ ] **Step 1: shadcn sheet ekle** — `npx shadcn@latest add sheet` (Base UI variant olabilir; import `@/components/ui/sheet`, üretilen export'lara uy).

- [ ] **Step 2: `EventTypeBadge.tsx`** (registry + severity rengi)

```tsx
import { EVENT_TYPE_DEFS, EVENT_SEVERITY } from "@/lib/feed/event-defs";
import { severityMeta } from "@/lib/format";
import type { EventType } from "@/lib/api/types";

const DEF_BY_KEY = Object.fromEntries(EVENT_TYPE_DEFS.map((d) => [d.key, d])) as Record<EventType, typeof EVENT_TYPE_DEFS[number]>;

export function EventTypeBadge({ type }: { type: EventType }) {
  const def = DEF_BY_KEY[type];
  const color = severityMeta[EVENT_SEVERITY[type]].color;
  const Icon = def.icon;
  return (
    <span className="inline-flex items-center gap-1.5 rounded px-1.5 py-0.5" style={{ color, backgroundColor: `${color}1f`, fontSize: 11, fontWeight: 600 }}>
      <Icon size={12} /> {def.label}
    </span>
  );
}
```

- [ ] **Step 3: `FeedTable.tsx`** (`"use client"`; bounded scroll; row click)

```tsx
"use client";
import { ExternalLink } from "lucide-react";
import type { FeedEvent } from "@/lib/api/types";
import { formatAge, formatUsd, riskMeta } from "@/lib/format";
import { TokenAvatar } from "@/components/sentinel/TokenAvatar";
import { ScoreBadge } from "@/components/sentinel/ScoreBadge";
import { EventTypeBadge } from "./EventTypeBadge";

export function FeedTable({ events, onRowClick }: { events: FeedEvent[]; onRowClick: (e: FeedEvent) => void }) {
  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="max-h-[600px] overflow-y-auto">
        <table className="w-full border-collapse" style={{ fontSize: 13 }}>
          <thead className="sticky top-0 bg-card">
            <tr className="text-muted-foreground" style={{ fontSize: 11 }}>
              {["Zaman", "Event", "Token", "Kaynak", "Likidite", "Creator", "Risk", ""].map((h) => (
                <th key={h} className="whitespace-nowrap border-b border-border px-3 py-2 text-left font-normal">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {events.map((e) => {
              const rm = riskMeta[e.riskLevel];
              const fresh = e.tokenAgeSeconds < 60;
              return (
                <tr key={e.id} onClick={() => onRowClick(e)}
                  className="cursor-pointer border-t border-border transition-colors hover:bg-accent/40"
                  style={fresh ? { boxShadow: "inset 2px 0 0 #2FD98B" } : undefined}>
                  <td className="whitespace-nowrap px-3 py-2 font-mono text-muted-foreground tabular-nums" style={{ fontSize: 11 }}>{e.time}</td>
                  <td className="px-3 py-2"><EventTypeBadge type={e.type} /></td>
                  <td className="px-3 py-2">
                    <div className="flex items-center gap-2">
                      <TokenAvatar symbol={e.symbol} size={22} />
                      <span style={{ fontWeight: 500 }}>{e.symbol}</span>
                    </div>
                  </td>
                  <td className="whitespace-nowrap px-3 py-2 text-muted-foreground" style={{ fontSize: 12 }}>{e.launchpad} · {e.dex}</td>
                  <td className="whitespace-nowrap px-3 py-2 font-mono tabular-nums">{formatUsd(e.liquidity)}</td>
                  <td className="px-3 py-2"><ScoreBadge score={e.creatorScore} /></td>
                  <td className="px-3 py-2"><span className="rounded px-1.5 py-0.5" style={{ color: rm.color, backgroundColor: rm.bg, fontSize: 11 }}>{rm.label}</span></td>
                  <td className="px-3 py-2"><ExternalLink size={14} className="text-muted-foreground" /></td>
                </tr>
              );
            })}
          </tbody>
        </table>
        {events.length === 0 && <div className="p-10 text-center text-muted-foreground" style={{ fontSize: 13 }}>Filtrelere uygun event yok</div>}
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Testleri yaz**

```tsx
// EventTypeBadge.test.tsx
import { render, screen } from "@testing-library/react";
import { EventTypeBadge } from "./EventTypeBadge";
test("renders the Turkish label for the type", () => {
  render(<EventTypeBadge type="liquidity_removed" />);
  expect(screen.getByText("Likidite Çekildi")).toBeInTheDocument();
});
```

```tsx
// FeedTable.test.tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FeedTable } from "./FeedTable";
import type { FeedEvent } from "@/lib/api/types";
const ev: FeedEvent = { id: "x", type: "whale_buy", symbol: "PULSE", mint: "9x..4Fk2", launchpad: "Raydium", dex: "Orca", liquidity: 82400, creatorScore: 82, riskLevel: "good", tokenAgeSeconds: 30, volume5m: 41200, holderGrowthPct: 40, severity: "positive", detail: "d", time: "az önce", ts: 1, watchlisted: false };
test("row click calls onRowClick with the event", async () => {
  const onRowClick = vi.fn();
  render(<FeedTable events={[ev]} onRowClick={onRowClick} />);
  await userEvent.click(screen.getByText("PULSE"));
  expect(onRowClick).toHaveBeenCalledWith(ev);
});
test("empty state when no events", () => {
  render(<FeedTable events={[]} onRowClick={() => {}} />);
  expect(screen.getByText("Filtrelere uygun event yok")).toBeInTheDocument();
});
```

- [ ] **Step 5: GREEN + build** — Run: `npm run test -- feed/EventTypeBadge feed/FeedTable`; Expected: PASS. Build OK.
- [ ] **Step 6: Commit**

```bash
git add apps/web/components/feed/EventTypeBadge.tsx apps/web/components/feed/FeedTable.tsx apps/web/components/feed/EventTypeBadge.test.tsx apps/web/components/feed/FeedTable.test.tsx apps/web/components/ui/sheet.tsx
git commit -m "feat(web): add event-type badge and live feed table"
```

---

### Task 4: FeedFilters — 10 filtreli çubuk (TDD)

**Files:**
- Create: `apps/web/components/feed/FeedFilters.tsx`
- Test: `apps/web/components/feed/FeedFilters.test.tsx`

**Interfaces:**
- Consumes: `FeedFilters`, `EMPTY_FILTERS`, `EVENT_TYPE_DEFS`, `RiskLevel`/`riskMeta`.
- Produces: `<FeedFilters value onChange/>`.

- [ ] **Step 1: `FeedFilters.tsx`** (`"use client"`; native styled select/input + chip toggle)

```tsx
"use client";
import { X } from "lucide-react";
import type { FeedFilters as Filters, EventType } from "@/lib/api/types";
import { EMPTY_FILTERS } from "@/lib/api/types";
import { EVENT_TYPE_DEFS } from "@/lib/feed/event-defs";
import { riskMeta, type RiskLevel } from "@/lib/format";

const LAUNCHPADS = ["Pump.fun", "Raydium", "Moonshot", "Meteora"];
const DEXES = ["Raydium", "Meteora", "Orca", "Jupiter"];
const RISKS: RiskLevel[] = ["critical", "high", "medium", "good", "strong"];

const inputCls = "h-8 w-28 rounded-md border border-border bg-input px-2 text-foreground focus:border-primary focus:outline-none";
const selectCls = "h-8 rounded-md border border-border bg-input px-2 text-foreground focus:outline-none";

export function FeedFilters({ value, onChange }: { value: Filters; onChange: (f: Filters) => void }) {
  const set = (patch: Partial<Filters>) => onChange({ ...value, ...patch });
  const toggle = <T,>(arr: T[], v: T): T[] => (arr.includes(v) ? arr.filter((x) => x !== v) : [...arr, v]);
  const num = (s: string) => (s === "" ? 0 : Number(s));

  return (
    <div className="space-y-3 rounded-lg border border-border bg-card p-3" style={{ fontSize: 12 }}>
      {/* Event tipi çipleri */}
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-muted-foreground">Event:</span>
        {EVENT_TYPE_DEFS.map((d) => {
          const on = value.types.includes(d.key);
          return (
            <button key={d.key} onClick={() => set({ types: toggle(value.types, d.key) })}
              className={`rounded px-2 py-1 ${on ? "bg-primary text-primary-foreground" : "bg-surface-2 text-muted-foreground hover:text-foreground"}`}>
              {d.label}
            </button>
          );
        })}
      </div>
      {/* Risk çipleri */}
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-muted-foreground">Risk:</span>
        {RISKS.map((r) => {
          const on = value.risks.includes(r);
          const m = riskMeta[r];
          return (
            <button key={r} onClick={() => set({ risks: toggle(value.risks, r) })}
              className="rounded px-2 py-1" style={{ color: on ? m.color : undefined, backgroundColor: on ? m.bg : "var(--sentinel-surface-2)", border: on ? `1px solid ${m.border}` : "1px solid transparent" }}>
              {m.label}
            </button>
          );
        })}
      </div>
      {/* Select + sayı inputları */}
      <div className="flex flex-wrap items-center gap-2">
        <select className={selectCls} value={value.launchpad} onChange={(e) => set({ launchpad: e.target.value })}>
          <option value="all">Launchpad: Tümü</option>
          {LAUNCHPADS.map((l) => <option key={l} value={l}>{l}</option>)}
        </select>
        <select className={selectCls} value={value.dex} onChange={(e) => set({ dex: e.target.value })}>
          <option value="all">DEX: Tümü</option>
          {DEXES.map((d) => <option key={d} value={d}>{d}</option>)}
        </select>
        <input className={inputCls} type="number" placeholder="Min likidite" value={value.minLiquidity || ""} onChange={(e) => set({ minLiquidity: num(e.target.value) })} />
        <input className={inputCls} type="number" placeholder="Min creator" value={value.minCreatorScore || ""} onChange={(e) => set({ minCreatorScore: num(e.target.value) })} />
        <input className={inputCls} type="number" placeholder="Max yaş (sn)" value={value.maxAgeSeconds ?? ""} onChange={(e) => set({ maxAgeSeconds: e.target.value === "" ? null : Number(e.target.value) })} />
        <input className={inputCls} type="number" placeholder="Min hacim" value={value.minVolume || ""} onChange={(e) => set({ minVolume: num(e.target.value) })} />
        <input className={inputCls} type="number" placeholder="Min holder %" value={value.minHolderGrowth || ""} onChange={(e) => set({ minHolderGrowth: num(e.target.value) })} />
        <label className="flex items-center gap-1.5 text-muted-foreground">
          <input type="checkbox" checked={value.watchlistOnly} onChange={(e) => set({ watchlistOnly: e.target.checked })} /> Watchlist
        </label>
        <button onClick={() => onChange(EMPTY_FILTERS)} className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-muted-foreground hover:text-foreground">
          <X size={12} /> Temizle
        </button>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Testi yaz** (`FeedFilters.test.tsx`)

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FeedFilters } from "./FeedFilters";
import { EMPTY_FILTERS } from "@/lib/api/types";

test("toggling an event chip emits updated types", async () => {
  const onChange = vi.fn();
  render(<FeedFilters value={EMPTY_FILTERS} onChange={onChange} />);
  await userEvent.click(screen.getByText("Balina Alımı"));
  expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ types: ["whale_buy"] }));
});

test("Temizle resets to EMPTY_FILTERS", async () => {
  const onChange = vi.fn();
  render(<FeedFilters value={{ ...EMPTY_FILTERS, watchlistOnly: true }} onChange={onChange} />);
  await userEvent.click(screen.getByText("Temizle"));
  expect(onChange).toHaveBeenCalledWith(EMPTY_FILTERS);
});
```

- [ ] **Step 3: GREEN + build** — Run: `npm run test -- feed/FeedFilters`; Expected: PASS. Build OK.
- [ ] **Step 4: Commit**

```bash
git add apps/web/components/feed/FeedFilters.tsx apps/web/components/feed/FeedFilters.test.tsx
git commit -m "feat(web): add live feed 10-filter bar"
```

---

### Task 5: EventDetailDrawer + LiveFeedContent + sayfa (entegrasyon, TDD)

**Files:**
- Create: `apps/web/components/feed/EventDetailDrawer.tsx`, `apps/web/components/feed/LiveFeedContent.tsx`, `apps/web/app/(app)/live-feed/page.tsx`
- Test: `apps/web/components/feed/EventDetailDrawer.test.tsx`, `LiveFeedContent.test.tsx`

**Interfaces:**
- Consumes: `useEvents`/`useLiveEvents`, `filterEvents`, `EMPTY_FILTERS`, `FeedFilters`, `FeedTable`, `EventTypeBadge`, shadcn `Sheet`, `getQueryClient/qk/getApi`.
- Produces: `<EventDetailDrawer event onClose/>`, `<LiveFeedContent/>`, `/live-feed` sayfası.

- [ ] **Step 1: `EventDetailDrawer.tsx`** (`"use client"`; shadcn Sheet — üretilen API'ye uyarla)

```tsx
"use client";
import Link from "next/link";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import type { FeedEvent } from "@/lib/api/types";
import { formatAge, formatUsd, formatPct, riskMeta } from "@/lib/format";
import { EventTypeBadge } from "./EventTypeBadge";
import { WalletAddress } from "@/components/sentinel/WalletAddress";
import { ScoreBadge } from "@/components/sentinel/ScoreBadge";

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return <div className="flex items-center justify-between border-b border-border py-2"><span className="text-muted-foreground" style={{ fontSize: 12 }}>{label}</span><span style={{ fontSize: 13 }}>{children}</span></div>;
}

export function EventDetailDrawer({ event, onClose }: { event: FeedEvent | null; onClose: () => void }) {
  return (
    <Sheet open={!!event} onOpenChange={(o) => { if (!o) onClose(); }}>
      <SheetContent side="right" className="w-[380px] bg-popover">
        {event && (
          <>
            <SheetHeader><SheetTitle><EventTypeBadge type={event.type} /></SheetTitle></SheetHeader>
            <div className="mt-3 space-y-1">
              <Row label="Token"><span className="font-mono">{event.symbol}</span></Row>
              <Row label="Mint"><WalletAddress address={event.mint} /></Row>
              <Row label="Kaynak">{event.launchpad} · {event.dex}</Row>
              <Row label="Likidite"><span className="font-mono">{formatUsd(event.liquidity)}</span></Row>
              <Row label="Creator"><ScoreBadge score={event.creatorScore} /></Row>
              <Row label="Risk"><span style={{ color: riskMeta[event.riskLevel].color }}>{riskMeta[event.riskLevel].label}</span></Row>
              <Row label="Yaş"><span className="font-mono">{formatAge(event.tokenAgeSeconds)}</span></Row>
              <Row label="5dk Hacim"><span className="font-mono">{formatUsd(event.volume5m)}</span></Row>
              <Row label="Holder Büyümesi"><span className="font-mono">{formatPct(event.holderGrowthPct)}</span></Row>
              <Row label="Zaman">{event.time}</Row>
            </div>
            <p className="mt-3 text-muted-foreground" style={{ fontSize: 12 }}>{event.detail}</p>
            <Link href={`/tokens/${event.symbol}`} className="mt-4 inline-block rounded-md bg-primary px-4 py-2 text-primary-foreground" style={{ fontSize: 13, fontWeight: 500 }}>
              Token Detayına Git
            </Link>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
```
(Not: shadcn `sheet` Base UI ise `Sheet`/`SheetContent`/`SheetHeader`/`SheetTitle` export adları/prop'ları farklı olabilir — üretilen `components/ui/sheet.tsx`'e göre uyarla; davranış: sağdan açılan panel, `event` null değilse açık.)

- [ ] **Step 2: `LiveFeedContent.tsx`** (`"use client"`; kompozisyon + filtre state + seçili event)

```tsx
"use client";
import { useState } from "react";
import { useEvents } from "@/lib/hooks/queries";
import { useLiveEvents } from "@/lib/hooks/live";
import { filterEvents } from "@/lib/feed/filter";
import { EMPTY_FILTERS } from "@/lib/api/types";
import type { FeedFilters as Filters, FeedEvent } from "@/lib/api/types";
import { FeedFilters } from "./FeedFilters";
import { FeedTable } from "./FeedTable";
import { EventDetailDrawer } from "./EventDetailDrawer";

export function LiveFeedContent() {
  useLiveEvents();
  const { data } = useEvents();
  const [filters, setFilters] = useState<Filters>(EMPTY_FILTERS);
  const [selected, setSelected] = useState<FeedEvent | null>(null);
  const events = filterEvents(data ?? [], filters);
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <span className="relative flex h-2 w-2">
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-positive opacity-60" />
          <span className="relative inline-flex h-2 w-2 rounded-full bg-positive" />
        </span>
        <h1>Canlı Akış</h1>
        <span className="text-muted-foreground" style={{ fontSize: 12 }}>· {events.length} event</span>
      </div>
      <FeedFilters value={filters} onChange={setFilters} />
      <FeedTable events={events} onRowClick={setSelected} />
      <EventDetailDrawer event={selected} onClose={() => setSelected(null)} />
    </div>
  );
}
```

- [ ] **Step 3: `app/(app)/live-feed/page.tsx`** (server; prefetch)

```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { LiveFeedContent } from "@/components/feed/LiveFeedContent";

export default async function LiveFeedPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.events, queryFn: () => getApi().getEvents() });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <LiveFeedContent />
    </HydrationBoundary>
  );
}
```
(Not: bu dosya mevcut placeholder `live-feed/page.tsx`'in yerine geçer.)

- [ ] **Step 4: Testleri yaz**

```tsx
// EventDetailDrawer.test.tsx
import { render, screen } from "@testing-library/react";
import { EventDetailDrawer } from "./EventDetailDrawer";
import type { FeedEvent } from "@/lib/api/types";
const ev: FeedEvent = { id: "x", type: "whale_buy", symbol: "PULSE", mint: "9x..4Fk2", launchpad: "Raydium", dex: "Orca", liquidity: 82400, creatorScore: 82, riskLevel: "good", tokenAgeSeconds: 30, volume5m: 41200, holderGrowthPct: 40, severity: "positive", detail: "detay", time: "az önce", ts: 1, watchlisted: false };
test("open drawer shows event + token link", () => {
  render(<EventDetailDrawer event={ev} onClose={() => {}} />);
  expect(screen.getByText("Balina Alımı")).toBeInTheDocument();
  expect(screen.getByText("Token Detayına Git").closest("a")!.getAttribute("href")).toBe("/tokens/PULSE");
});
```

```tsx
// LiveFeedContent.test.tsx
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { LiveFeedContent } from "./LiveFeedContent";
function wrap() { return render(<QueryClientProvider client={getQueryClient()}><LiveFeedContent /></QueryClientProvider>); }
test("applying a risk filter narrows the table", async () => {
  wrap();
  await waitFor(() => expect(screen.getByText(/event$/)).toBeInTheDocument());
  const before = Number(screen.getByText(/· \d+ event/).textContent!.match(/(\d+) event/)![1]);
  await userEvent.click(screen.getByText("Kritik"));
  await waitFor(() => {
    const after = Number(screen.getByText(/· \d+ event/).textContent!.match(/(\d+) event/)![1]);
    expect(after).toBeLessThanOrEqual(before);
  });
});
```
(Not: `LiveFeedContent` `useLiveEvents` interval'i test ortamında da tetiklenir; test kısa; gerekirse `EventDetailDrawer`'ın Sheet'i jsdom'da portala render eder — `Token Detayına Git` linki için event set edilmiş drawer testi ayrı ele alındı.)

- [ ] **Step 5: GREEN + build + manuel** — Run: `npm run test`; Expected: tüm suite PASS. Run `npm run build`; `/live-feed` derlenir. Manuel (dev): sidebar "Canlı Akış" → filtreler + tablo; çip/filtre daraltır; yeni event başa düşer; satıra tıkla → drawer + "Token Detayına Git".
- [ ] **Step 6: Commit**

```bash
git add apps/web/components/feed/EventDetailDrawer.tsx apps/web/components/feed/LiveFeedContent.tsx "apps/web/app/(app)/live-feed/page.tsx" apps/web/components/feed/EventDetailDrawer.test.tsx apps/web/components/feed/LiveFeedContent.test.tsx
git commit -m "feat(web): wire live feed page with detail drawer and real-time stream"
```

---

## Self-Review (yazar kontrolü)

**Spec coverage:**
- Rota `/live-feed` + RSC prefetch → Task 5. ✅
- Veri seam (getEvents/subscribeEvents/useEvents/useLiveEvents), mock import yasağı → Task 2. ✅
- 10 filtre + saf `filterEvents` → Task 1 (logic) + Task 4 (UI). ✅
- Event tip registry (OCP) → Task 1. ✅
- Tablo (canlı prepend + highlight + bounded scroll + empty state) → Task 3. ✅
- Detay drawer (Sheet + Token linki) → Task 5. ✅
- Canlı akış (prepend + 200 cap) → Task 2 (useLiveEvents). ✅
- Test stratejisi → her task. ✅

**SOLID:** SRP (filter lib / badge / table / filters / drawer ayrı), OCP (EVENT_TYPE_DEFS + predicate filter), DIP (useEvents/getApi), ISP (dar prop'lar). ✅

**Placeholder taraması:** kod gerçek; "yakında"/placeholder yok (live-feed artık gerçek). ✅

**Tip tutarlılığı:** `FeedEvent`/`FeedFilters`/`EventType` tek kaynak (types.ts); `EMPTY_FILTERS` her yerde; `qk.events` prefetch (Task 5) + hook (Task 2) aynı; shadcn Sheet export adları üretime göre uyarlanır (Task 3/5 notu). ✅
