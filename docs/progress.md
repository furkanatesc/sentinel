# SENTINEL — İlerleme Kaydı (PROGRESS)

> Yaşayan doküman. Bir iş kalemi bitince, karar değişince ya da karar ağacında
> dallanma olunca **aynı turda** güncellenir. Tek gerçek kaynaklar: ürün için
> `ROADMAP.md`, tasarım için `docs/design/sentinel-ui-ux-design.md`.
>
> Son güncelleme: 2026-07-30

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
| **3** | **Live Feed** (event terminali: 10 filtre + tablo + detay drawer) | 🔧 Spec onaylı, **plan yazıldı**, uygulanacak (SDD) |
| 4 | Wallet Graph (Cytoscape) | ⬜ |
| 5 | Creators (Creator Profile) | ⬜ |
| 6 | Strategies | ⬜ |
| 7 | Portfolio / Positions | ⬜ |
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

Increment 3 = **Live Feed** ekranı (öneri). Gerçek zamanlı event terminali: üst filtreler
(event type, launchpad, DEX, min likidite/creator score, risk, yaş…), event kartları/tablosu,
event açılınca sağdan detay drawer. Mevcut seam'e `getEvents`/`subscribeEvents` eklenir.
Onayınla spec→plan→SDD.

Alternatif olarak backlog'daki **Ayarlar (API key) + Polymarket** entegrasyonuna da
öncelik verilebilir (backend/secret-store bağımlılığı ile birlikte).
