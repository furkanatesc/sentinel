# SENTINEL Frontend — Increment 1 Design Spec
### İskele + Design System + App Shell + Overview Dashboard

- **Tarih:** 2026-07-30
- **Durum:** Onay bekliyor (brainstorming çıktısı)
- **Kapsam:** Frontend'in ilk artımı. Diğer 11 ekran sonraki artımların konusu.
- **Kaynaklar:** `ROADMAP.md`, `docs/design/sentinel-ui-ux-design.md`, Figma Make referans implementasyonu (`zAWGuUwmbKkSK0n7YKlANt`), `graphify-out/GRAPH_REPORT.md`.

---

## 1. Amaç ve bağlam

SENTINEL, Solana'da yeni çıkan tokenları saniyeler içinde tespit eden, creator/wallet güven skorlaması yapan, risk analizi üreten ve (ileride) otomatik trade eden gerçek zamanlı bir istihbarat + trading platformu. Backend **AWS**'de Go (event ingestion, düşük gecikmeli worker'lar, trading dispatch) + Python (scoring, clustering, ML, backtest, RAG) olarak konumlanıyor (ROADMAP §12).

Bu spec **yalnızca frontend'in ilk artımını** tanımlar: çalışan bir Next.js uygulaması, Sentinel tasarım sistemi, app shell (sidebar + header + trading-mode) ve tam işlevsel **Overview Dashboard**. Backend henüz yokken veriler **mock adapter**'dan gelir; adapter ileride gerçek AWS REST + WebSocket'e bağlanacak şekilde tasarlanır.

### Neden bu kapsam
- İlk artım sonunda **görülebilir, tıklanabilir** bir ürün olur (shell + gerçek dashboard).
- Design system + shell + veri-seam, sonraki 11 ekranın **hepsinin** üzerine kurulacağı temeldir.
- Figma Make'te Overview + primitive'ler zaten çalışıyor; bunları düzgün bir Next.js projesine **port** ediyoruz, sıfırdan tasarlamıyoruz.

---

## 2. Teknoloji kararları (onaylı)

| Katman | Seçim | Not |
|---|---|---|
| Framework | **Next.js (App Router), server-first** | Pure CSR SPA değil; RSC + client island'lar |
| Dil | **TypeScript** (strict) | |
| Styling | **Tailwind CSS v4** | Referans `@theme inline` / `@custom-variant` = v4 |
| Komponent | **shadcn/ui** (Base UI) | shadcn'in güncel varsayılanı `@base-ui/react` (Radix'in aynı ekipten halefi); Referanstaki `ui/*` seti port edilir |
| Server state | **TanStack Query** | RSC prefetch + `HydrationBoundary` |
| Client/UI state | **Zustand** | sidebar collapse, trading mode |
| Real-time | Yerel **WebSocket** client (adapter arkasında) | Increment 1'de mock stream |
| Metrik grafikleri | **Recharts** | Overview'daki radar + sparkline |
| Fiyat/mum grafikleri | **lightweight-charts** | *Bu artımda değil* → Token Detail/Terminal |
| Wallet graph | **Cytoscape.js** | *Bu artımda değil* → Wallet Graph ekranı |
| Form + validation | **React Hook Form + Zod** | *Bu artımda minimal* |
| Test | **Vitest + React Testing Library** | TDD; saf mantık + kritik komponentler |
| Fontlar | **Inter** + **JetBrains Mono** (`next/font`) | |

### Server-first sınırı (kritik)
- **Server Components:** kök layout, shell iskeleti (statik kısımlar), Overview sayfasının **ilk veri prefetch'i** (RSC'de `queryClient.prefetchQuery` → `dehydrate` → `HydrationBoundary`).
- **Client Components (`"use client"`):** yalnızca etkileşimli/canlı adacıklar — `LiveTokenFeed` (sort, watchlist, canlı highlight), `OpportunityRadar` (Recharts), `TradingModeBadge`/sidebar collapse (Zustand), Toaster.
- Yani "her şey tarayıcıda" değil; sunucu ilk render + veriyi verir, canlı güncelleme client island'larda olur. Bu, "client-side gitmeyeceğiz + backend AWS'de" kararıyla bire bir örtüşür.

---

## 3. Mimari

### 3.1 Klasör yapısı
Frontend **monorepo** içinde `apps/web/` altında yaşar (backend ileride `apps/api-go/`, `apps/scoring-py/` olarak yanına gelir; paylaşılan tipler/docs kök seviyede).
```
SENTINEL/                        # monorepo kökü (ROADMAP.md, docs/, graphify-out/ burada)
└─ apps/web/                     # ← bu artımın tüm işi burada
├─ app/
│  ├─ layout.tsx                 # <html class="dark">, fontlar, Providers
│  ├─ providers.tsx              # QueryClientProvider (client boundary)
│  ├─ globals.css                # Tailwind v4 + Sentinel token'ları (theme.css port)
│  └─ (app)/
│     ├─ layout.tsx              # Sidebar + Header shell (server) + Toaster
│     ├─ page.tsx                # Overview (RSC prefetch → HydrationBoundary)
│     ├─ live-feed/page.tsx      # Placeholder
│     ├─ discover/page.tsx       # Placeholder
│     ├─ tokens/page.tsx         # Placeholder
│     ├─ creators/page.tsx       # Placeholder
│     ├─ wallet-graph/page.tsx   # Placeholder
│     ├─ smart-wallets/page.tsx  # Placeholder
│     ├─ strategies/page.tsx     # Placeholder
│     ├─ positions/page.tsx      # Placeholder
│     ├─ orders/page.tsx         # Placeholder
│     ├─ portfolio/page.tsx      # Placeholder
│     ├─ backtesting/page.tsx    # Placeholder
│     ├─ alerts/page.tsx         # Placeholder
│     ├─ telegram/page.tsx       # Placeholder
│     ├─ research/page.tsx       # Placeholder
│     ├─ system-health/page.tsx  # Placeholder
│     └─ settings/page.tsx       # Placeholder
├─ components/
│  ├─ ui/*                       # shadcn primitives (port)
│  ├─ shell/                     # Sidebar, Header, TradingModeBadge, nav.ts
│  ├─ sentinel/                  # ScoreBadge, Sparkline, TokenAvatar, WalletAddress, KpiCard
│  └─ dashboard/                 # LiveTokenFeed, OpportunityRadar, AlertsTimeline
├─ lib/
│  ├─ api/
│  │  ├─ types.ts                # TokenRow, Kpi, AlertEvent, RadarPoint, RiskLevel...
│  │  ├─ contract.ts             # interface SentinelApi { getKpis(), getTokens(), ... , subscribe() }
│  │  ├─ mock.ts                 # mockApi: SentinelApi (referans mock.ts port)
│  │  ├─ http.ts                 # httpApi stub (REST+WS) — TODO backend gelince
│  │  └─ index.ts                # getApi() — env'e göre mock|http seçer
│  ├─ hooks/                     # useKpis, useTokens, useAlerts, useRadar (TanStack Query)
│  ├─ store/                     # Zustand: uiStore (collapsed), sessionStore (tradingMode)
│  └─ format.ts                  # formatAge, formatPrice, formatUsd, scoreToLevel, riskMeta
├─ test/                         # Vitest kurulumu + testler
├─ tailwind / postcss / tsconfig / next.config / .env.example
```

### 3.2 Veri erişim seam'i (en kritik tasarım kararı)
Bileşenler **asla** `lib/api/mock`'u doğrudan import etmez. Akış:

```
Component → lib/hooks/useTokens() → getApi().getTokens() → (mock | http)
```

- `lib/api/contract.ts`: tek arayüz.
  ```ts
  export interface SentinelApi {
    getKpis(): Promise<Kpi[]>;
    getTokens(): Promise<TokenRow[]>;
    getAlerts(): Promise<AlertEvent[]>;
    getRadar(): Promise<RadarPoint[]>;
    // Real-time seam — mock'ta interval, http'de WebSocket
    subscribeTokens(cb: (t: TokenRow[]) => void): () => void;
    subscribeAlerts(cb: (a: AlertEvent) => void): () => void;
  }
  ```
- `getApi()` seçimi `NEXT_PUBLIC_DATA_SOURCE` env'ine göre (`mock` varsayılan, `http` backend gelince).
- **Graph içgörüsü uygulaması:** GRAPH_REPORT'taki ekran↔servis köprüleri (`Live Feed → Token Discovery`, `Trading Terminal → Trading Engine + Pre-trade Safety`, `Portfolio → Trading Engine`, `Token Detail → Token Safety Score`) bu seam'de birer endpoint ailesine karşılık gelecek. Increment 1'de sadece Overview'un ihtiyacı olan `kpis/tokens/alerts/radar` uçları var; sonraki ekranlar bu arayüzü genişletir.
- Backend geldiğinde **yalnızca** `http.ts` yazılır ve env değişir; bileşen/hook'lar dokunulmaz.

### 3.3 Real-time modeli
- `subscribe*` mock'ta `setInterval` ile küçük, kontrollü güncellemeler yayar (yeni token satırı, skor değişimi) — canlı highlight micro-interaction'ını göstermek için.
- http impl'de aynı imza `WebSocket` + reconnect/backoff ile karşılanır.
- TanStack Query cache'i `subscribe` callback'inde `queryClient.setQueryData` ile güncellenir (WS → cache patch deseni).

### 3.4 State
- **Server state** (token/kpi/alert/radar): TanStack Query.
- **UI state:** Zustand — `uiStore { sidebarCollapsed, toggleSidebar }`, `sessionStore { tradingMode: 'paper'|'shadow'|'live', setTradingMode }`. `tradingMode` uygulama genelinde görünür (Live'da güvenlik etiketi).

---

## 4. Design System

Referans `theme.css`'teki Sentinel dark token'ları `globals.css`'e port edilir. **Uygulama dark-only** (design dark-first; light token seti bu artımda kapsam dışı — `.dark` gövde sınıfı köke sabitlenir).

Token özeti:
- Yüzeyler: bg `#080B12`, surface-1 `#111722`, surface-2 `#151C28`, surface-3 `#1A2331`
- Metin: foreground `#E6EAF2`, muted `#8A94A6`
- Accent: primary mor `#7C5CFF`, accent-blue `#3E9BFF`
- Durum: positive `#2FD98B`, warning `#FFB020`, critical `#F0476B`, info `#3E9BFF`, neutral `#8A94A6`
- Border `rgba(255,255,255,0.07)`, radius `0.625rem`
- Fontlar: sans **Inter**, mono **JetBrains Mono** (yalnız adres/mint/hash/teknik metrik)
- Chart paleti: `#7C5CFF, #2FD98B, #3E9BFF, #FFB020, #F0476B`

Skor seviyeleri (tek kaynak `lib/format.ts`):
`0–24 Critical · 25–49 High · 50–69 Medium · 70–84 Good · 85–100 Strong`. Renk + **her zaman metin etiketi** (erişilebilirlik).

---

## 5. Bileşenler (bu artımda üretilecekler)

### 5.1 shadcn `ui/*` (port, ihtiyaç kadar)
Overview'un kullandıkları öncelik: `card, button, badge, tooltip, scroll-area, separator, skeleton, sonner (toaster), dropdown-menu, utils(cn)`. Kalan geniş set (dialog, table, form, tabs, sheet…) sonraki ekranlar geldikçe eklenir (sessiz düşürme yok — burada bilinçli erteleniyor).

**Not (Base UI vs Radix):** shadcn'in ürettiği primitive'ler Radix değil **Base UI** (`@base-ui/react`) tabanlıdır — "başka bir elementi render et" davranışı Radix'in `asChild` prop'u yerine Base UI'ın `render` prop'u (`useRender` / `mergeProps`) ile sağlanır. Referanstan port edilecek bir bileşen `<Button asChild>` / `<Trigger asChild>` kullanıyorsa, port sırasında `render` prop'una çevrilmelidir.

### 5.2 Sentinel primitive'leri
- **ScoreBadge** — sayısal skor + seviye rengi + label; `scoreToLevel` ile.
- **Sparkline** — küçük SVG/Recharts trend çizgisi (momentum/KPI).
- **TokenAvatar** — symbol baş harfleri + deterministik renk.
- **WalletAddress** — kısaltılmış mono adres + copy butonu (toast feedback) + explorer linki.
- **KpiCard** — ana değer, % değişim, mini sparkline, tooltip, "son güncelleme", tone (positive/critical/warning/neutral).

### 5.3 Shell
- **Sidebar** — 17 nav öğesi (`nav.ts`), aktif route vurgusu (`usePathname`), collapse (Zustand), footer: RPC status / Solana network / Telegram durumu / trading mode / kullanıcı profili (statik/mock göstergeler).
- **Header** — global arama input'u (placeholder), network seçici, son güncelleme, RPC latency, priority-fee göstergesi, notification ikonu, **Emergency Pause** butonu, profil.
- **TradingModeBadge** — Paper/Shadow/Live; Live'da amber/kırmızı güvenlik etiketi.

### 5.4 Overview dashboard bileşenleri
- **KPI grid** — 8 kart (`useKpis`).
- **LiveTokenFeed** — sıralanabilir tablo (Age/Liq/Momentum/Creator), watchlist toggle, `<60s` satırlarda yeşil highlight, skor badge'leri, sparkline, signal etiketi (Buy/Watch/Avoid), hızlı aksiyonlar (watchlist/analyze/trade — bu artımda no-op + toast). Veri `useTokens` + canlı `subscribeTokens`.
- **OpportunityRadar** — Recharts scatter: X=Creator, Y=Momentum, boyut=likidite, renk=risk (`useRadar`).
- **AlertsTimeline** — canlı alert akışı, severity noktaları, zaman damgası (`useAlerts` + `subscribeAlerts`).

---

## 6. Yönlendirme (routing)
- Tüm 17 nav hedefi App Router route'u olarak var; Overview gerçek, diğerleri **Placeholder** (başlık + "Yakında" + kısa açıklama). Bu, shell navigasyonunu baştan tam çalışır kılar ve sonraki artımlar sadece ilgili `page.tsx`'i doldurur.
- Route grubu `(app)` shell layout'unu paylaşır; ileride `(auth)`/`(marketing)` grupları eklenebilir (şimdi değil).

---

## 7. Test stratejisi (TDD)
Önce testler, sonra implementasyon (superpowers:test-driven-development).
- **Saf mantık (zorunlu):** `format.ts` (formatAge/Price/Usd), `scoreToLevel` sınır değerleri (24/25/49/50/69/70/84/85), `riskMeta` eşleşmesi.
- **Adapter kontratı:** `mockApi` tüm `SentinelApi` metodlarını sağlıyor mu; `getApi()` env'e göre doğru impl'i döndürüyor mu; `subscribe*` unsubscribe fonksiyonu döndürüyor mu.
- **Komponent (kritik):** `ScoreBadge` skora göre doğru seviye/etiket; `WalletAddress` copy → toast; `LiveTokenFeed` sort davranışı ve `<60s` highlight; `KpiCard` tone renkleri.
- **Smoke:** Overview sayfası hook'lar mock'la render oluyor mu.
- Playwright E2E ve görsel regresyon → sonraki artım (şimdi kapsam dışı).

---

## 8. Kapsam dışı (bilinçli — sessiz düşürme yok)
Aşağıdakiler bu artımda **yapılmayacak**, sonraki artımlara işaretlendi:
- Diğer 11 ekranın içeriği (yalnız placeholder).
- Gerçek backend/AWS entegrasyonu, auth, Solana wallet-connect.
- Light tema; mobil-optimize layout'lar (desktop-first; responsive breakpoint'ler sonra).
- lightweight-charts fiyat grafikleri (Token Detail/Terminal ile gelir).
- Cytoscape wallet graph (Wallet Graph ekranı ile).
- Command palette, onboarding akışı, Telegram/alert formları, gerçek trade confirm modalı.
- Geniş shadcn seti (ihtiyaç oldukça eklenecek).

## 9. Kabul kriterleri (Definition of Done)
1. `npm run dev` ile uygulama açılıyor; dark Sentinel teması uygulanmış.
2. Sidebar (collapse dahil) + Header + TradingModeBadge çalışıyor; 17 route gezinilebiliyor (Overview gerçek, diğerleri placeholder).
3. Overview: 8 KPI kartı, LiveTokenFeed (sort + watchlist + canlı highlight), OpportunityRadar, AlertsTimeline — hepsi mock veriyle çalışıyor.
4. Tüm veri **hook → getApi() → mock** üzerinden akıyor; hiçbir bileşen mock'u doğrudan import etmiyor.
5. `subscribe*` mock stream ile en az bir canlı güncelleme görülüyor (yeni satır/alert).
6. Testler yeşil (`format`, `scoreToLevel`, adapter kontratı, ScoreBadge/WalletAddress/LiveTokenFeed).
7. `.env.example` `NEXT_PUBLIC_DATA_SOURCE`, `NEXT_PUBLIC_API_BASE_URL`, `NEXT_PUBLIC_WS_URL` içeriyor.

## 10. Açık riskler / notlar
- Tailwind v4 + shadcn + Next 15 uyumu: shadcn'in v4 kurulumu takip edilecek.
- RSC prefetch + mock (senkron) uyumu: mock async imzalı tutulacak ki http'ye geçiş sorunsuz olsun.
- Referans react-router → Next App Router: route path'leri `nav.ts`'te aynı, sadece `<Link>`/`usePathname` kullanılacak.
