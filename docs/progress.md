# SENTINEL — İlerleme Kaydı (PROGRESS)

> Yaşayan doküman. Bir iş kalemi bitince, karar değişince ya da karar ağacında
> dallanma olunca **aynı turda** güncellenir. Tek gerçek kaynaklar: ürün için
> `ROADMAP.md`, tasarım için `docs/design/sentinel-ui-ux-design.md`.
>
> Son güncelleme: 2026-08-03

## Genel bakış

SENTINEL = Solana'da yeni çıkan tokenları saniyeler içinde tespit eden, creator/wallet
güven skorlaması yapan, açıklanabilir risk analizi üreten, Telegram bildirimi gönderen ve
(ileride) otomatik trade eden gerçek zamanlı istihbarat + trading platformu.

- **Backend:** AWS'de, Go (event ingestion, düşük gecikmeli worker'lar, trading dispatch) +
  Python (scoring, clustering, ML, backtest, RAG). *Henüz başlanmadı.*
- **Frontend:** `apps/web/` (monorepo). Next.js (App Router, server-first). *Increment 1 tamam.*

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
| **7** | **Portfolio / Positions** (portföy genel bakış + KPI + 4 grafik + pozisyon tablosu + detay drawer) | ✅ **Branch `feat/portfolio` tamam — 119/119 test, build + whole-branch review + fix wave temiz; görsel doğrulandı; merge onayı bekliyor** |
| 8 | Trading Terminal | ⬜ |
| 9 | Backtesting (+ Event Replay) | ⬜ |
| 10 | Alerts / Telegram | ⬜ |
| 11 | Research Assistant | ⬜ |
| 12 | System Health | ⬜ |

Her ekran kendi spec → plan → implementasyon (SDD) döngüsünden geçer; hepsi mevcut
shell + veri seam'i üzerine kurulur.

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
- 2026-08-03 — **Increment 7 (Portfolio / Positions) branch `feat/portfolio` tamamlandı; merge onayı bekliyor.** SDD ile 13 implementasyon task'ı (fresh subagent + task review döngüsü), her biri spec ✅ + kalite Approved; ardından final whole-branch review (opus) + tek fix wave + scoped re-review temiz. İki read-only rota: `/portfolio` (KPI grid + equity curve + PnL-by-strateji bar + risk-allocation donut + win/loss bar + açık pozisyon özeti) ve `/positions` (sıralanabilir tablo + risk filtresi + detay drawer). Seam: `getPortfolio`/`getPositions` + `usePortfolio`/`usePositions`, httpApi→`notReady`; hiçbir bileşen mock import etmez (DIP). **Reuse kararı:** `EquityCurve` `components/strategy/` → `components/sentinel/` taşındı (`title?`/`color?` prop + renkten türetilen gradient id — eski sabit-id followup'ını kapattı); Strategies detay regresyonsuz aynı bileşeni kullanıyor. `MetricTile`'a geriye-uyumlu `valueColor?` eklendi (PnL renklendirmesi). OCP: `lib/position/risk-filter.ts` (`POSITION_RISK_LEVELS` + `pnlColor`). Read-only aksiyonlar (Kapat / SL-TP): canlı modda `toast.warning`, kağıt/gölge modda simüle `toast` — gerçek trade yok (Trading Terminal / Orders artımına ertelendi). Kapsam dışı (bilinçli): gerçek emir gönderimi, PnL-by-creator/token-age grafikleri, tablo saved views/CSV/kolon customization/virtual scroll, pozisyon düzenleme formu. Fix wave (5 bulgu): risk-allocation dilimlerine ayrık hex renkler (iki-yeşil çakışması), tokenRisk mock aralığı genişletildi + "Sonuç yok" boş-durum, PositionsContent loading/error (Skeleton), Türkçe tooltip name/formatter, yaş sıralaması `parseInt` (leksikografik değil). 119/119 test, `npm run build` başarılı (iki statik rota), whole-branch review "Ready to merge (with fixes)" → fix wave sonrası tüm bulgular ADDRESSED. Görsel doğrulandı (2026-08-03): her iki rota + KPI PnL renkleri + 4 grafik + donut ayrık renkler + filtre daraltma/Temizle + satır→drawer + aksiyon toast. Deferred minor'lar: `docs/superpowers/followups-frontend.md`.
- 2026-08-01 — **Increment 6 (Strategies) branch `feat/strategies` tamamlandı ve master'a merge edildi (4d43309).** SDD ile 11 implementasyon task'ı (fresh subagent + task review döngüsü), her biri spec ✅ + kalite Approved. `/strategies` liste (durum filtresi) + `/strategies/[id]` read-only detay. Seam: `getStrategies`/`getStrategy` + `useStrategies`/`useStrategy`; OCP `STATUS_DEFS`/`CONDITION_LABELS` + `formatCondition`; SRP bileşen ağacı (StatusBadge/StrategyCard/ConditionList/StrategyPerformancePanel/BacktestSummaryPanel/EquityCurve/VersionHistory/AuditLog/StrategiesListContent/StrategyDetailContent); paylaşımlı `MetricTile` reuse; EquityCurve OverviewTab MiniChart desenini takip eder. Read-only kapsam (builder/deploy/execution bilinçli dışarıda). 101/101 test, `npm run build` başarılı, whole-branch review (opus) **"Ready to merge: Yes"** (0 Critical/Important). Görsel doğrulandı (liste + filtre + detay: koşullar/risk/performans/equity curve/backtest/launchpad/versiyon/audit). Deferred minor'lar: `docs/superpowers/followups-frontend.md`.
- 2026-07-30 — **Increment 4 (Wallet Graph) tamamlandı ve master'a merge edildi** (540436e). `/wallet-graph` Cytoscape entity graph: `WalletGraph` seam (getWalletGraph/useWalletGraph), `NODE_TYPE_DEFS`(8)+`EDGE_TYPE_DEFS`(9) registry → stylesheet/legend/filtre türer, saf `toCytoscapeElements`/`neighborsOf`/`buildStylesheet`, canvas dynamic ssr:false, stabil-instance focus fade. Yeni dep cytoscape. 71/71 test. Görsel doğrulandı. Parked minor: stale-fade (bkz followups).

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
- Takip listesi: `docs/superpowers/followups-frontend.md`
- Knowledge graph: `graphify-out/graph.html` (+ `GRAPH_REPORT.md`)

## Sırada

**Önce:** `feat/portfolio` branch'inin master'a merge'i (kullanıcı onayı bekliyor).

Sonraki artım: Increment 8 = **Trading Terminal** ekranı. Manuel/otomatik emir gönderimi (buy/sell,
size, SL/TP), emir defteri/fills, pozisyon açma-kapama akışı — Increment 7'de read-only bırakılan
gerçek trade aksiyonları burada devreye girer. Seam'e `getOrders`/emir gönderim endpoint'leri eklenir.
Kendi spec → plan → SDD döngüsünden geçer.

**Strategies devamı (sonraki artım, ertelendi):** "Create Strategy" no-code condition builder stepper
(8 adım, IF creator>75 & safety>70 & liquidity>25k THEN buy...), strateji düzenle/deploy, live paper/shadow
toggle, versiyon geri alma — Increment 6'da bilinçli kapsam dışıydı.

Alternatif olarak backlog'daki **Ayarlar (API key) + Polymarket** entegrasyonuna da
öncelik verilebilir (backend/secret-store bağımlılığı ile birlikte).
