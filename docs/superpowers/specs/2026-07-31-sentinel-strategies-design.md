# SENTINEL Frontend — Increment 6 Design Spec
### Strategies (Liste + Read-only Detay)

- **Tarih:** 2026-07-31
- **Durum:** Onaylandı (2026-07-31) — plan yazıldı: `docs/superpowers/plans/2026-07-31-sentinel-frontend-increment-6-strategies.md`
- **Önkoşul:** Increment 1–5 master'da.
- **Kaynaklar:** `docs/design/sentinel-ui-ux-design.md` (Ekran 6), `ROADMAP.md` (§8 strateji platformu), `docs/progress.md`.

---

## 1. Amaç ve kapsam

Strateji yönetim ekranı: strateji kartları listesi + read-only strateji detayı. Bu artımda:
- **`/strategies`** kart listesi (durum/mode/win rate/profit factor/drawdown/net PnL) + durum filtresi.
- **`/strategies/[id]`** read-only detay: IF/THEN entry/exit conditions, risk kuralları, position sizing, performans metrikleri, **equity curve (Recharts)**, backtest özeti, supported launchpads/min scores, version history, audit log.

**Kapsam dışı (bilinçli — sonraki artım):** "Create Strategy" no-code condition builder stepper (8 adım), strateji düzenleme/deploy, gerçek execution. Bu artım salt **görüntüleme**.

---

## 2. Clean Code & SOLID + reuse (ölçüt)

- **Reuse:** performans + backtest metrikleri `MetricTile` (paylaşımlı); equity curve Recharts (OverviewTab MiniChart deseni); durum/renk `riskMeta` paletiyle uyumlu.
- **SRP:** `StrategyCard`, `ConditionList`, `StrategyPerformancePanel`, `EquityCurve`, `BacktestSummaryPanel`, `VersionHistory`, `AuditLog` ayrı; her biri tek iş.
- **OCP:** durumlar `STATUS_DEFS` registry; koşul metrik etiketleri `CONDITION_LABELS` config → yeni durum/metrik = entry.
- **DIP:** bileşenler `useStrategies`/`useStrategy`/`getApi()`; mock import etmez.
- **ISP:** `ConditionList` yalnız `StrategyCondition[]`; `EquityCurve` yalnız noktalar; dar prop'lar.

---

## 3. Mimari

### 3.1 Veri seam genişlemesi (`lib/api`)
```ts
export type StrategyStatus = "draft" | "backtesting" | "paper" | "shadow" | "live" | "paused" | "archived";
export type ConditionOp = ">" | "<" | ">=" | "<=" | "==";

export interface StrategyCondition { metric: string; op: ConditionOp; value: number; unit?: string; }
export interface StrategyRow {
  id: string; name: string; status: StrategyStatus; timeframe: string;
  winRatePct: number; profitFactor: number; maxDrawdownPct: number; totalTrades: number;
  netPnlSol: number; lastSignal: string;
}
export interface StrategyPerformance {
  winRatePct: number; profitFactor: number; maxDrawdownPct: number; sharpe: number; sortino: number;
  totalTrades: number; netPnlSol: number; expectancy: number;
}
export interface EquityPoint { t: number; v: number; }
export interface BacktestSummary {
  netPnlSol: number; winRatePct: number; profitFactor: number; sharpe: number; maxDrawdownPct: number;
  trades: number; avgHoldingHours: number; rugExposurePct: number;
}
export interface StrategyVersion { version: string; date: string; note: string; }
export interface AuditEntry { time: string; action: string; detail: string; }
export interface StrategyRisk { riskPerTradePct: number; stopLossPct: number; takeProfitLevels: number[]; maxDrawdownStopPct: number; }
export interface StrategySizing { model: string; sizePct: number; }
export interface StrategyDetail {
  id: string; name: string; status: StrategyStatus; timeframe: string; description: string;
  entry: StrategyCondition[]; exit: StrategyCondition[];
  risk: StrategyRisk; sizing: StrategySizing;
  supportedLaunchpads: string[]; minScores: { creator: number; safety: number };
  performance: StrategyPerformance; equityCurve: EquityPoint[]; backtest: BacktestSummary;
  versions: StrategyVersion[]; audit: AuditEntry[];
}
```
- `SentinelApi.getStrategies(): Promise<StrategyRow[]>` (mock ~6 strateji), `getStrategy(id): Promise<StrategyDetail>` (deterministik).
- `httpApi` → `notReady`. `qk.strategies`, `qk.strategy(id)`; `useStrategies()`, `useStrategy(id)`.

### 3.2 Config (OCP)
`lib/strategy/status-defs.ts`:
```ts
export const STATUS_DEFS: Record<StrategyStatus, { label: string; color: string }>; // 7 durum (Taslak/Backtest/Kağıt/Gölge/Canlı/Duraklatıldı/Arşiv)
export const CONDITION_LABELS: Record<string, string>; // creatorScore→"Creator Skoru", tokenSafety→"Token Güvenliği", liquidity→"Likidite", holderGrowth5m→"Holder Büyümesi 5dk", manipulationRisk→"Manipülasyon Riski", momentum→"Momentum", ageSeconds→"Yaş"
export function formatCondition(c: StrategyCondition): string; // "Creator Skoru > 75" (unit varsa ekle)
```

### 3.3 Bileşenler (`components/strategy/`)
```
components/strategy/
├─ StatusBadge.tsx          # STATUS_DEFS'ten durum rozeti
├─ StrategyCard.tsx         # tek strateji kartı (durum + metrikler + son sinyal) → /strategies/[id]
├─ StrategiesListContent.tsx# kart grid + durum filtresi; useStrategies
├─ ConditionList.tsx        # IF/THEN koşul çipleri (formatCondition); {title, conditions}
├─ StrategyPerformancePanel.tsx # performans metrik tile'ları (MetricTile reuse)
├─ EquityCurve.tsx          # Recharts area (equity curve)
├─ BacktestSummaryPanel.tsx # backtest metrik tile'ları
├─ VersionHistory.tsx       # version listesi
├─ AuditLog.tsx             # audit entries
└─ StrategyDetailContent.tsx# kompozisyon: header + status + conditions + risk/sizing + perf + equity + backtest + launchpads/minScores + versions + audit; useStrategy
app/(app)/strategies/page.tsx           # server: getStrategies prefetch
app/(app)/strategies/[id]/page.tsx      # server: getStrategy prefetch
```
- **StrategiesListContent:** `useStrategies()`; durum çipleri (STATUS_DEFS) ile client-side filtre; kart grid.
- **ConditionList:** `entry`/`exit` için IF/THEN başlığı + koşul çipleri (`formatCondition`).
- **StrategyDetailContent:** `useStrategy(id)`; loading (Skeleton)/error state; tüm panelleri kompoze eder.

### 3.4 Sayfalar
- `/strategies` (liste) + `/strategies/[id]` (detay) — server components, RSC prefetch + hydration. `/strategies` placeholder değişir.

---

## 4. Kapsam dışı (bilinçli)
Create/edit builder stepper + no-code condition kurucu, gerçek deploy/execution, live paper/shadow toggle, strateji versiyonu geri alma. Backend gerçek strateji API'si (mock; seam hazır).

## 5. Test stratejisi (TDD)
- `status-defs`/`CONDITION_LABELS`: 7 durum + metrik etiketleri; `formatCondition` doğru string.
- `getStrategies`/`getStrategy` adapter (deterministik; entry/exit/perf/equity/backtest/versions/audit dolu); `useStrategies`/`useStrategy` hook.
- `StatusBadge`: durum label/renk.
- `ConditionList`: koşullar `formatCondition` ile render (IF/THEN başlık).
- `StrategyCard`: durum + metrikler + detay linki `/strategies/<id>`.
- `EquityCurve`: smoke (Recharts container).
- `StrategiesListContent`: kartlar + durum filtresi daraltır.
- `StrategyDetailContent`: smoke (conditions + performans + backtest başlıkları).

## 6. Kabul kriterleri
1. Sidebar "Stratejiler" → `/strategies`: strateji kartları + durum filtresi; karta tıkla → `/strategies/[id]`.
2. Detay: header + durum + entry/exit IF/THEN koşulları + risk/sizing + performans metrikleri + equity curve grafiği + backtest özeti + supported launchpads/min scores + version history + audit log.
3. Durum filtresi kartları süzer; "Temizle" sıfırlar.
4. Tüm veri `component → useStrategies/useStrategy → getApi() → mock`; hiçbir bileşen mock import etmez.
5. Testler yeşil; build başarılı; SOLID/clean + reuse ölçütü.
