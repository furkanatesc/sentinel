# SENTINEL Backend — Alt-proje 0 Design Spec
### Platform İskeleti (Go API + Railway Postgres, ilk dikey dilim)

- **Tarih:** 2026-08-04
- **Durum:** Onaylandı (2026-08-04) — plan yazılacak.
- **Önkoşul:** Frontend Increment 1–9 master'da + Vercel'de canlı (mock seam, `sentinel-brown-alpha.vercel.app`). Backend programı ayrıştırması onaylandı (bkz `docs/progress.md` → Backend programı).
- **Kaynaklar:** `docs/design/sentinel-ui-ux-design.md` (backend vizyonu: Go ingestion/serving + Python scoring/ML), `apps/web/lib/api/contract.ts` (`SentinelApi` — backend'in hedef API yüzeyi), `docs/progress.md`.

---

## 0. Program bağlamı (ayrıştırma)

Bu, SENTINEL **backend programının** ilk alt-projesidir. Tam prod backend vizyonu (kullanıcı onayı 2026-08-04) tek spec'e sığmayacak kadar büyük olduğu için 6 alt-projeye ayrıldı; her biri frontend kontratının (`SentinelApi`) bir dilimini gerçeğe çevirir:

| # | Alt-proje | Kontrat dilimi | Bağımlılık |
|---|---|---|---|
| **0** | **Platform iskeleti** (bu spec) | `getStrategies` (ilk dikey dilim) | — |
| 1 | Solana ingestion | `getTokens`/`getEvents`/`getKpis`/`getRadar`/`getToken` + `subscribe*` (+ WebSocket transport) | 0 |
| 2 | Scoring & graph (Python/ML) | `getCreators`/`getCreator`/`getWalletGraph` | 1 |
| 3 | Alerts & Telegram | `getAlerts`/`subscribeAlerts` + alert-kural CRUD + gerçek Telegram delivery | 1 |
| 4 | Strategies & backtest | `getStrategy`/`runBacktest` (gerçek motor) | 1, 2 |
| 5 | Trading engine | `getPortfolio`/`getPositions`/`getOrders`/`getTransactions`/`getTradeLogs`/`getMarketData`/`getCandles` | 1, 3, 4 |

Bu spec **yalnız Alt-proje 0**'dır. Sıra: 0 → 1 → 2/3 → 4 → 5.

---

## 1. Amaç ve kapsam

**Dikey dilim:** Bir Go API servisi + Railway yönetilen Postgres ayağa kaldır, **tek gerçek endpoint** (`getStrategies`) uçtan uca çalıştır, ve frontend'i endpoint-endpoint göçe hazır **hibrit adapter**'a çevirerek seam flip'ini kanıtla. Amaç minimal çalışan altyapı — sonraki alt-projeler bunun üzerine gerçek endpoint ekler.

**Kapsam DIŞI (bilinçli — sonraki alt-projeler):** Solana ingestion, scoring/ML, WebSocket/real-time transport (`subscribe*` mock kalır — Alt-proje 1), trading, **auth** (public read-only; Alt-proje 3+'ta yazma gelince eklenir), Python servisi (Alt-proje 2), diğer kontrat endpoint'leri, Docker/compose (Railway nixpacks Go'yu derler), OpenAPI codegen (hafif kontrat testi yeterli).

---

## 2. Clean Code & SOLID + reuse (ölçüt)

- **SRP:** Go tarafı katmanlı — `config` / `http` (router+handler) / `store` (repository+Postgres) ayrı; her biri tek iş.
- **OCP:** Frontend `LIVE_ENDPOINTS` config'i (yeni endpoint eklemek = listeye ekleme, `getApi()` değişmez); Go şeması migration ile genişler.
- **DIP:** Frontend bileşenleri yalnız `getApi()` görür (değişmez); Go handler'ı `StrategyStore` **interface**'ine bağımlı (Postgres impl arkasında — test için in-memory fake).
- **ISP:** Handler yalnız ihtiyacı olan store metodunu alır.
- **Reuse:** Mevcut `StrategyRow` tipi + `StrategiesListContent`/`StrategyCard` UI **değişmez**; Postgres seed'i mevcut `mock.ts` strateji verisinden türetilir (aynı şekil, tek gerçek kaynak).

---

## 3. Mimari

### 3.1 Go servisi (`apps/api-go/`)
```
apps/api-go/
├─ cmd/server/main.go        # bootstrap: config yükle, DB pool, router, graceful shutdown
├─ internal/config/          # env okuma (DATABASE_URL, PORT, CORS_ORIGIN)
├─ internal/http/            # chi router + handlers + CORS + JSON hata middleware
│   ├─ router.go             #   GET /healthz, GET /api/strategies
│   └─ strategies.go         #   handler → StrategyStore.List() → JSON
├─ internal/store/           # StrategyStore interface + postgresStore impl + fake (test)
├─ migrations/               # goose: 0001_strategies.sql (+ seed)
├─ go.mod
└─ README.md                 # local run + Railway deploy adımları
```
- Router: **chi** (minimal, idiomatic). Structured logging: `log/slog`. Graceful shutdown (SIGTERM).
- **`GET /healthz`** → `200 {"status":"ok"}` (DB ping dahil).
- **`GET /api/strategies`** → `200 []StrategyRow` (json). Go struct'ları `StrategyRow`'u json tag'leriyle **birebir** yansıtır.

### 3.2 Postgres (Railway yönetilen)
- `strategies` tablosu; kolonlar `StrategyRow` alanlarını yansıtır.
- Şema **goose migration** ile; seed migration `mock.ts` strateji verisini taşır (deterministik, aynı id'ler).

### 3.3 Frontend seam — hibrit adapter (çekirdek)
- **`lib/api/http.ts`:** `getStrategies()` gerçek `fetch(\`${NEXT_PUBLIC_API_BASE_URL}/api/strategies\`)` → `StrategyRow[]`; non-200/network → reject. Diğer metotlar `notReady` kalır.
- **`lib/api/index.ts` `getApi()`:** tümü-ya-hiç yerine **config-driven composite**:
  - `LIVE_ENDPOINTS: Set<keyof SentinelApi>` (bu turda `{"getStrategies"}`).
  - `NEXT_PUBLIC_DATA_SOURCE==="http"` iken: metot `LIVE_ENDPOINTS`'te ise `httpApi`'ye, değilse `mockApi`'ye bağlanır. `="mock"` (veya set değil) iken: her şey mock.
  - Bileşenler yine `getApi()` görür (DIP korunur).
- **Canlı endpoint runtime'da mock'a DÜŞMEZ:** yalnız "henüz implemente değil" olanlar mock kullanır. Canlı bir endpoint down ise gerçek hata durumu gösterilir (sessiz mock demoyu yanıltır).
- **Env:** `.env.example`'a `NEXT_PUBLIC_API_BASE_URL=` eklenir; Vercel'e `NEXT_PUBLIC_API_BASE_URL=<railway-url>` + `NEXT_PUBLIC_DATA_SOURCE=http`.

### 3.4 Kontrat sadakati (Go ↔ TS)
- Go struct'ları `StrategyRow`'u json tag'leriyle yansıtır. Drift'e karşı **hafif kontrat testleri**: Go tarafı seed'in beklenen JSON anahtarlarına serialize olduğunu; frontend http-adapter testi mock'lanmış `fetch`'in `StrategyRow[]`'a maplendiğini doğrular. (OpenAPI codegen ertelendi — followup.)

### 3.5 Deploy (Railway)
- `apps/api-go` ikinci Railway servisi (root=`apps/api-go`, nixpacks Go build). Postgres eklentisi `DATABASE_URL` enjekte eder.
- CORS yalnız Vercel origin'ine izin. Deploy sonrası migration çalışır (release komutu / başlangıçta `goose up`).
- **CI:** push'ta `go build ./...` + `go test ./...` (minimal GitHub Actions veya Railway build).

---

## 4. Veri akışı

Strategies sayfası (RSC prefetch `getStrategies`) → `getApi()` (hibrit) → `httpApi.getStrategies()` → Railway `GET /api/strategies` → Go handler → `StrategyStore.List()` → Postgres sorgu → `[]StrategyRow` JSON → frontend `StrategyRow[]` map → **aynı `StrategiesListContent` UI, gerçek veri**. Diğer 8 ekran mock ile çalışmaya devam eder.

---

## 5. Hata yönetimi

- **Go:** doğru status kodları (200/400/500/503), JSON hata gövdesi (`{"error":"..."}`), structured log, DB connection pool, DB erişilemezse `/healthz` + `/api/strategies` → 503.
- **Frontend:** `httpApi.getStrategies` non-200/network → reject → mevcut React Query error state'i (`StrategiesListContent` loading/error dalları). Canlı endpoint mock'a düşmez.
- **Güvenlik:** CORS kilitli; endpoint public read-only (hassas veri yok); `DATABASE_URL` yalnız Railway env'inde (repoya/log'a yazılmaz); repo public → secret sızıntısı yok (mevcut ön-kontrol temiz).

---

## 6. Test stratejisi

- **Go:** handler unit (fake `StrategyStore` → JSON şekil/anahtar doğrulama), `store` integration (Postgres — Railway test DB veya `pgx` + geçici şema; ağır testcontainer yerine hafif), `/healthz` testi, contract test (seed → beklenen JSON anahtarları).
- **Frontend:** `http.ts` `getStrategies` map testi (mock'lanmış `fetch` → `StrategyRow[]`), hibrit `getApi()` testi (`DATA_SOURCE=http` → `getStrategies` http'ye, diğerleri mock'a; `=mock` → hepsi mock).
- **Canlı doğrulama:** Railway `/healthz` + `/api/strategies` 200; Vercel `DATA_SOURCE=http` iken Strategies ekranı gerçek API'den render, diğer ekranlar mock ile sağlam.

---

## 7. Kabul kriterleri

1. `apps/api-go` derlenir; `go test ./...` geçer.
2. Railway'de `/healthz` → 200; `/api/strategies` → seeded `StrategyRow[]` JSON.
3. Frontend `NEXT_PUBLIC_DATA_SOURCE=http` iken **Strategies ekranı gerçek API'den** render; diğer 8 ekran mock ile regresyonsuz çalışır (hibrit doğrulandı).
4. Kontrat testleri (Go + frontend) yeşil.
5. Mevcut frontend test paketi (175/175) regresyonsuz.
6. Yaşayan dokümanlar (progress + memory) backend programı + Alt-proje 0 ile güncel.

---

## 8. Kullanıcı aksiyonları (harici/ücretli — DUR ve iste)

Şu adımlar kullanıcının hesabını gerektirir; kod/migration/CI hazır olunca **ayrı ayrı sorulacak**:
- Railway hesabı + GitHub repo bağlama + **Postgres eklentisi** ekleme.
- Railway servis env'leri (otomatik `DATABASE_URL`; gerekiyorsa `CORS_ORIGIN`).
- Vercel'e `NEXT_PUBLIC_API_BASE_URL=<railway-url>` + `NEXT_PUBLIC_DATA_SOURCE=http`.

---

## 9. Kapsam dışı / sonraki alt-projeler

- **WebSocket/real-time transport:** ayrıştırmada 0'a yazılmıştı; **bilinçle Alt-proje 1'e (ingestion) taşındı** — `getStrategies` dilimi WS gerektirmez (sessiz düşürme yok).
- **Auth, ingestion, scoring, trading, Python, diğer endpoint'ler:** ilgili alt-projelerde.
- **OpenAPI codegen:** kontrat drift büyürse ileride (followup).
