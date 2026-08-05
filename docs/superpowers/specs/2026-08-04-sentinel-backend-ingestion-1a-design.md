# SENTINEL Backend Alt-proje 1 (Solana Ingestion) — Slice 1a Design Spec
### Gerçek-zaman çoklu-launchpad yeni-token tespiti + WebSocket transport

- **Tarih:** 2026-08-04
- **Durum:** Onaylandı (2026-08-04) — **plan yazıldı (2026-08-05):** `docs/superpowers/plans/2026-08-05-sentinel-backend-ingestion-1a.md` (11 task, SDD). Implementasyon sırada; canlı Helius yalnız deploy'da.
- **Önkoşul:** Backend Alt-proje 0 (Platform iskeleti) master'da + Railway/Vercel'de canlı (Go `apps/api-go`, hibrit `getApi()`, Postgres, goose migration, CI). Helius hesabı + API key (implementasyon/deploy'da; bkz repo kökü `api_key_alinacakplatformlar.md`).
- **Kaynaklar:** `docs/design/sentinel-ui-ux-design.md` (Live Feed / Discover), `apps/web/lib/api/contract.ts` (`SentinelApi`), `apps/web/lib/api/types.ts` (`TokenRow`/`FeedEvent`/`EventType`), `apps/web/lib/feed/` (EVENT_TYPE_DEFS, sources), `docs/progress.md`.

---

## 0. Program bağlamı

Backend programının **Alt-proje 1**'idir. Kullanıcı **tam ingestion vizyonu** seçti (2026-08-04); tek spec'e sığmadığından sıralı dilimlere bölündü, hepsi tam vizyona doğru inşa edilir:

| Slice | Kapsam | Kontrat |
|---|---|---|
| **1a (bu spec)** | Gerçek-zaman çoklu-launchpad yeni-token tespiti + WebSocket transport | `getEvents`/`getTokens` + `subscribeEvents`/`subscribeTokens` |
| 1b (sonra) | Fiyat+likidite+holder+momentum enrichment + agregasyon | `getKpis`/`getRadar`/`getToken` + TokenRow tam dolu |

`creatorScore`/`safetyScore`/`riskLevel` alanları **Alt-proje 2 (scoring/ML)**'ye aittir; 1a'da nötr placeholder.

---

## 1. Amaç ve kapsam

Go servisine her-zaman-açık bir **ingestion worker** + frontend için **WebSocket hub** ekle: Helius üzerinden birden çok launchpad program'ının log'larını dinle, yeni-token doğum olaylarını decode et, Postgres'e yaz ve bağlı frontend'lere gerçek-zaman push et. `getEvents`/`getTokens` gerçek (DB) olur; `subscribeEvents`/`subscribeTokens` gerçek WebSocket'e döner (mock `setInterval` yerine). Ürünün çekirdek değeri: **"saniyede yeni token yakala."**

**Kapsam DIŞI (bilinçli — sonraki dilim/alt-projeler):**
- `getKpis`/`getRadar`/`getToken` gerçekleştirme (Slice 1b — mock kalır).
- Fiyat/likidite/vol5m/holders/momentum/spark enrichment (Slice 1b).
- `creatorScore`/`safetyScore`/`riskLevel` (Alt-proje 2 — nötr placeholder).
- Event tipleri: liquidity_removed/creator_sell/whale_buy (1b/enrichment), suspicious_cluster/score_change (Alt-proje 2), strategy_signal (Alt-proje 4).
- Geçmiş backfill (fresh DB boş başlar, event geldikçe dolar — 1a'da backfill yok).
- Auth (Alt-proje 0'daki gibi public read-only; WS de public).

---

## 2. Clean Code & SOLID + reuse (ölçüt)

- **OCP — decoder registry:** her launchpad = `LaunchpadDecoder` arayüzünü uygulayan bir decoder; yeni launchpad = yeni decoder + registry kaydı (worker/hub değişmez).
- **SRP:** ingestion worker (bağlantı+yönlendirme) / decoder (parse) / store (persist) / ws hub (broadcast) / helius client (transport) ayrı.
- **DIP:** worker `LaunchpadDecoder` + `EventStore` interface'lerine bağımlı; frontend yalnız `getApi()`/`subscribe*` görür.
- **Reuse:** mevcut `FeedEvent`/`TokenRow`/`EventType` tipleri (backend JSON birebir); frontend `useLiveEvents`/`useLiveTokens` (cache patch) + `EVENT_TYPE_DEFS`/`LAUNCHPADS`/`DEXES` değişmez; Alt-proje 0 hibrit adapter (`LIVE_ENDPOINTS`'e ekleme).

---

## 3. Mimari

### 3.1 Go servisi (`apps/api-go`) — yeni bileşenler
```
internal/ingest/
├─ worker.go        # ingestion worker: Helius WS bağlantısı, log→decoder yönlendirme, backoff reconnect, dedup
├─ helius.go        # Helius WS client (logsSubscribe) + DAS getAsset (metadata) client
├─ decoder.go       # LaunchpadDecoder interface + Registry
├─ decode_pumpfun.go / decode_pumpswap.go / decode_raydium.go   # somut decoder'lar (ilk 3)
└─ types.go         # ingest-içi normalize event tipi
internal/ws/
└─ hub.go           # WebSocket hub: client kayıt/çıkış, topic (events/tokens) broadcast
internal/store/     # (genişleme) events.go + tokens.go: insert + recent-query; migration 0002
internal/api/       # (genişleme) events/tokens handler + /ws endpoint
```
- **Worker:** boot'ta goroutine. `Registry.ProgramIDs()` için Helius `logsSubscribe`. Gelen log → `Registry.Match(programId).Decode(log)` → `[]FeedEvent` + `TokenRow` upsert → `store` + `hub.Broadcast`. WS koparsa **exponential backoff reconnect**; kısa süreli in-memory **dedup** (mint+type+slot).
- **Decoder registry (OCP):**
```go
type Decoded struct { Event FeedEvent; Token TokenRow }
type LaunchpadDecoder interface {
    ProgramID() string
    Launchpad() string                 // "Pump.fun" | "pump.swap" | "Raydium" (LAUNCHPADS ile uyumlu)
    Decode(log LogNotification) ([]Decoded, error)
}
```
  İlk somut: pump.fun (`create` → new_mint/metadata_created), pump.swap + Raydium (pool_created/first_swap/liquidity_added). Moonshot/Meteora sonra (framework hazır). **Not:** `Launchpad()` değeri frontend `LAUNCHPADS` (`lib/feed/sources.ts`: Pump.fun/Raydium/Moonshot/Meteora) ile uyumlu olmalı; listede olmayan bir launchpad (ör. "pump.swap") emit edilecekse `sources.ts`'e eklenir (küçük frontend değişikliği) — yoksa filtre çipinde görünmez (event yine akar).
- **Metadata:** pump.fun create name/symbol/uri içerir → doğrudan. Diğerlerinde mint → Helius **DAS `getAsset`** (cache'li, tekrar çağrı yok).
- **WS hub (`/ws`):** frontend bağlanır; topic bazlı (events/tokens) JSON mesaj yayını (`{topic, payload}`). Client yazma-pump + ping/pong keepalive.

### 3.2 Postgres (migration 0002)
- `events` tablosu — `FeedEvent` alanları (append-only; recent-N sorgusu, ts index).
- `tokens` tablosu — `TokenRow` alanları (mint PK, upsert; enrichment/scoring alanları nullable/nötr default).
- Alan doldurma (1a): mint/symbol/name/launchpad/dex/type/tokenAgeSeconds/ts/detail gerçek; price/liquidity/vol5m/holders/momentum/spark/holderGrowthPct → nötr (0/null); creatorScore/safetyScore → nötr (0), riskLevel → `"medium"` placeholder (Alt-proje 2'ye kadar).

### 3.3 Frontend seam (hibrit'e ekleme)
- `lib/api/http.ts`: `getEvents`/`getTokens` gerçek fetch (`/api/events`, `/api/tokens` → `FeedEvent[]`/`TokenRow[]`). `subscribeEvents`/`subscribeTokens` → `/ws`'e **gerçek WebSocket** (topic'e abone; mesajda cb çağır; unsubscribe = socket kapat). Mock `setInterval` yerine.
- `lib/api/live-endpoints.ts`: `LIVE_ENDPOINTS`'e `getEvents`, `getTokens`, `subscribeEvents`, `subscribeTokens` eklenir. (`getKpis`/`getRadar`/`getToken` MOCK kalır.)
- **WS base URL:** `NEXT_PUBLIC_API_BASE_URL`'den türetilir (`https`→`wss`). Mevcut `useLiveEvents`/`useLiveTokens` (cache patch) değişmez — sadece kaynak gerçek olur.
- **Dürüst placeholder:** nötr skorlar UI'da "sahte" görünmemeli — score badge 0/nötr için "—"/"bilinmiyor" gösterir (gerekirse `ScoreBadge`/`EVENT_SEVERITY`'ye küçük nötr-durum uyarlaması; ayrı ele alınır).

---

## 4. Veri akışı

Helius WS (`logsSubscribe`, N program) → worker → decoder(programId) → normalize `FeedEvent`+`TokenRow` → (a) Postgres insert/upsert, (b) WS hub broadcast → frontend `/ws` → `useLiveEvents`/`useLiveTokens` cache patch → Live Feed/Overview canlı güncellenir. İlk yükte `getEvents`/`getTokens` DB'den son-N okur (RSC prefetch + client).

---

## 5. Hata yönetimi / operasyonel

- **Worker:** Helius WS kopması → exponential backoff reconnect (yeniden subscribe); decode hatası → log + drop (pipeline durmaz); DAS getAsset başarısız → metadata'sız (symbol=kısaltılmış mint) devam.
- **WS hub:** yavaş/ölü client → bounded kanal + drop/disconnect; ping/pong.
- **Servis:** DATABASE_URL zorunlu (Alt-proje 0); `HELIUS_API_KEY` yoksa worker başlamaz + net log (REST yine çalışır — mock'suz boş DB). Graceful shutdown worker+hub'ı kapatır.
- **Maliyet/ölçek:** kalıcı WS + Helius abonelikleri sürekli kullanım; Railway uzun-çalışan servis. Free tier ile başla, gerekince yükselt.

---

## 6. Test stratejisi

- **Go:** decoder unit testleri **kayıtlı Helius log fixture'larıyla** (deterministik, ağsız — her launchpad için örnek `logsNotification` → beklenen `FeedEvent`/`TokenRow`); WS hub (subscribe/broadcast/disconnect); store (events/tokens insert+recent-query, integration DATABASE_URL guard'lı — Alt-proje 0 deseni); registry (programId→decoder eşleşme).
- **Frontend:** `getEvents`/`getTokens` map testleri (mock fetch); WS-client testi (mock `WebSocket` → mesaj → cb çağrılır, unsubscribe socket kapar); hibrit `getApi()` (yeni LIVE_ENDPOINTS http'ye, getKpis mock).
- **Canlı doğrulama (deploy):** Helius key ile worker gerçek yeni token'ları yakalar; `/api/events` gerçek veri; `/strategies` gibi Live Feed canlı akar. (Canlı Helius = integration/deploy, Postgres deseni gibi.)

---

## 7. Kabul kriterleri

1. `apps/api-go` derlenir; `go test ./...` geçer (decoder/hub/store/registry testleri; canlı Helius skip/integration).
2. Deploy'da (Helius key ile): worker Helius'a bağlanır, gerçek yeni-token olaylarını decode edip DB'ye yazar + WS'e yayar.
3. Frontend `DATA_SOURCE=http` iken **Live Feed gerçek Solana yeni-token akışını** gösterir (getEvents gerçek + subscribeEvents WS canlı); Overview token feed gerçek. Skorlar dürüst-nötr ("henüz yok"), sahte değil.
4. `getKpis`/`getRadar`/`getToken` + diğer ekranlar mock ile regresyonsuz (hibrit korunur).
5. Yeni launchpad eklemek = yeni decoder + registry kaydı (worker/hub değişmeden — OCP kanıtı).
6. Frontend test paketi + Go testleri yeşil; yaşayan dokümanlar + memory güncel.

---

## 8. Kullanıcı aksiyonları (harici/ücretli — implementasyon/deploy'da DUR ve iste)

- **Helius** hesabı + API key → Railway `api-go` servisine `HELIUS_API_KEY` (bkz `api_key_alinacakplatformlar.md`). Free tier ile başla.
- (Deploy) Railway servisinin kalıcı WS + worker'ı çalıştırdığını doğrula; Vercel env zaten http modunda.

---

## 9. Açık noktalar (implementasyon/plan aşamasında netleşir)

- Helius `logsSubscribe` vs enhanced `transactionSubscribe` seçimi (log parse edilebilirliğine göre; logsSubscribe başlangıç).
- Her launchpad program ID'si + log/instruction formatı (decoder yazımında kayıtlı fixture'larla doğrulanır).
- Nötr-skor UI uyarlaması (ScoreBadge "—") ayrı küçük frontend işi.
- Recent-N pencere boyutu (events/tokens) ve WS backpressure eşiği.
