# SENTINEL Frontend — Increment 8 Design Spec
### Trading Terminal

- **Tarih:** 2026-08-03
- **Durum:** Onaylandı (2026-08-03) — plan yazılacak.
- **Önkoşul:** Increment 1–7 master'da (Portfolio/Positions dahil).
- **Kaynaklar:** `docs/design/sentinel-ui-ux-design.md` (Ekran 7: Trading Terminal + Güvenlik UX'i), `ROADMAP.md` (§7 Trading engine), `docs/progress.md`.

---

## 1. Amaç ve kapsam

Tasarım dokümanı **Ekran 7 (Trading Terminal)**'in tamamı: 4 bölmeli profesyonel işlem terminali —
sol token/watchlist listesi, orta market data + candlestick fiyat grafiği, sağ order paneli, alt
sekmeli panel (Pozisyonlar / Emirler / İşlemler / Loglar).

**Rota:** Mevcut placeholder **"Emirler" / `/orders`** nav'ı **"Terminal" / `/terminal`** olarak
yeniden adlandırılır. Gerekçe: 4 bölmeli tam terminal'e "Emirler" demek onu küçültür; **Emirler zaten
alt sekmelerden biri.** `components/shell/nav.ts` tek satır güncellenir; eski `/orders` placeholder
sayfası `/terminal`'e taşınır (gerçeğe döner).

**Emir davranışı (tam simüle, durumsuz):** Backend yok. Order paneli → onay modalı → gönderim
**gerçek trade yapmaz ve hiçbir emir kalıcı olmaz.** Token Detail / Positions deseni: canlı modda
`toast.warning` + güçlü güvenlik uyarısı, kağıt/gölge modda simüle `toast` + simulation özeti. Alt
"Emirler" sekmesi seam'den gelen **seed** emirleri gösterir (yeni gönderilen emir orada belirmez —
durumsuz).

**Fiyat grafiği:** `lightweight-charts` (TradingView açık kütüphanesi) candlestick — yeni bağımlılık;
Cytoscape'teki gibi `next/dynamic ssr:false` ile yüklenir.

**Order formu:** Yeni bağımlılık (RHF+Zod) **eklenmez** — kontrollü local state + saf `validateOrder()`
modülü. Simüle/durumsuz form için yeterli, SOLID/küçük-modül etiğiyle uyumlu, saf birim test edilebilir.

**Kapsam dışı (bilinçli — sonraki artımlar):** gerçek emir gönderimi/blockchain (Jupiter route,
tx sign/submit/confirmation/retry), durumlu emir yaşam döngüsü (gönderilen emrin listeye düşmesi /
pozisyon-bakiye güncellemesi), order book / market depth, DCA & trailing/creator-sale/risk-score
tetikli otomatik exit'ler, çoklu-wallet, tablo saved views / CSV export / kolon customization /
virtual scroll, RHF+Zod (gerçek submit gelince değerlendirilir), mobilde tam terminal (mobilde
portfolio/positions/emergency-pause öncelikli — tasarım Responsive bölümü).

---

## 2. Clean Code & SOLID + reuse (ölçüt)

- **Reuse:**
  - Alt "Pozisyonlar" sekmesi Increment 7 **`PositionsTable`**'ı aynen kullanır (drawer'sız veya
    mevcut satır-tıklama davranışıyla — plan netleştirir).
  - Sol token listesi + market data mevcut **`getTokens`/`getToken`** verisini reuse eder.
  - `riskMeta`, `ScoreBadge`, `TokenAvatar`, `WalletAddress`, `pnlColor` (`lib/position/risk-filter`),
    `MetricTile`, shadcn `Tabs` (`components/ui/tabs.tsx`), `toast` (sonner), `useSessionStore` (`tradingMode`).
  - shadcn **`Dialog`** (Base UI tabanlı) yeni eklenir (`components/ui/dialog.tsx`) — confirmation modalı
    için doğru primitive (mevcut `Sheet`/drawer deseninin modal karşılığı).
- **SRP:** her bölme/bileşen tek iş (`TokenWatchlistPanel`, `MarketDataHeader`, `PriceChart`,
  `OrderPanel`, `OrderConfirmDialog`, `OrdersTable`, `TransactionsTable`, `TradeLogsList`, `BottomTabsPanel`).
- **OCP:** emir alanları registry-tabanlı (`ORDER_SIDE_DEFS`, `ORDER_TYPE_DEFS`); order defter/işlem/log
  sekmeleri registry-tabanlı (`TERMINAL_TAB_DEFS`); saf `validateOrder`/`simulateOrder` config'ten türer.
- **DIP:** bileşenler `useCandles`/`useMarketData`/`useOrders`/`useTransactions`/`useTradeLogs`/`useTokens`
  → `getApi()`; hiçbir bileşen mock import etmez.
- **ISP:** dar prop'lar (`PriceChart` yalnız `Candle[]`; `OrderConfirmDialog` yalnız `{draft, market, mode, onConfirm, onClose}`).

---

## 3. Mimari

### 3.1 Veri seam genişlemesi (`lib/api`)
```ts
import type { RiskLevel } from "@/lib/format";

export interface Candle { time: number; open: number; high: number; low: number; close: number; }
export interface MarketData {
  mint: string; symbol: string;
  price: number; change24hPct: number;
  liquiditySol: number; volume24hSol: number; marketCapSol: number;
  tokenScore: number; creatorScore: number;
}
export type OrderSide = "buy" | "sell";
export type OrderType = "market" | "limit";
export type OrderStatus = "open" | "filled" | "cancelled";
export interface Order {
  id: string; tokenSymbol: string; tokenMint: string;
  side: OrderSide; type: OrderType; status: OrderStatus;
  price: number; amountSol: number; createdAt: string;
}
export interface Txn {
  id: string; hash: string; kind: "buy" | "sell" | "approve";
  tokenSymbol: string; amountSol: number; status: "success" | "pending" | "failed"; time: string;
}
export interface TradeLog { id: string; level: "info" | "warn" | "error"; message: string; time: string; }
```
- `SentinelApi`: `getCandles(mint)`, `getMarketData(mint)`, `getOrders()`, `getTransactions()`,
  `getTradeLogs()` (hepsi deterministik mock; `Order`/`Txn` token id'leri mevcut mock token'lara
  referans verir ki semboller/linkler tutarlı olsun). `httpApi` → `notReady`.
- `qk.candles(mint)`, `qk.marketData(mint)`, `qk.orders`, `qk.transactions`, `qk.tradeLogs`;
  `useCandles(mint)`, `useMarketData(mint)`, `useOrders()`, `useTransactions()`, `useTradeLogs()`.

### 3.2 Config & saf mantık (OCP, `lib/terminal/`)
```ts
export const ORDER_SIDE_DEFS: { key: OrderSide; label: string; color: string }[]; // Al / Sat
export const ORDER_TYPE_DEFS: { key: OrderType; label: string }[];                // Market / Limit
export const TERMINAL_TAB_DEFS: { key: string; label: string }[];                 // Pozisyonlar/Emirler/İşlemler/Loglar

export interface OrderDraft {
  side: OrderSide; type: OrderType; amountSol: number; sizePct: number;
  limitPrice?: number; slippagePct: number; priorityFee: number;
  stopLossPct?: number; takeProfitPct?: number; trailingPct?: number;
}
export interface OrderErrors { [field: string]: string } // saf; boşsa geçerli
export function validateOrder(d: OrderDraft, market: MarketData): OrderErrors;
export interface OrderSimulation {
  estPrice: number; priceImpactPct: number; minReceived: number;
  estFeeSol: number; route: string;
}
export function simulateOrder(d: OrderDraft, market: MarketData): OrderSimulation;
```

### 3.3 Bileşenler
```
components/ui/
└─ dialog.tsx                   # shadcn Base UI Dialog (yeni; confirmation modalı)

components/terminal/
├─ TerminalContent.tsx          # kompozisyon + activeMint state; loading/error
├─ TokenWatchlistPanel.tsx      # sol; useTokens listesi, seçim → activeMint
├─ MarketDataHeader.tsx         # orta üst; useMarketData; fiyat/24s/likidite/hacim/MC + skor rozetleri
├─ PriceChart.tsx               # orta; next/dynamic ssr:false candlestick wrapper (useCandles)
├─ PriceChartCanvas.tsx         # lightweight-charts instance (client-only)
├─ OrderPanel.tsx               # sağ; kontrollü form + türetilen simulation status; validateOrder
├─ OrderConfirmDialog.tsx       # onay modalı; canlı→uyarı, kağıt→simüle; {draft, market, mode, onConfirm, onClose}
├─ OrdersTable.tsx              # alt; useOrders; side/type/status/price/amount + iptal (simüle toast)
├─ TransactionsTable.tsx        # alt; useTransactions; hash+explorer link/status/time
├─ TradeLogsList.tsx            # alt; useTradeLogs; level renkli mesaj akışı
└─ BottomTabsPanel.tsx          # tabs.tsx; Pozisyonlar(PositionsTable reuse)/Emirler/İşlemler/Loglar

app/(app)/terminal/page.tsx     # server: getMarketData(default)+getCandles(default)+getOrders+
                                #         getTransactions+getTradeLogs+getPositions prefetch → TerminalContent
```
- **activeMint:** varsayılan ilk mock token; `TokenWatchlistPanel` seçimi yukarı bildirir; orta+sağ
  bölmeler activeMint'e göre `useMarketData(activeMint)`/`useCandles(activeMint)` okur.
- **PriceChart:** `next/dynamic ssr:false` (Cytoscape `WalletGraphCanvas` deseni) — lightweight-charts
  yalnız client'ta yüklenir; container resize handling.
- **OrderPanel:** kontrollü form; her değişimde `simulateOrder` türetilir (price impact / min received /
  est fee / route gösterilir); `validateOrder` hataları alan altında; "Önizle" geçerliyse dialog açar.

### 3.4 State
Terminal-yerel: `activeMint` + `orderDraft` `TerminalContent` içinde `useState`/`useReducer` (Zustand'a
global ekleme yok — bölmeler arası paylaşım prop ile). Global `useSessionStore` yalnız `tradingMode`
(canlı-uyarı dallanması) için okunur.

### 3.5 Emir akışı & güvenlik (Güvenlik UX'i)
`OrderPanel` "Önizle" → `validateOrder` geçerse `OrderConfirmDialog`. Modal içeriği: token, side, tutar,
tahmini fiyat, slippage, price impact, **risk skoru + creator skoru** (`MarketData.tokenScore`/`creatorScore`,
`riskMeta`/`scoreToLevel` ile seviyelenir), wallet bakiyesi (mock sabit/session), tahmini fee. "Açık
riskler" ayrı bir liste değil — düşük token/creator skorundan **türetilen kısa uyarı satırı** (yeni seam
verisi yok). **Canlı modda güçlü güvenlik uyarısı bloğu** (kırmızı vurgulu). "Onayla":
- **Canlı mod** → `toast.warning` ("Canlı işlem devre dışı — güvenlik") + gerçek trade **yok**.
- **Kağıt/gölge** → simüle `toast` ("Emir simüle edildi") + simulation özeti.
Her iki durumda emir kalıcı olmaz. Emergency Pause (mevcut header) ileride bağlanabilir (kapsam dışı not).

### 3.6 Sayfa
`/terminal` — server component, RSC prefetch + HydrationBoundary. Varsayılan token için `getMarketData`
+ `getCandles`, ayrıca `getOrders`/`getTransactions`/`getTradeLogs`/`getPositions` prefetch → `TerminalContent`.
`nav.ts` "Emirler"/`/orders` → "Terminal"/`/terminal`.

---

## 4. Kapsam dışı (bilinçli)
Yukarıda §1'de listelenen maddeler: gerçek emir/blockchain gönderimi, durumlu emir yaşam döngüsü,
order book/depth, otomatik exit otomasyonu, çoklu-wallet, tablo saved views/CSV/kolon custom/virtual
scroll, RHF+Zod, mobil tam terminal. Backend gerçek trading engine (mock; seam hazır).

## 5. Test stratejisi (TDD)
- Seam adapter'ları: `getCandles`/`getMarketData`/`getOrders`/`getTransactions`/`getTradeLogs`
  (deterministik; dolu seriler; id'ler mevcut token'lara referans). Hook'lar smoke.
- `validateOrder`: geçerli draft → boş hata; negatif/aşırı miktar, geçersiz slippage, limit'te eksik
  fiyat → ilgili alan hatası. `simulateOrder`: price impact/min received/fee türetimi deterministik.
- `TokenWatchlistPanel`: liste render; seçim callback aktif token'ı değiştirir.
- `MarketDataHeader`: fiyat/istatistik/skor rozetleri render.
- `PriceChart`: smoke (dynamic wrapper mount; canvas mock'lanır — Cytoscape test deseni).
- `OrderPanel`: alanlar render; geçersiz girişte hata gösterir + "Önizle" disabled/engellenir;
  simulation değerleri güncellenir.
- `OrderConfirmDialog`: canlı modda `toast.warning` + gerçek trade yok; kağıt modda simüle `toast`
  (sonner mock'lanır); modal içeriği (tutar/impact/skor) render.
- `OrdersTable`/`TransactionsTable`/`TradeLogsList`: satırlar render; iptal butonu simüle toast;
  tx explorer linki; log seviyesi renkli.
- `BottomTabsPanel`: sekme geçişi her tabloyu gösterir; Pozisyonlar `PositionsTable` reuse render.
- `TerminalContent`: smoke (4 bölme + market başlığı + order paneli + alt sekmeler).

## 6. Kabul kriterleri
1. Sidebar "Terminal" → `/terminal`: sol token listesi + orta market data & candlestick grafik + sağ
   order paneli + alt sekmeler (Pozisyonlar/Emirler/İşlemler/Loglar) render.
2. Sol listeden token seçmek orta (market data + grafik) ve sağ (order paneli) bölmeleri günceller.
3. Order paneli: Buy/Sell, Market/Limit, miktar, size %, slippage, priority fee, SL/TP, trailing;
   geçersiz girişte hata; geçerli girişte simulation status (price impact/min received/route/fee).
4. "Önizle" → onay modalı (token/side/tutar/tahmini fiyat/slippage/impact/risk & creator skor/bakiye/fee).
   "Onayla": canlı → `toast.warning` (güçlü uyarı, gerçek trade yok); kağıt/gölge → simüle `toast`.
5. Alt sekmeler: Pozisyonlar Inc7 tablosunu reuse; Emirler/İşlemler/Loglar seam'den render; emir iptal
   + tx explorer linki çalışır (iptal simüle toast).
6. Fiyat grafiği `lightweight-charts` candlestick, `ssr:false` client-only; SSR/prerender kırılmaz.
7. Tüm veri `component → hook → getApi() → mock`; hiçbir bileşen mock import etmez.
8. Testler yeşil; `npm run build` başarılı; SOLID/clean + reuse ölçütü.
