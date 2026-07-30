# SENTINEL Frontend — Increment 3 Design Spec
### Live Feed (Gerçek Zamanlı Event Terminali)

- **Tarih:** 2026-07-30
- **Durum:** Onay bekliyor
- **Önkoşul:** Increment 1 (shell + seam + Overview) ve Increment 2 (Token Detail) master'da.
- **Kaynaklar:** `docs/design/sentinel-ui-ux-design.md` (Ekran 2), `ROADMAP.md` (event tipleri §1), `docs/progress.md`.

---

## 1. Amaç ve kapsam

`/live-feed` altında gerçek zamanlı bir **event terminali**: üst filtre çubuğu + canlı event tablosu + event'e tıklayınca sağdan **detay drawer**. Kullanıcı sidebar'dan "Canlı Akış"a gelir.

Bu artımda: **tablo görünümü** + **tam filtre seti (10 filtre)** + detay drawer + canlı akış. Event'ler mock stream'den gelir; backend gelince aynı seam gerçek WebSocket'e geçer.

---

## 2. Clean Code & SOLID ilkeleri (ölçüt)

- **SRP:** `FeedFilters` (kontroller), `FeedTable` (satırlar), `EventDetailDrawer` (detay), `EventTypeBadge` (tip rozeti) ayrı; **filtreleme mantığı bileşende değil** `lib/feed/filter.ts`'te saf `filterEvents` fonksiyonunda (test edilebilir).
- **OCP:** Event tipleri `EVENT_TYPE_DEFS` registry'sinden (rozet + filtre çipi + tablo bundan türer); risk çipleri `RiskLevel`'dan. Yeni event tipi = registry entry. `filterEvents` her filtre için ayrı predicate (yeni filtre = predicate ekle).
- **DIP:** Bileşenler `useEvents()`/`getApi()` soyutlamasına bağımlı; hiçbiri `lib/api/mock`'u import etmez.
- **ISP:** `FeedTable` `{ events, onRowClick }`; `FeedFilters` `{ value, onChange }`; drawer `{ event, onClose }` — dar prop'lar.
- **Clean code:** küçük odaklı dosyalar, anlamlı isimler, DRY, pristine test, TDD.

---

## 3. Mimari

### 3.1 Veri seam genişlemesi (`lib/api`)
```ts
export type EventType =
  | "new_mint" | "metadata_created" | "pool_created" | "first_swap"
  | "liquidity_added" | "liquidity_removed" | "creator_sell" | "whale_buy"
  | "suspicious_cluster" | "score_change" | "strategy_signal";

export interface FeedEvent {
  id: string;
  type: EventType;
  symbol: string; mint: string;
  launchpad: string;   // "Pump.fun" | "Raydium" | "Moonshot" | "Meteora" | ...
  dex: string;         // "Raydium" | "Meteora" | "Orca" | "Jupiter" | ...
  liquidity: number;
  creatorScore: number;
  riskLevel: RiskLevel;        // @/lib/format
  tokenAgeSeconds: number;
  volume5m: number;
  holderGrowthPct: number;
  severity: AlertSeverity;     // @/lib/format (nokta rengi)
  detail: string;
  time: string; ts: number;
  watchlisted: boolean;
}
```
- `SentinelApi.getEvents(): Promise<FeedEvent[]>` (mock ~24 seed event).
- `SentinelApi.subscribeEvents(cb: (e: FeedEvent) => void): () => void` (mock: `setInterval` ile yeni event üretir; http'de WebSocket).
- `httpApi` → `getEvents: notReady`, `subscribeEvents: () => () => {}`.
- `qk.events`; `useEvents()` (queries.ts); `useLiveEvents()` (live.ts) — `subscribeEvents` → `setQueryData(qk.events, prev => [e, ...prev].slice(0, 200))` (başa ekle, 200 ile sınırla).

### 3.2 Filtre modeli (saf mantık — SRP/OCP)
```ts
export interface FeedFilters {
  types: EventType[];        // boş = tümü
  risks: RiskLevel[];        // boş = tümü
  launchpad: string;         // "all" | belirli
  dex: string;               // "all" | belirli
  minLiquidity: number;      // 0 = filtre yok
  minCreatorScore: number;   // 0 = filtre yok
  maxAgeSeconds: number | null; // null = filtre yok
  minVolume: number;
  minHolderGrowth: number;
  watchlistOnly: boolean;
}
export const EMPTY_FILTERS: FeedFilters;   // hepsi kapalı varsayılan
export function filterEvents(events: FeedEvent[], f: FeedFilters): FeedEvent[];
```
`filterEvents` her filtre için tek predicate uygular (10 filtrenin tümü). `lib/feed/filter.ts`. Tamamen saf + TDD.

### 3.3 Event tip registry (OCP)
`lib/feed/event-defs.ts`:
```ts
export interface EventTypeDef { key: EventType; label: string; severity: AlertSeverity; icon: LucideIcon; }
export const EVENT_TYPE_DEFS: EventTypeDef[]; // 11 tip, Türkçe label + ikon + severity
```
Rozet, filtre çipleri ve tablo bundan türetilir.

### 3.4 Bileşenler (`components/feed/`)
```
components/feed/
├─ event-defs.ts             # EVENT_TYPE_DEFS registry (OCP)
├─ EventTypeBadge.tsx        # tek event tipi rozeti (ikon+label+renk)
├─ FeedFilters.tsx           # üst filtre çubuğu (10 filtre); {value,onChange}
├─ FeedTable.tsx             # event tablosu; canlı prepend+highlight; {events,onRowClick}
├─ EventDetailDrawer.tsx     # shadcn Sheet; seçili event detayı + Token Detail linki
└─ LiveFeedContent.tsx       # kompozisyon: filtreler + tablo + drawer; useEvents+useLiveEvents
lib/feed/filter.ts           # filterEvents (saf)
app/(app)/live-feed/page.tsx # server: getEvents prefetch + hydration
```
- **FeedFilters:** Event tipi (çip toggle, çoklu), Risk (çip toggle, çoklu), Launchpad (select), DEX (select), Min likidite (sayı input), Min creator score (sayı input), Max token yaşı (sayı input, saniye/dakika), Min hacim (sayı input), Min holder büyümesi (sayı input %), Watchlist only (toggle). "Temizle" butonu → `EMPTY_FILTERS`.
- **FeedTable:** kolonlar — Zaman, Event (EventTypeBadge), Token (avatar+symbol+kısaltılmış mint), Launchpad/DEX, Likidite, Creator (ScoreBadge), Risk, Aksiyon (Detay/Token'a git). Yeni event başa düşer + kısa highlight (`<10s` veya `isNew`). `overflow-y-auto`, `max-h` ile sınırlı (Overview'daki AlertsTimeline dersi — sınırsız büyümez).
- **EventDetailDrawer:** shadcn Sheet (sağdan). Event tipi, token (symbol+mint+`WalletAddress`), launchpad/DEX, likidite, creator score, risk, yaş, hacim, holder büyümesi, açıklama, zaman. "Token Detayına Git" → `/tokens/[symbol]`.
- **LiveFeedContent:** `"use client"`; `useEvents()` + `useLiveEvents()`; filter state (useState<FeedFilters>) + seçili event; `filterEvents` uygular; FeedTable'a filtrelenmiş liste verir; satır tıklama → drawer.

### 3.5 Sayfa & shell
- `app/(app)/live-feed/page.tsx` — server component, `getEvents` prefetch + `HydrationBoundary`, `<LiveFeedContent/>`. Placeholder route bu sayfayla değişir.
- shadcn eklemeleri: **sheet, select, input** (varsa yeniden kullan). Çoklu-seçim event/risk için hazır bileşen yerine **çip toggle** (button) — yeni ağır bağımlılık yok.

---

## 4. Kapsam dışı (bilinçli)
- Event akışının sonsuz virtual-scroll'u (şimdilik 200 cap + overflow scroll).
- Kaydedilmiş filtre görünümleri (saved views), CSV export.
- Gerçek backend/WebSocket (mock stream; seam hazır).
- Drawer içinde tam token analizi (özet + Token Detail linki yeterli).
- Diğer ekranlar (Discover, Tokens listesi vb.) hâlâ placeholder.

## 5. Test stratejisi (TDD)
- `filterEvents` (saf): her 10 filtre için ayrı test (event type, risk, launchpad, dex, min liquidity, min creator, max age, min volume, min holder growth, watchlist only) + kombinasyon + boş filtre = hepsi.
- `EMPTY_FILTERS` doğru varsayılan.
- `getEvents`/`subscribeEvents` adapter kontratı; `useEvents` hook; `useLiveEvents` cache prepend + cap.
- `EventTypeBadge` registry'den doğru label/renk.
- `FeedTable` satır tıklama `onRowClick(event)` çağırır; canlı highlight.
- `FeedFilters` çip toggle + input değişimi `onChange` emit eder; "Temizle" `EMPTY_FILTERS` verir.
- `EventDetailDrawer` event alanlarını + Token linkini render eder.
- `LiveFeedContent` smoke: filtre uygulayınca tablo daralır.

## 6. Kabul kriterleri
1. Sidebar "Canlı Akış" → `/live-feed`: filtre çubuğu + canlı event tablosu (SSR prefetch + hydration).
2. 10 filtre çalışır; birden fazla filtre birlikte uygulanır; "Temizle" sıfırlar.
3. Canlı akış: birkaç saniyede yeni event başa düşer + highlight; liste 200 ile sınırlı, içeride scroll.
4. Event satırına tıklayınca sağdan drawer açılır; detay + "Token Detayına Git" linki `/tokens/[symbol]`'e gider.
5. Tüm veri `component → useEvents → getApi() → mock`; hiçbir bileşen mock import etmez.
6. Testler yeşil; build başarılı; SOLID/clean-code ölçütü karşılanır.
