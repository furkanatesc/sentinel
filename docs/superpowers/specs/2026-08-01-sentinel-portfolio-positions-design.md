# SENTINEL Frontend — Increment 7 Design Spec
### Portfolio & Positions

- **Tarih:** 2026-08-01
- **Durum:** Onaylandı (2026-08-01) — plan yazıldı: `docs/superpowers/plans/2026-08-01-sentinel-frontend-increment-7-portfolio-positions.md`
- **Önkoşul:** Increment 1–6 master'da.
- **Kaynaklar:** `docs/design/sentinel-ui-ux-design.md` (Ekran 8: Portfolio ve Positions), `docs/progress.md`, `ROADMAP.md`.

---

## 1. Amaç ve kapsam

Portföy takibi + açık pozisyonlar. Sidebar'da iki ayrı nav (`Portföy`, `Pozisyonlar`) olduğu için **iki rota**:

- **`/portfolio`** — portföy genel bakışı: KPI'lar (toplam değer, bakiye, yatırılan, realized/unrealized/günlük PnL, max drawdown, risk/rug maruziyeti) + 4 grafik (equity curve, PnL by strateji, risk allocation, win/loss) + küçük "açık pozisyonlar özeti" (→ `/positions`).
- **`/positions`** — açık pozisyonlar: zengin tablo (token/strateji linkleri, PnL renkleri, risk rozetleri, read-only aksiyon butonları) + satır tıklayınca **detay drawer** (entry/current, SL/TP, risk kırılımı, read-only aksiyonlar).

**Read-only:** Pozisyon aksiyonları (Kapat / SL-TP ayarla) gerçek trade yapmaz. Token Detail deseni: canlı modda `toast.warning`, paper/shadow modda `toast` (simüle bildirimi).

**Kapsam dışı (bilinçli — sonraki artımlar):** gerçek pozisyon kapatma/emir gönderimi (Trading Terminal / Orders artımı), PnL by creator-score & token-age grafikleri, tablo saved views / CSV export / kolon özelleştirme / virtual scroll, pozisyon düzenleme formu.

---

## 2. Clean Code & SOLID + reuse (ölçüt)

- **Reuse:**
  - **`EquityCurve` paylaşımlıya taşınır:** `components/strategy/EquityCurve.tsx` → **`components/sentinel/EquityCurve.tsx`**; opsiyonel `title?`/`color?` prop'ları eklenir (default `"Equity Curve"` / `#2FD98B`) ve gradient `id` renkten türetilir (mevcut sabit `equity-grad` çakışma followup'ını kapatır). Strategy detay (`StrategyDetailContent`) import'u güncellenir. Portfolio aynı bileşeni kullanır.
  - `MetricTile` (`components/sentinel/`) KPI/metrik tile'ları için; PnL renklendirmesi için opsiyonel `valueColor?: string` prop eklenir (dar, geriye-uyumlu ISP genişletmesi — `ScoreCard.hideExplain?` precedent'i gibi).
  - `riskMeta` (risk rozeti renk/etiket), `TokenAvatar`/`WalletAddress`, shadcn `Sheet` (drawer, Live Feed `EventDetailDrawer` deseni), `toast` (sonner) reuse.
  - `EquityPoint` tipi Strategies'ten yeniden kullanılır.
- **SRP:** her bileşen tek iş (`PortfolioKpis`, `PnlByStrategyChart`, `RiskAllocationChart`, `WinLossChart`, `OpenPositionsSummary`, `PositionsTable`, `PositionDetailDrawer`, `PositionActions` ayrı).
- **OCP:** risk allocation renkleri `riskMeta` paletinden türer; pozisyon filtresi risk seviyesi çipleri (registry-tabanlı).
- **DIP:** bileşenler `usePortfolio`/`usePositions`/`getApi()`; hiçbir bileşen mock import etmez.
- **ISP:** dar prop'lar (`PnlByStrategyChart` yalnız `StrategyPnl[]`, `PositionDetailDrawer` yalnız `{position, onClose}`).

---

## 3. Mimari

### 3.1 Veri seam genişlemesi (`lib/api`)
```ts
import type { RiskLevel } from "@/lib/format";
// EquityPoint mevcut (Strategies) — yeniden kullanılır.

export interface PortfolioSummary {
  totalValueSol: number; availableSol: number; investedSol: number;
  realizedPnlSol: number; unrealizedPnlSol: number; dailyPnlSol: number;
  maxDrawdownPct: number; riskExposurePct: number; rugExposurePct: number;
}
export interface StrategyPnl { strategyId: string; name: string; pnlSol: number; }
export interface AllocationSlice { label: string; pct: number; color: string; }
export interface WinLossBucket { label: string; count: number; }
export interface PortfolioOverview {
  summary: PortfolioSummary;
  equityCurve: EquityPoint[];
  pnlByStrategy: StrategyPnl[];
  riskAllocation: AllocationSlice[];
  winLoss: WinLossBucket[];
}
export interface Position {
  id: string; tokenMint: string; tokenSymbol: string;
  strategyId: string; strategyName: string;
  entryPrice: number; currentPrice: number; sizeSol: number;
  pnlSol: number; pnlPct: number;
  stopLossPct: number; takeProfitPct: number;
  tokenRisk: RiskLevel; creatorRisk: RiskLevel;
  ageLabel: string; openedAt: string;
}
```
- `SentinelApi.getPortfolio(): Promise<PortfolioOverview>` (tek prefetch; deterministik), `getPositions(): Promise<Position[]>` (deterministik, ~8 açık pozisyon; `tokenMint`/`strategyId` mevcut mock token/strateji id'lerine referans verir ki linkler gerçek sayfalara gitsin).
- `httpApi` → `notReady`. `qk.portfolio`, `qk.positions`; `usePortfolio()`, `usePositions()`.

### 3.2 Config (OCP)
`lib/position/risk-filter.ts`:
```ts
// riskMeta'dan türeyen filtre seçenekleri; POSITIONS_RISK_LEVELS: RiskLevel[]
export const POSITION_RISK_LEVELS: RiskLevel[]; // ["strong","good","medium","high","critical"] gibi
// PnL renk yardımcısı (StrategyCard.pnlColor deseni) tek yerde:
export function pnlColor(v: number): string; // >=0 #2FD98B, <0 #F0476B
```
Risk allocation dilim renkleri mock'ta `riskMeta[level].color` ile üretilir (ayrı palet yok).

### 3.3 Bileşenler
```
components/sentinel/
└─ EquityCurve.tsx            # strategy/'den taşındı; title?/color? props; renkten türetilen gradient id

components/portfolio/
├─ PortfolioKpis.tsx          # KPI grid (MetricTile reuse, PnL valueColor)
├─ PnlByStrategyChart.tsx     # Recharts BarChart (StrategyPnl[])
├─ RiskAllocationChart.tsx    # Recharts PieChart/donut (AllocationSlice[])
├─ WinLossChart.tsx           # Recharts BarChart (WinLossBucket[])
├─ OpenPositionsSummary.tsx   # ilk N açık pozisyon özeti + "Tümü →" (/positions); usePositions
└─ PortfolioContent.tsx       # kompozisyon: KPIs + 4 grafik + özet; usePortfolio (loading/error)

components/position/
├─ PositionsTable.tsx         # sıralanabilir tablo; token→/tokens, strateji→/strategies; PnL renk; risk rozetleri; PositionActions; satır→drawer
├─ PositionActions.tsx        # read-only Kapat / SL-TP butonları (toast; live→toast.warning); useSession mode
├─ PositionDetailDrawer.tsx   # shadcn Sheet; entry/current, size, SL/TP, PnL, risk kırılımı, read-only aksiyonlar; {position, onClose}
└─ PositionsContent.tsx       # usePositions; risk filtre çipleri + sıralama + drawer state (loading/error)

app/(app)/portfolio/page.tsx  # server: getPortfolio + getPositions prefetch → PortfolioContent
app/(app)/positions/page.tsx  # server: getPositions prefetch → PositionsContent
```
- **PortfolioKpis:** `PortfolioSummary`'den tile grid; PnL alanları `pnlColor` ile renklenir (MetricTile `valueColor`).
- **RiskAllocationChart:** donut; her dilim `AllocationSlice.color`; yüzde etiketleri.
- **PositionsTable:** kolonlar token/strateji/entry/current/size/PnL(SOL+%)/SL-TP/token risk/creator risk/yaş/aksiyon; client-side sıralama (PnL, size, yaş); satır tıklaması drawer açar (aksiyon butonlarına tıklama satır tıklamasını tetiklemez — `stopPropagation`).
- **PositionsContent:** risk seviyesi filtre çipleri (`POSITION_RISK_LEVELS`, tokenRisk'e göre süz) + "Temizle"; sıralama state; seçili pozisyon için drawer.

### 3.4 Sayfalar
`/portfolio` + `/positions` — server components, RSC prefetch + HydrationBoundary. İkisi de mevcut `PlaceholderScreen`'den gerçeğe döner. `/portfolio` sayfası hem `getPortfolio` hem `getPositions` prefetch eder (çünkü `OpenPositionsSummary` `usePositions` okur).

---

## 4. Kapsam dışı (bilinçli)
Gerçek pozisyon kapatma/emir gönderimi (Trading Terminal / Orders artımı), PnL by creator-score & token-age grafikleri, tablo saved views / CSV export / kolon customization / virtual scroll, pozisyon düzenleme formu. Backend gerçek portföy/pozisyon API'si (mock; seam hazır).

## 5. Test stratejisi (TDD)
- `getPortfolio`/`getPositions` adapter (deterministik; summary + 4 seri dolu; ≥6 pozisyon, id'ler mevcut token/strateji id'lerine referans); `usePortfolio`/`usePositions` hook.
- `risk-filter`: `pnlColor` (>=0 yeşil, <0 kırmızı), `POSITION_RISK_LEVELS` dolu.
- **EquityCurve taşıma:** yeni konumda smoke (Recharts container); `title`/`color` prop'ları render'ı değiştirir; strategy detay hâlâ equity curve gösterir (regresyon).
- `PortfolioKpis`: KPI etiketleri + PnL renk.
- `PnlByStrategyChart`/`RiskAllocationChart`/`WinLossChart`: smoke (Recharts container).
- `PositionsTable`: satırlar render; token linki `/tokens/<mint>`, strateji linki `/strategies/<id>`; sıralama PnL'e göre sıralar; aksiyon butonu satır drawer'ını açmaz.
- `PositionActions`: live modda `toast.warning`, paper modda `toast` (sonner mock'lanır).
- `PositionDetailDrawer`: açık pozisyonla entry/current/SL/TP başlıkları render.
- `PositionsContent`: risk filtre kartları/satırları daraltır; "Temizle" sıfırlar; satır tıklaması drawer açar.
- `PortfolioContent`: smoke (KPIs + grafik başlıkları + açık pozisyon özeti).

## 6. Kabul kriterleri
1. Sidebar "Portföy" → `/portfolio`: KPI'lar + equity curve + PnL by strateji + risk allocation + win/loss + açık pozisyon özeti (→ /positions).
2. Sidebar "Pozisyonlar" → `/positions`: zengin tablo; token/strateji linkleri gerçek sayfalara gider; PnL renkli; risk rozetleri; sıralama çalışır; risk filtresi süzer + "Temizle" sıfırlar.
3. Satıra tıkla → detay drawer (entry/current, SL/TP, risk kırılımı, read-only aksiyonlar). Aksiyon butonu: live→`toast.warning`, paper/shadow→`toast`; gerçek trade yok.
4. `EquityCurve` `components/sentinel/`'e taşındı; Strategies detay hâlâ equity curve gösteriyor (regresyon yok); Portfolio aynı bileşeni kullanıyor.
5. Tüm veri `component → usePortfolio/usePositions → getApi() → mock`; hiçbir bileşen mock import etmez.
6. Testler yeşil; build başarılı; SOLID/clean + reuse ölçütü.
