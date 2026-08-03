# SENTINEL Frontend — Increment 9 Design Spec
### Backtesting

- **Tarih:** 2026-08-03
- **Durum:** Onaylandı (2026-08-03) — plan yazılacak.
- **Önkoşul:** Increment 1–8 master'da (Trading Terminal dahil).
- **Kaynaklar:** `docs/design/sentinel-ui-ux-design.md` (Ekran 9: Backtesting + Event Replay), `ROADMAP.md` (§Backtest), `docs/progress.md`.

---

## 1. Amaç ve kapsam

Strateji geriye-test (backtest) sonuç ekranı: parametre formu → simüle çalıştır → metrikler + grafikler.
Tasarım Ekran 9'un **backtest sonuç** yarısı. **Event Replay bilinçli olarak sonraki artıma ertelendi**
(ayrı bir etkileşim: playback state'li timeline oynatıcı + look-ahead engelleme).

**Rota:** Mevcut **"Geriye Test" / `/backtesting`** nav placeholder'ı gerçeğe döner (rename yok).

**Yerleşim:** sol **parametre paneli (~300px)** + ana **sonuç alanı** (metrik grid + grafik grid);
dar viewport'ta dikey yığılır. İlk açılışta sonuç alanı **boş-durum** ("Parametreleri seç ve Çalıştır'a bas").

**Run davranışı (tam simüle, deterministik-seeded):** Backend yok. "Çalıştır" → `runBacktest(params)`
seam'den **parametrelerce seed'lenen deterministik** sonuç döndürür (strateji + tarih aralığı + sermaye
sonucu anlamlı değiştirir; form işlevsel hisseder). React Query deseni: form draft params tutar;
"Çalıştır" `submittedParams`'ı set eder; `useBacktest(submittedParams)` yalnız submit sonrası (`enabled`)
çalışır, params'a göre cache'lenir (aynı parametrelerle tekrar bedava).

**Kapsam dışı (bilinçli — sonraki artımlar):** **Event Replay** (timeline oynatıcı + look-ahead
engelleme), missed-opportunity grafiği, gerçek backtest engine (mock; seam hazır), backtest run
kaydetme/karşılaştırma, parametre preset'leri kaydetme, CSV export, entry/exit'te tam mum grafiği
(basit fiyat çizgisi + marker yeterli), parametre formu için RHF+Zod (kontrollü form + saf `validateParams` yeterli).

---

## 2. Clean Code & SOLID + reuse (ölçüt)

- **Reuse:**
  - **`EquityCurve`** (`components/sentinel/`) sermaye eğrisi için (başlık "Sermaye Eğrisi").
  - **`MetricTile`** (`components/sentinel/`) 10 metrik tile için; PnL renklendirmesi `pnlColor` (`lib/position/risk-filter`).
  - `getStrategies`/`useStrategies` (Increment 6) strateji dropdown'unu besler.
  - `EquityPoint` tipi (sermaye eğrisi + fiyat serisi) yeniden kullanılır.
  - Recharts (area/bar/composed) portföy grafik desenleri; Türkçe tooltip `name`/`formatter`.
  - `ScoreBadge` (skor kovaları etiketinde opsiyonel).
  - Native styled `<select>` (Header'daki "Solana Mainnet" deseni) param dropdown'ları için — yeni dep yok.
- **SRP:** her bileşen tek iş (`BacktestParamsForm`, `BacktestMetrics`, her grafik ayrı, `BacktestContent`).
- **OCP:** model/preset/metric registry'leri (`RANGE_PRESETS`/`SLIPPAGE_MODELS`/`LATENCY_MODELS`/
  `LIQUIDITY_MODELS`/`BACKTEST_METRIC_DEFS`); grafikler ve metrik grid config'ten türer.
- **DIP:** bileşenler `useBacktest`/`useStrategies` → `getApi()`; hiçbir bileşen mock import etmez.
- **ISP:** dar prop'lar (her grafik yalnız kendi serisini alır; `DrawdownChart` yalnız `DrawdownPoint[]`).

---

## 3. Mimari

### 3.1 Veri seam genişlemesi (`lib/api`)
```ts
import type { EquityPoint } from "./types"; // mevcut, reuse

export interface BacktestParams {
  strategyId: string;
  rangePreset: string;               // "7g" | "30g" | "90g" | "1y" (RANGE_PRESETS)
  initialCapitalSol: number; maxPositions: number;
  slippageModel: string; priorityFee: number;
  latencyModel: string; liquidityModel: string;
  minCreatorScore: number; minTokenSafety: number;
}
export interface BacktestMetrics {
  netPnlSol: number; winRatePct: number; profitFactor: number; sharpe: number; sortino: number;
  maxDrawdownPct: number; avgTradeSol: number; rugExposurePct: number; trades: number; avgHoldingHours: number;
}
export interface MonthlyReturn { label: string; pct: number; }
export interface DistributionBucket { label: string; count: number; }
export interface ScorePnl { scoreBucket: string; pnlSol: number; }
export interface DrawdownPoint { t: number; v: number; }        // v ≤ 0
export interface BacktestTrade { time: number; price: number; side: "buy" | "sell"; pnlSol: number; }
export interface BacktestResult {
  metrics: BacktestMetrics;
  equityCurve: EquityPoint[];
  drawdown: DrawdownPoint[];
  monthlyReturns: MonthlyReturn[];
  tradeDistribution: DistributionBucket[];
  pnlByScore: ScorePnl[];
  priceSeries: EquityPoint[];        // entry/exit grafiği fiyat çizgisi
  trades: BacktestTrade[];           // entry/exit marker'ları (fiyat serisi zaman aralığında)
}
```
- `SentinelApi.runBacktest(params: BacktestParams): Promise<BacktestResult>` (deterministik;
  `seedOf(strategyId + rangePreset + initialCapitalSol + ...)`; params değişince sonuç değişir; 10 metrik
  dolu, tüm seriler dolu, `trades` en az bir buy + bir sell içerir ve zamanları `priceSeries` aralığında).
- `httpApi` → `notReady`. `qk.backtest(params)` (params'ı deterministik stringify ile anahtarla),
  `useBacktest(params: BacktestParams | null)` (`enabled: !!params`). Strateji dropdown'u `useStrategies`.

### 3.2 Config & saf mantık (OCP, `lib/backtest/`)
```ts
export const RANGE_PRESETS: { key: string; label: string }[];      // Son 7/30/90 gün, 1 yıl
export const SLIPPAGE_MODELS: { key: string; label: string }[];    // sabit / dinamik / kötümser
export const LATENCY_MODELS: { key: string; label: string }[];     // düşük / gerçekçi / yüksek
export const LIQUIDITY_MODELS: { key: string; label: string }[];   // kısıtsız / likidite-oranlı
export const DEFAULT_BACKTEST_PARAMS: BacktestParams;
export const BACKTEST_METRIC_DEFS: { key: keyof BacktestMetrics; label: string; kind: "pnl" | "pct" | "num" }[];
export function validateParams(p: BacktestParams): { [field: string]: string }; // sermaye>0, maxPositions≥1, skor 0–100
```

### 3.3 Bileşenler (SRP, `components/backtest/`)
```
components/backtest/
├─ BacktestParamsForm.tsx     # strateji select (useStrategies) + tarih aralığı + sermaye + max pozisyon
│                             #   + slippage/latency/liquidity model select + min creator/safety + "Çalıştır"
├─ BacktestMetrics.tsx        # 10 metrik tile (MetricTile reuse; BACKTEST_METRIC_DEFS; pnlColor)
├─ DrawdownChart.tsx          # Recharts area (DrawdownPoint[], negatif)
├─ MonthlyReturnChart.tsx     # Recharts bar (MonthlyReturn[])
├─ TradeDistributionChart.tsx # Recharts bar (DistributionBucket[])
├─ PnlByScoreChart.tsx        # Recharts bar (ScorePnl[], pnlColor per-Cell)
├─ EntryExitChart.tsx         # Recharts ComposedChart: fiyat Line + al/sat Scatter marker (yeşil/kırmızı)
└─ BacktestContent.tsx        # kompozisyon: params form + (boş-durum | loading | error | sonuç);
                              #   submittedParams state; sonuç = BacktestMetrics + EquityCurve + 5 grafik

app/(app)/backtesting/page.tsx  # server: getStrategies prefetch (form hazır); backtest on-demand (prefetch yok)
```
- **BacktestParamsForm:** kontrollü form; select'ler registry'lerden türer; `validateParams` hataları alan
  altında; "Çalıştır" geçerliyse `onRun(params)` çağırır. Strateji listesi `useStrategies`'ten.
- **BacktestContent:** `submittedParams` yoksa boş-durum; varsa `useBacktest(submittedParams)` → loading
  Skeleton / error / sonuç. Sonuç: metrik grid + Sermaye Eğrisi (`EquityCurve`) + Drawdown + Aylık Getiri +
  Trade Dağılımı + PnL by Score + Entry/Exit grafikleri.
- **EntryExitChart:** `priceSeries` fiyat `Line`'ı + `trades` marker'ları (`Scatter`; buy yeşil `#2FD98B`,
  sell kırmızı `#F0476B`). Tam mum grafiği değil (YAGNI).

### 3.4 Sayfa
`/backtesting` — server component, `getStrategies` prefetch + HydrationBoundary (form dropdown hazır olsun).
Backtest sonucu on-demand (kullanıcı Çalıştır'a basınca) — prefetch edilmez. Mevcut `PlaceholderScreen`'den gerçeğe döner.

---

## 4. Kapsam dışı (bilinçli)
Event Replay (sonraki artım), missed-opportunity grafiği, gerçek backtest engine (mock), run
kaydetme/karşılaştırma, parametre preset kaydetme, CSV export, tam mum grafiği, RHF+Zod. Backend gerçek
backtest servisi (mock; seam hazır).

## 5. Test stratejisi (TDD)
- `runBacktest` adapter (deterministik: aynı params → eşit sonuç; farklı params → farklı sonuç; 10 metrik
  dolu; equityCurve/drawdown/monthlyReturns/tradeDistribution/pnlByScore/priceSeries dolu; `trades` ≥1 buy +
  ≥1 sell, zamanları priceSeries aralığında); `useBacktest` (`enabled` — params null iken fetch etmez).
- `validateParams`: geçerli default → boş; sermaye 0/negatif, maxPositions 0, skor >100 → ilgili alan hatası.
- `BacktestParamsForm`: strateji + model select'leri render; geçersiz sermaye → hata + Çalıştır engellenir;
  geçerli submit → `onRun` doğru params ile çağrılır (useStrategies wrap edilir).
- `BacktestMetrics`: 10 tile etiketi + PnL renk (net PnL yeşil/kırmızı).
- `DrawdownChart`/`MonthlyReturnChart`/`TradeDistributionChart`/`PnlByScoreChart`: smoke (Recharts container).
- `EntryExitChart`: fiyat serisi + al/sat marker render (Recharts container + Scatter).
- `BacktestContent`: boş-durum ("Çalıştır'a bas") render; run sonrası metrik + grafik başlıkları (smoke).

## 6. Kabul kriterleri
1. Sidebar "Geriye Test" → `/backtesting`: sol parametre formu (strateji + tarih aralığı + sermaye + max
   pozisyon + slippage/latency/liquidity + min creator/safety) + boş sonuç alanı.
2. Parametreleri seç → "Çalıştır" → sonuç alanı: 10 metrik (PnL renkli) + Sermaye Eğrisi + Drawdown +
   Aylık Getiri + Trade Dağılımı + PnL by Score + Entry/Exit grafikleri.
3. Parametreler sonucu **anlamlı değiştirir** (deterministik-seeded); aynı parametrelerle tekrar aynı sonuç.
4. Geçersiz parametre (sermaye ≤ 0 vb) → alan hatası + Çalıştır engellenir.
5. Entry/Exit grafiği fiyat çizgisi + al (yeşil) / sat (kırmızı) marker'ları gösterir.
6. Tüm veri `component → useBacktest/useStrategies → getApi() → mock`; hiçbir bileşen mock import etmez.
7. Testler yeşil; `npm run build` başarılı; SOLID/clean + reuse ölçütü.
