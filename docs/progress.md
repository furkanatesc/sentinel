# SENTINEL — İlerleme Kaydı (PROGRESS)

> Yaşayan doküman. Bir iş kalemi bitince, karar değişince ya da karar ağacında
> dallanma olunca **aynı turda** güncellenir. Tek gerçek kaynaklar: ürün için
> `ROADMAP.md`, tasarım için `docs/design/sentinel-ui-ux-design.md`.
>
> Son güncelleme: 2026-08-05 (Alt-proje 1 slice 1a **CANLI + DOĞRULANDI** — Railway backend gerçek pump.fun token'larını ingest ediyor; master'a merge + deploy edildi)

## Genel bakış

SENTINEL = Solana'da yeni çıkan tokenları saniyeler içinde tespit eden, creator/wallet
güven skorlaması yapan, açıklanabilir risk analizi üreten, Telegram bildirimi gönderen ve
(ileride) otomatik trade eden gerçek zamanlı istihbarat + trading platformu.

- **Backend:** Go (serving/ingestion/trading) + Python (scoring/ML/backtest, sonra). **Program başladı
  (2026-08-04):** 6 alt-projeye ayrıştırıldı; **Alt-proje 0 (Platform iskeleti) spec'lendi**. Hosting: Railway
  (AWS uzun-vade). Bkz "Backend programı" bölümü.
- **Frontend:** `apps/web/` (monorepo). Next.js (App Router, server-first). *Increment 1–9 tamam, Vercel'de canlı.*

## Teknoloji kararları (onaylı)

| Katman | Seçim | Not |
|---|---|---|
| Framework | Next.js (App Router), server-first | Pure CSR SPA değil; RSC + client island |
| Styling | Tailwind CSS v4 | Sentinel dark token'ları |
| Komponent | shadcn/ui (**Base UI** tabanlı) | Radix değil; ported bileşenler `render` prop kullanır |
| Server state | TanStack Query | RSC prefetch + HydrationBoundary |
| Client state | Zustand | sidebar collapse, trading mode |
| Real-time | WebSocket (adapter arkasında) | Increment 1'de mock stream |
| Grafikler | Recharts (metrik); lightweight-charts (fiyat, sonra); Cytoscape.js (wallet graph, sonra) | |
| Form | React Hook Form + Zod | minimal |
| Test | Vitest + React Testing Library | TDD |

**Kritik mimari:** bileşenler `component → hook → getApi() → (mock \| http)` seam'inden okur;
hiçbiri mock'u doğrudan import etmez. Backend gelince yalnız `lib/api/http.ts` yazılır +
`NEXT_PUBLIC_DATA_SOURCE=http`.

## Frontend artım durumu

| # | Artım | Durum |
|---|---|---|
| **1** | **İskele + Design System + App Shell + Overview Dashboard** | ✅ **Tamam — master'a merge (05546e7), 20/20 test** |
| **2** | **Token Detail** (Header + 4 skor + Overview + Risk + Açıklanabilir Skor) | ✅ **Tamam — master'a merge (fc36f0a), 37/37 test** |
| **3** | **Live Feed** (event terminali: 10 filtre + tablo + detay drawer) | ✅ **Tamam — master'a merge (2cbfe4e), 55/55 test** |
| **4** | **Wallet Graph** (Cytoscape entity graph: 8 node + 9 edge + filtre + detay) | ✅ **Tamam — master'a merge (540436e), 71/71 test** |
| **5** | **Creators** (liste + profil: reputation + metrik + token geçmişi + davranış) | ✅ **Tamam — master'a merge (5904288), 81/81 test** |
| **6** | **Strategies** (liste + read-only detay: koşullar/risk/performans/equity/backtest/versiyon/audit) | ✅ **Tamam — master'a merge (4d43309), 101/101 test** |
| **7** | **Portfolio / Positions** (portföy genel bakış + KPI + 4 grafik + pozisyon tablosu + detay drawer) | ✅ **Tamam — master'a merge (22b4f0e), 119/119 test** |
| **8** | **Trading Terminal** (4 bölme: watchlist + candlestick grafik + order paneli + alt sekmeler) | ✅ **Tamam — master'a merge (41342d6), 147/147 test** |
| **9** | **Backtesting** (parametre formu + simüle çalıştır + 10 metrik + 6 grafik; Event Replay ertelendi) | ✅ **Tamam — master'a merge (6029ff6), 175/175 test; Vercel'de canlı** |
| 10 | Alerts / Telegram | ⏸️ **Duraklatıldı** — frontend-mock yerine backend ile birlikte gelecek (Backend Alt-proje 3). Bkz aşağı. |
| 11 | Research Assistant | ⬜ (frontend-mock; sıralama backend sonrası netleşir) |
| 12 | System Health | ⬜ (frontend-mock; sıralama backend sonrası netleşir) |

Her frontend ekranı kendi spec → plan → implementasyon (SDD) döngüsünden geçer; hepsi mevcut
shell + veri seam'i üzerine kurulur. **Karar (2026-08-04):** Increment 10 (Alerts/Telegram) frontend-mock
olarak yapılmak yerine **backend programına** geçildi; Alerts/Telegram yeteneği (frontend + gerçek Telegram)
Backend Alt-proje 3'te teslim edilecek (sessiz düşürme yok — aşağıdaki Backend programı bölümü).

## Backend programı

Tam prod backend vizyonu (kullanıcı onayı 2026-08-04) tek spec'e sığmayacağından 6 alt-projeye ayrıştırıldı;
her biri frontend kontratının (`apps/web/lib/api/contract.ts` → `SentinelApi`) bir dilimini gerçeğe çevirir.
Frontend endpoint-endpoint göç eder (**hibrit adapter**: `LIVE_ENDPOINTS`'te olan gerçek, kalanı mock).

**Stack kararları (2026-08-04, onaylı):** Go API (serving/ingestion/trading) — Python (scoring/ML) Alt-proje 2'de;
Hosting **Railway** (Go servisi + yönetilen Postgres), AWS uzun-vade; DB Postgres (goose migration).

| # | Alt-proje | Kontrat dilimi | Durum |
|---|---|---|---|
| **0** | **Platform iskeleti** (Go API + Railway Postgres, `getStrategies` dikey dilimi + hibrit adapter) | `getStrategies` | ✅ **TAMAM — master'a merge (ae9b8ee), Railway+Vercel'de CANLI ve doğrulandı (2026-08-04)** |
| 1 | Solana ingestion (+ WebSocket transport) | `getTokens`/`getEvents`/`getKpis`/`getRadar`/`getToken` + `subscribe*` | ✅ **Slice 1a CANLI (master, Railway'de deploy) — gerçek pump.fun token'ları ingest ediliyor & doğrulandı.** `getKpis`/`getRadar`/`getToken` → Slice 1b |
| 2 | Scoring & graph (Python/ML) | `getCreators`/`getCreator`/`getWalletGraph` | ⬜ |
| 3 | **Alerts & Telegram** (kural CRUD + gerçek Telegram delivery) | `getAlerts`/`subscribeAlerts` | ⬜ (Increment 10 buraya taşındı) |
| 4 | Strategies & backtest (gerçek motor) | `getStrategy`/`runBacktest` | ⬜ |
| 5 | Trading engine (Jupiter, emir icra, pozisyon) | `getPortfolio`/`getPositions`/`getOrders`/`getTransactions`/`getTradeLogs`/`getMarketData`/`getCandles` | ⬜ |

**Sıra:** 0 → 1 → 2/3 → 4 → 5. Gerekçe: iskele önkoşul; ingestion çekirdek değer + çoğu ekranın veri kaynağı;
trading en riskli (gerçek para) en son (tasarım paper-default). Alt-proje 0 spec: `docs/superpowers/specs/2026-08-04-sentinel-backend-platform-skeleton-design.md`; plan: `docs/superpowers/plans/2026-08-04-sentinel-backend-platform-skeleton.md`.

**Alt-proje 0 teslim (2026-08-04, branch `feat/backend-skeleton`):** SDD ile 7 kod task'ı (fresh subagent + task-review döngüsü; Task 1'de go.mod `1.25.0`→`1.23` fix round'u) + final whole-branch review (opus) **"Ready to merge: Yes"** (0 Critical/Important, 2 Minor) + 1 fix wave (`db.Close` hata-yolu sızıntısı) + re-review temiz. Yeni Go servisi `apps/api-go/` (chi router, katmanlı config/api/store, `GET /api/strategies` + `/healthz`, graceful shutdown), Postgres store (goose migration + `SeedRows` 6 satır mock ile birebir + `ON CONFLICT DO NOTHING`), frontend hibrit `getApi()` (`LIVE_ENDPOINTS`={getStrategies}; canlı runtime'da mock'a düşmez; diğer 8 ekran regresyonsuz mock). CI `.github/workflows/api-go.yml` (go 1.23). 179/179 frontend test + Go build/vet/test yeşil. **DB round-trip runtime doğrulanmadı** (yerel Postgres yok) → Railway deploy'da doğrulanacak. Ertelenen: CORS preflight Allow-Methods/Headers (Alt-proje 1, YAGNI), `ON CONFLICT` update-etmez (ileri migration). **DEPLOY EDİLDİ + DOĞRULANDI (2026-08-04):** Railway servisi (root `apps/api-go`, Postgres eklentisi, `CORS_ORIGIN`, public domain **`sentinel-production-e14d.up.railway.app`**) — `/healthz` 200, `/api/strategies` 6 satır → **DB round-trip canlıda kanıtlı**. Vercel env `NEXT_PUBLIC_API_BASE_URL`+`NEXT_PUBLIC_DATA_SOURCE=http`. Canlı doğrulama: `/strategies` gerçek Railway API'den 6 kart (`status: success`), Overview + diğer ekranlar mock ile regresyonsuz (hibrit çalışıyor). **Deploy dersi:** Vercel "Redeploy" eski build'i sundu (kod + `NEXT_PUBLIC_*` env stale kaldı → eski `getApi` her şeyi notReady httpApi'ye yönlendirip TÜM ekranları bozdu); çözüm master'a taze commit push → cache'siz otomatik build (commit 64d2acb). İleride: env/kod değişince Vercel'de "Use existing Build Cache" KAPALI ile deploy ya da yeni commit.

**Alt-proje 1 Slice 1a teslim (2026-08-05, branch `feat/backend-ingestion-1a`):** SDD ile 10 kod task'ı (fresh
subagent + task-review döngüsü) tamam ve tek tek review edildi; whole-branch review + deploy bu turda (Task 11).
Yeni Go paketleri: `internal/ingest/` (decoder registry [OCP] + pump.fun log-only CreateEvent decoder + Raydium
CPMM tx-based decoder + Helius client [logsSubscribe WS + getTransaction + DAS getAsset] + ingestion worker
[route/dedup/persist/broadcast + exponential-backoff reconnect]) ve `internal/ws/` (frontend-facing WebSocket
hub, topic broadcast). Postgres migration `0002_create_events_tokens.sql` (events + tokens tabloları);
`OpenPostgres` artık `store.Bundle{Strategies, Events, Tokens}` döndürüyor. Yeni endpoint'ler: `GET /api/events`,
`GET /api/tokens` (gerçek, DB-backed), `GET /ws` (WebSocket; `events` topic'i tekil `FeedEvent`, `tokens` topic'i
tam `TokenRow[]` snapshot yayınlar). Frontend: `getEvents`/`getTokens` gerçek fetch + `subscribeEvents`/
`subscribeTokens` gerçek WebSocket (`lib/api/ws.ts`); `LIVE_ENDPOINTS`'e +4; dürüst nötr skor gösterimi (0 →
"—"). Yeni bağımlılıklar: `github.com/gagliardetto/solana-go`, `github.com/coder/websocket`,
`github.com/mr-tron/base58`. **Go 1.23 → 1.24 yükseltildi** (solana-go transitive dep zorladı; `go.mod` + CI
`.github/workflows/api-go.yml` ikisi de 1.24). Env: `HELIUS_API_KEY` (worker bunsuz başlamaz, REST/mock yine
çalışır), `EVENTS_WINDOW` (default 200). **Kapsam kararları:** pump.fun + Raydium CPMM decoder'ları somut
teslim edildi; PumpSwap + Moonshot/Meteora decoder'ları framework-ready ama **ertelendi**. Teslim edilen event
tipleri: `new_mint`, `metadata_created`, `pool_created` (`first_swap`/`liquidity_added` → Slice 1b).
`getKpis`/`getRadar`/`getToken` **MOCK kalmaya devam ediyor** (Slice 1b). Skorlar dürüst-nötr placeholder
(0 / riskLevel "medium") — Alt-proje 2 (scoring) dolduracak. **Canlı Helius + DB round-trip yalnızca DEPLOY'da
doğrulanacak** (yerel Postgres/key yok) — Alt-proje 0 ile aynı desen. Ertelenen maddelerin tam listesi:
`docs/superpowers/followups-frontend.md` "Backend Alt-proje 1 slice 1a — deferred" bölümü.

**MERGE + DEPLOY + CANLI DOĞRULANDI (2026-08-05):** Whole-branch review (opus) "With fixes" → 1 Critical
(SubscribeLogs canlı kopmada dönmüyordu → reconnect deadlock) + 2 Important (tokens index, ws.ts reconnect) fix
wave + scoped re-review temiz. Master'a `--no-ff` merge (kullanıcı onayı), `git push origin master`. Railway
backend (Go 1.24) build oldu; migration 0002 uygulandı; `HELIUS_API_KEY` Railway Variables'a girildi → worker
Helius WS'e bağlandı. **Deploy hotfix (`4e2396f`):** İlk deploy'da worker gerçek create'leri alıyor ama her buy'da
`decode error` (`kısa buffer`) — kök neden: `hasMarker` substring'i her buy'ın ATA `Instruction: CreateIdempotent`
satırıyla eşleşip TradeEvent'i CreateEvent sanıyordu. Fix: exact create marker (trimmed-suffix) + CreateEvent byte
offset otomatik-tespit (emit! 8B / emit_cpi! 16B; uri `://` doğrulaması). **Doğrulama:** `/api/events` & `/api/tokens`
200 + gerçek veri — 14 gerçek pump.fun token (Stark/PEPE/ORANGECHIP/…), doğru mint/symbol/name, 2 event/token
(new_mint+metadata_created), skorlar dürüst-nötr (0 / "medium"). **AÇIK GÜVENLİK MADDESİ:** Railway'deki
`HELIUS_API_KEY` hâlâ sohbete sızan `5f5e…` key'i — kullanıcı rotate ETMEDİ (re-paste etti); Helius'ta Regenerate
edilip taze key Railway'e konmalı. **Frontend (Vercel) push'ta yeniden build edildi** — Live Feed/Overview artık
gerçek akışı göstermeli (görsel doğrulama kullanıcıda).

### Backlog (kuyruk — henüz spec'lenmedi)
- **Entegrasyonlar için Ayarlar sekmesi (API key girişi)** — `/settings` altında; kullanıcı
  entegrasyon API key'lerini girer. **Güvenlik zorunlulukları:** key'ler frontend'de saklanmaz;
  HTTPS ile backend'e iletilir ve **secret store**'da (AWS Secrets Manager/KMS) tutulur; repoya
  commit veya log'a yazılmaz; UI'da maskelenir. (Kullanıcı talebi: 2026-07-30.)
- **Polymarket entegrasyonu** — prediction-market veri kaynağı; veri seam'ine yeni endpoint
  ailesi (`getPolymarket*`). Yerleşim (Research/Discover/yeni ekran) spec aşamasında netleşecek;
  yukarıdaki Ayarlar/API-key altyapısına bağımlı. (Kullanıcı talebi: 2026-07-30.)

### Increment 1 — teslim edilenler (2026-07-30)

- Next.js 16 App Router iskele, Tailwind v4 + Sentinel dark token'ları, Inter + JetBrains Mono,
  Vitest + RTL harness.
- Tasarım sistemi: `lib/format.ts` (skor seviyeleri, riskMeta/severityMeta, formatter'lar).
- shadcn/ui primitive'leri (Base UI tabanlı) + `cn`.
- Veri seam'i: `lib/api/` (types, contract, **mockApi**, http stub, `getApi()`).
- TanStack Query (`providers`, `getQueryClient`, `qk`, hook'lar) + Zustand (`ui`, `session`).
- Sentinel primitive'leri: ScoreBadge, Sparkline, TokenAvatar, WalletAddress, KpiCard.
- App shell: Sidebar (17 nav, collapse), Header (arama, Emergency Pause, Live şerit),
  TradingModeBadge (Paper/Shadow/Live). 17 route (Overview gerçek + 16 placeholder).
- **Overview dashboard:** 8 KPI kartı, LiveTokenFeed (sort + watchlist + `<60s` highlight),
  OpportunityRadar (Recharts), AlertsTimeline. RSC prefetch + hydration.
- Mock real-time stream → query cache patch (`lib/hooks/live.ts`).

**Çalıştırma:** `cd apps/web && npm run dev`

## Kararlar günlüğü

- 2026-07-30 — Frontend framework: **Next.js** (server-first), Vite SPA değil. Gerekçe: backend AWS'de;
  RSC + client island hibriti "client-side gitmeyeceğiz" isteğiyle örtüşüyor.
- 2026-07-30 — Monorepo: frontend `apps/web/`; backend ileride `apps/api-go/`, `apps/scoring-py/`.
- 2026-07-30 — Komponent kütüphanesi: shadcn'in güncel default'u **Base UI** (`@base-ui/react`),
  Radix değil. Ported bileşenler `asChild` yerine `render` prop kullanır.
- 2026-07-30 — Increment 1 SDD ile 10 task; her task review + final whole-branch review temiz; master'a merge.
- 2026-07-30 — **UI dili Türkçe** olacak (tüm görünen metinler; teknik tokenlar/simgeler hariç). Review sonrası tüm arayüz + mock veri etiketleri Türkçeleştirildi (`cfc9aba`, `b43812f`).
- 2026-07-30 — Post-Increment-1 polish: scrollbar dark temaya uyumlu hale getirildi; Opportunity Radar boş-render bug'ı (Recharts sizing) düzeltildi; `apps/web/AGENTS.md`/`CLAUDE.md` scaffold gürültüsü kaldırıldı.
- 2026-07-30 — **Increment 2 (Token Detail) tamamlandı ve master'a merge edildi** (fc36f0a). `/tokens/[mint]` rotası, config-driven skorlar (SCORE_DEFS, Manipulation ters polarity), TAB_DEFS sekme registry, getToken seam genişlemesi. Clean-code/SOLID review ölçütü uygulandı. 37/37 test.
- 2026-07-30 — **Clean code + SOLID** kullanıcı önceliği: tüm spec/plan/review'larda ölçüt (SRP/OCP/DIP/ISP; config-driven; küçük dosyalar).
- 2026-07-30 — **Increment 3 (Live Feed) tamamlandı ve master'a merge edildi** (2cbfe4e). `/live-feed` event terminali: `FeedEvent` seam (getEvents/subscribeEvents/useLiveEvents 200-cap), `EVENT_TYPE_DEFS` registry, saf `filterEvents` (10 filtre), FeedFilters/FeedTable/EventDetailDrawer (shadcn Sheet). 55/55 test. Görsel doğrulandı.
- 2026-07-31 — **Increment 5 (Creators) tamamlandı ve master'a merge edildi** (5904288). `/creators` liste + `/creators/[address]` profil: `getCreators`/`getCreator` seam, reputation Token Detail'in `ScoreCard`+`ExplainableScore`'unu reuse (ScoreCard'a `hideExplain` prop eklendi), 8 metrik (paylaşımlı `MetricTile`), token geçmişi tablosu (outcome/liquidity/riskFlags), davranış paterni, Wallet Graph creator node linki. 81/81 test. Görsel doğrulandı. Parked: mock derivation dup (bkz followups).
- 2026-08-04 — **Backend programı BAŞLADI (kullanıcı kararı).** Kullanıcı "Increment 10 spec yazalım" derken tam prod backend'e geçmek istedi. Brainstorming ile ayrıştırıldı: tam backend vizyonu (Solana ingestion + scoring/ML + trading) tek spec'e sığmaz → **6 alt-proje** (0 Platform iskeleti → 1 ingestion → 2 scoring → 3 alerts/telegram → 4 strategies/backtest → 5 trading), her biri `SentinelApi` kontratının bir dilimini gerçeğe çevirir; frontend **hibrit adapter** (`LIVE_ENDPOINTS`) ile endpoint-endpoint göç eder. **Stack (onaylı):** Go API (Python scoring Alt-proje 2), Hosting **Railway** (Go + yönetilen Postgres, AWS uzun-vade), DB Postgres/goose. **Alt-proje 0 (Platform iskeleti) spec yazıldı** (`docs/superpowers/specs/2026-08-04-sentinel-backend-platform-skeleton-design.md`): Go servisi + Railway Postgres + tek gerçek endpoint `getStrategies` uçtan uca + frontend hibrit adapter (seam flip kanıtı); auth yok (public read-only, sonra), WS Alt-proje 1'e taşındı. **Increment 10 (Alerts/Telegram) frontend-mock DURAKLATILDI** → Backend Alt-proje 3'te frontend+gerçek Telegram ile gelecek (bkz Backend programı bölümü).
- 2026-08-04 — **Increment 9 (Backtesting) tamamlandı ve master'a merge edildi (6029ff6).** SDD ile 9 task (fresh subagent + task-review döngüsü, hepsi spec ✅ + kalite Approved; Task 1 priorityFee seed eksiğini 1 fix round'da kapattı). Tasarım Ekran 9'un **backtest sonuç yarısı** (Event Replay bilinçli sonraki artıma ertelendi). `/backtesting` nav placeholder'ı gerçeğe döndü (rename yok). Yerleşim: sol ~300px `BacktestParamsForm` (useStrategies dropdown[DIP] + 4 registry select + 5 sayısal alan, saf `validateParams` — 6 alan Türkçe hata span'i, Çalıştır gating) + ana sonuç alanı (submittedParams state → boş/loading/error/sonuç). Seam: **simüle, deterministik-seeded** `runBacktest(params)` + `useBacktest(params|null)` (enabled-on-submit; `qk.backtest`=`["backtest",JSON.stringify(params)]` → her run taze cache key, stale-flash yok), httpApi→`notReady`; hiçbir bileşen mock import etmez (DIP). Sonuçlar: **10 metrik** (config-driven `BACKTEST_METRIC_DEFS` + `MetricTile` reuse + `pnlColor`) + **6 grafik** — `EquityCurve` reuse (Sermaye Eğrisi) + DrawdownChart(Area, domain [dataMin,0]) + MonthlyReturnChart/PnlByScoreChart (Bar + pnlColor Cells) + TradeDistributionChart (Bar) + EntryExitChart (ComposedChart: fiyat Line + al/sat Scatter, trade.time===price.t merge). RSC page qk.strategies prefetch + HydrationBoundary. **Reuse:** `EquityCurve`/`MetricTile`/`pnlColor`/`useStrategies`/native-select Header deseni. Kapsam dışı bilinçle: Event Replay (look-ahead engelleme + timeline playback), gerçek backend backtest motoru, kaçırılan-fırsat/rug-timeline grafikleri, parametre preset kaydetme, sonuç export/karşılaştırma. 164/164 test, `npm run build` başarılı (`○ /backtesting` statik). Görsel doğrulandı (2026-08-04): boş durum → form dolu (strateji dropdown strategies'ten) → Çalıştır → 10 metrik (Net PnL renkli) + 6 grafik → sermaye 100→250 yeniden çalıştır sayılar değişti (Net PnL -18 kırmızı → 3 yeşil) → sermaye=0 alan altında "Sermaye 0'dan büyük olmalı" + çalıştırma bloke. Whole-branch review (opus) **"Ready to merge: Yes"** (0 Critical/Important, 4 Minor) → tek fix wave (order-sensitive seed: `seedOf` paylaşımlı olduğu için dokunulmadı, backtest seed'i `v.repeat(i+1)` ile pozisyon-ağırlıklı → transpozisyon çakışması cebirsel kapandı; + 10-satır per-param sensitivity table testi) → scoped re-review CLEAN. **164→175 test**, master'a merge `6029ff6`, Vercel'de canlı. Deferred minor'lar: `docs/superpowers/followups-frontend.md` (Inc9 bölümü).
- 2026-08-03 — **Increment 8 (Trading Terminal) branch `feat/terminal` tamamlandı ve master'a merge edildi (41342d6).** SDD ile 14 implementasyon/wiring task'ı (fresh subagent + task-review döngüsü, hepsi spec ✅ + kalite Approved; Task 8 OrderPanel 2 fix round'da hata-render eksiğini kapattı), ardından final whole-branch review (opus) **"Ready to merge: Yes"** (0 Critical/Important) + tek fix wave + scoped re-review temiz. Tasarım Ekran 7'nin tamamı: `/terminal` rotası (eski "Emirler"/`/orders` placeholder'ı **"Terminal"/`/terminal`** olarak yeniden adlandırıldı) — 4 bölme: sol token watchlist (aktif token seçer), orta market data başlığı + **candlestick fiyat grafiği (lightweight-charts v4, `next/dynamic ssr:false`** — Cytoscape deseni), sağ order paneli (kontrollü form + `validateOrder`/`simulateOrder` + simulation status) + **`OrderConfirmDialog`** (yeni shadcn Base-UI `Dialog` primitive), alt sekmeli panel (Pozisyonlar[Inc7 `PositionsTable` reuse] / Emirler / İşlemler / Loglar). Seam: `getCandles`/`getMarketData`/`getOrders`/`getTransactions`/`getTradeLogs` + hook'ları, httpApi→`notReady`; hiçbir bileşen mock import etmez (DIP). **Emir davranışı: tam simüle + durumsuz** — canlı→`toast.warning` (güçlü uyarı, gerçek trade YOK), kağıt/gölge→simüle `toast`; **`SentinelApi`'de hiç mutation metodu yok → gerçek-trade yolu yapısal olarak imkânsız** (güvenlik özelliği review'da yapısal olarak doğrulandı). Yeni dep: `lightweight-charts@^4.2.3` (v4 pin; v5 seri API'sini değiştirdi). OCP: `lib/terminal/order-defs.ts` registries + saf `lib/terminal/order-logic.ts`. Kapsam dışı (bilinçli): gerçek emir/blockchain gönderimi, durumlu emir yaşam döngüsü, order book/depth, otomatik exit otomasyonu, RHF+Zod (kontrollü form yeterli), mobil tam terminal. Fix wave: paylaşımlı `sortPositions` helper (DRY, `lib/position/sort.ts` — BottomTabsPanel + PositionsContent ortak kullanır), OrdersTable iptal testi non-vacuous (mock `orders[0].status="open"` garantisi), inert `sizePct`/`trailingPct` + `onRowClick` no-op yorumları. 147/147 test, `npm run build` başarılı (20 rota, /terminal statik). Görsel doğrulandı (2026-08-03): token-switch header+grafik günceller, candlestick render, order paneli + onay modalı (düşük-skor uyarısı) + kağıt-mod simüle toast (durumsuz — Emirler tablosuna emir düşmez), alt sekme geçişleri. Deferred minor'lar: `docs/superpowers/followups-frontend.md`.
- 2026-08-03 — **Increment 7 (Portfolio / Positions) branch `feat/portfolio` tamamlandı ve master'a merge edildi (22b4f0e).** SDD ile 13 implementasyon task'ı (fresh subagent + task review döngüsü), her biri spec ✅ + kalite Approved; ardından final whole-branch review (opus) + tek fix wave + scoped re-review temiz. İki read-only rota: `/portfolio` (KPI grid + equity curve + PnL-by-strateji bar + risk-allocation donut + win/loss bar + açık pozisyon özeti) ve `/positions` (sıralanabilir tablo + risk filtresi + detay drawer). Seam: `getPortfolio`/`getPositions` + `usePortfolio`/`usePositions`, httpApi→`notReady`; hiçbir bileşen mock import etmez (DIP). **Reuse kararı:** `EquityCurve` `components/strategy/` → `components/sentinel/` taşındı (`title?`/`color?` prop + renkten türetilen gradient id — eski sabit-id followup'ını kapattı); Strategies detay regresyonsuz aynı bileşeni kullanıyor. `MetricTile`'a geriye-uyumlu `valueColor?` eklendi (PnL renklendirmesi). OCP: `lib/position/risk-filter.ts` (`POSITION_RISK_LEVELS` + `pnlColor`). Read-only aksiyonlar (Kapat / SL-TP): canlı modda `toast.warning`, kağıt/gölge modda simüle `toast` — gerçek trade yok (Trading Terminal / Orders artımına ertelendi). Kapsam dışı (bilinçli): gerçek emir gönderimi, PnL-by-creator/token-age grafikleri, tablo saved views/CSV/kolon customization/virtual scroll, pozisyon düzenleme formu. Fix wave (5 bulgu): risk-allocation dilimlerine ayrık hex renkler (iki-yeşil çakışması), tokenRisk mock aralığı genişletildi + "Sonuç yok" boş-durum, PositionsContent loading/error (Skeleton), Türkçe tooltip name/formatter, yaş sıralaması `parseInt` (leksikografik değil). 119/119 test, `npm run build` başarılı (iki statik rota), whole-branch review "Ready to merge (with fixes)" → fix wave sonrası tüm bulgular ADDRESSED. Görsel doğrulandı (2026-08-03): her iki rota + KPI PnL renkleri + 4 grafik + donut ayrık renkler + filtre daraltma/Temizle + satır→drawer + aksiyon toast. Deferred minor'lar: `docs/superpowers/followups-frontend.md`.
- 2026-08-01 — **Increment 6 (Strategies) branch `feat/strategies` tamamlandı ve master'a merge edildi (4d43309).** SDD ile 11 implementasyon task'ı (fresh subagent + task review döngüsü), her biri spec ✅ + kalite Approved. `/strategies` liste (durum filtresi) + `/strategies/[id]` read-only detay. Seam: `getStrategies`/`getStrategy` + `useStrategies`/`useStrategy`; OCP `STATUS_DEFS`/`CONDITION_LABELS` + `formatCondition`; SRP bileşen ağacı (StatusBadge/StrategyCard/ConditionList/StrategyPerformancePanel/BacktestSummaryPanel/EquityCurve/VersionHistory/AuditLog/StrategiesListContent/StrategyDetailContent); paylaşımlı `MetricTile` reuse; EquityCurve OverviewTab MiniChart desenini takip eder. Read-only kapsam (builder/deploy/execution bilinçli dışarıda). 101/101 test, `npm run build` başarılı, whole-branch review (opus) **"Ready to merge: Yes"** (0 Critical/Important). Görsel doğrulandı (liste + filtre + detay: koşullar/risk/performans/equity curve/backtest/launchpad/versiyon/audit). Deferred minor'lar: `docs/superpowers/followups-frontend.md`.
- 2026-07-30 — **Increment 4 (Wallet Graph) tamamlandı ve master'a merge edildi** (540436e). `/wallet-graph` Cytoscape entity graph: `WalletGraph` seam (getWalletGraph/useWalletGraph), `NODE_TYPE_DEFS`(8)+`EDGE_TYPE_DEFS`(9) registry → stylesheet/legend/filtre türer, saf `toCytoscapeElements`/`neighborsOf`/`buildStylesheet`, canvas dynamic ssr:false, stabil-instance focus fade. Yeni dep cytoscape. 71/71 test. Görsel doğrulandı. Parked minor: stale-fade (bkz followups).

- 2026-08-05 — **Backend Alt-proje 1 Slice 1a kod tamamlandı (branch `feat/backend-ingestion-1a`).** SDD ile
  10 kod task'ı (fresh subagent + task-review döngüsü, hepsi spec ✅ + kalite Approved) + Task 11 (bu doküman
  turu — living docs). Detay: yukarıdaki "Backend programı" bölümü "Alt-proje 1 Slice 1a teslim" paragrafı.
  Whole-branch review + master'a merge + Railway/Helius deploy henüz yapılmadı (Task 11 sonrası sıradaki adım).

## Açık takip maddeleri

Bloke etmeyen maddeler `docs/superpowers/followups-frontend.md`'de. Öne çıkanlar:
- **httpApi ile:** WalletAddress truncation'ı sunum katmanına taşı; `NEXT_PUBLIC_DATA_SOURCE`
  flip edip Overview'un aynı render'ı ürettiğini doğrulayan seam swappability testi.
- **CI hijyeni:** `typecheck` script + tsconfig `vitest/globals`; `engines >=20`.

## İlgili dokümanlar

- Ürün yol haritası: `ROADMAP.md`
- Tasarım spec'i (12 ekran): `docs/design/sentinel-ui-ux-design.md`
- Increment 1 spec: `docs/superpowers/specs/2026-07-30-sentinel-frontend-increment-1-design.md`
- Increment 1 plan: `docs/superpowers/plans/2026-07-30-sentinel-frontend-increment-1.md`
- **Backend Alt-proje 0 spec:** `docs/superpowers/specs/2026-08-04-sentinel-backend-platform-skeleton-design.md`
- **Backend Alt-proje 1 slice 1a spec:** `docs/superpowers/specs/2026-08-04-sentinel-backend-ingestion-1a-design.md`
- **API key checklist:** `api_key_alinacakplatformlar.md` (repo kökü)
- Takip listesi: `docs/superpowers/followups-frontend.md`
- Knowledge graph: `graphify-out/graph.html` (+ `GRAPH_REPORT.md`)

## Sırada

**Backend Alt-proje 0 CANLI ve doğrulandı** (Railway `sentinel-production-e14d.up.railway.app` + Vercel http modu;
`/strategies` gerçek API'den, diğer ekranlar mock — hibrit çalışıyor).

**Şimdi: Backend Alt-proje 1 — Slice 1a kod TAMAM** (branch `feat/backend-ingestion-1a`, `getEvents`/`getTokens`
+ `subscribe*` gerçeğe döndü — LIVE_ENDPOINTS'e +4). Sırada: whole-branch review (opus) → kullanıcı onayıyla
master'a merge → **deploy DUR-noktası:** Railway'e `HELIUS_API_KEY` girmeden önce kullanıcıdan Helius key'i
**rotate** ettirilir (sohbete sızan key iptal), taze key Railway Variables'a girilir; `/api/events` + `/ws`
canlı doğrulanır (Live Feed gerçek Solana akışı). `getKpis`/`getRadar`/`getToken` 1a'da mock (→Slice 1b
enrichment). Sonra: Slice 1b → Alt-proje 2/3 → 4 → 5.

**Frontend Increment 10 (Alerts/Telegram):** frontend-mock olarak DURAKLATILDI; Alerts/Telegram yeteneği
(frontend + gerçek Telegram delivery) Backend Alt-proje 3'te teslim edilecek. Increment 11 (Research) / 12
(System Health) frontend-mock ekranlarının sırası backend ilerledikçe netleşecek.

**Backtesting devamı (ertelendi):** **Event Replay** — look-ahead bias'sız timeline oynatma (playback
state'li oynatıcı + look-ahead engelleme), kaçırılan-fırsat/rug-timeline grafikleri, parametre preset
kaydetme, sonuç export/karşılaştırma. Gerçek backtest motoru Backend Alt-proje 4'e (Python) bağlı.

**Trading Terminal devamı (sonraki artım, ertelendi):** gerçek emir/blockchain gönderimi (Jupiter route,
tx sign/submit/retry), durumlu emir yaşam döngüsü (gönderilen emrin listeye/pozisyona yansıması), order
book/market depth, DCA & trailing/creator-sale/risk-score tetikli otomatik exit'ler — Increment 8'de
bilinçli kapsam dışıydı (backend trading engine gerektirir).

**Strategies devamı (sonraki artım, ertelendi):** "Create Strategy" no-code condition builder stepper
(8 adım, IF creator>75 & safety>70 & liquidity>25k THEN buy...), strateji düzenle/deploy, live paper/shadow
toggle, versiyon geri alma — Increment 6'da bilinçli kapsam dışıydı.

Alternatif olarak backlog'daki **Ayarlar (API key) + Polymarket** entegrasyonuna da
öncelik verilebilir (backend/secret-store bağımlılığı ile birlikte).
