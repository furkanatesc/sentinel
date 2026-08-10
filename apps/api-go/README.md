# SENTINEL API (Go)

Backend Alt-proje 0 (platform iskeleti) + Alt-proje 1 slice 1a (Solana ingestion + WS transport).
Endpoint'ler: `GET /api/strategies`, `GET /api/events`, `GET /api/tokens`, `GET /api/creators`,
`GET /api/creator/{address}`, `GET /ws` (WebSocket).

## Yerel çalıştırma
```bash
cd apps/api-go
go run ./cmd/server           # DATABASE_URL yoksa in-memory fake store
# Postgres ile:
DATABASE_URL=postgres://user:pass@localhost:5432/sentinel PORT=8080 \
  CORS_ORIGIN=http://localhost:3000 HELIUS_API_KEY=... go run ./cmd/server
```
`GET http://localhost:8080/healthz` → `{"status":"ok"}`
`GET http://localhost:8080/api/strategies` → 6 strateji.
`GET http://localhost:8080/api/events` → son N event (bkz `EVENTS_WINDOW`).
`GET http://localhost:8080/api/tokens` → görülen token'lar.
`GET http://localhost:8080/api/creators` → creator listesi (adres + `totalTokens`, bkz `CREATORS_LIST_LIMIT`).
`GET http://localhost:8080/api/creator/{address}` → creator profili (kimlik + `firstSeen` + token geçmişi;
bilinmeyen adres → 404).
`GET ws://localhost:8080/ws` → WebSocket; `events` topic'i tekil `FeedEvent` yayınlar, `tokens` topic'i tam
`TokenRow[]` snapshot yayınlar.

## Env değişkenleri
| Değişken | Zorunlu mu | Açıklama |
|---|---|---|
| `DATABASE_URL` | Hayır (yoksa in-memory fake store) | Postgres bağlantı string'i |
| `PORT` | Hayır (default 8080) | HTTP port |
| `CORS_ORIGIN` | Hayır | İzin verilen frontend origin |
| `HELIUS_API_KEY` | **Ingestion worker için evet** | Yoksa worker başlamaz — REST endpoint'ler (`/api/*`) yine çalışır, sadece boş/durağan DB ile |
| `EVENTS_WINDOW` | Hayır (default 200) | `/api/events` + `/api/tokens` snapshot'larının döndürdüğü satır sayısı |
| `GECKOTERMINAL_RATE_PER_MIN` | Hayır (default 25) | Paylaşılan GeckoTerminal istek bütçesi (istek/dk). Keşif + enrichment + token detail TEK token-bucket'tan çeker; keysiz free-tier (~30/dk) 429'unu önler |
| `GECKOTERMINAL_BURST` | Hayır (default 2) | Token-bucket burst kapasitesi (detail'in 2 çağrısı — header+OHLCV — birlikte geçebilsin) |
| `SAFETY_ENABLED` | Hayır (default true) | Token güvenliği arka plan scorer'ı (2a). Helius key yoksa başlamaz |
| `SAFETY_INTERVAL_SEC` | Hayır (default 60) | Skorlama döngüsü aralığı (saniye) |
| `SAFETY_LIMIT` | Hayır (default 40) | Döngü başına skorlanan token |
| `SAFETY_HOLDERS_CAP` | Hayır (default 5000) | Holder dağılımı için getTokenAccounts sayfalama tavanı |
| `CREATORS_LIST_LIMIT` | Hayır (default 100) | `/api/creators` listesinin döndürdüğü maksimum creator sayısı |
| `OUTCOME_ENABLED` | Hayır (default true) | Token outcome (sonuç) sınıflandırma arka plan worker'ı (2b-2a). Helius gerektirmez |
| `OUTCOME_INTERVAL_SEC` | Hayır (default 60) | Outcome sınıflandırma döngüsü aralığı (saniye) |
| `OUTCOME_LIMIT` | Hayır (default 60) | Döngü başına sınıflandırılan token |
| `OUTCOME_RUG_LIQ_RATIO` | Hayır (default 0.10) | Outcome sınıflandırma eşikleri, deploy'da kalibre edilebilir — rug likidite/tepe oranı |
| `OUTCOME_GRADUATION_MCAP` | Hayır (default 69000) | Outcome sınıflandırma eşikleri, deploy'da kalibre edilebilir — graduated marketCap eşiği (USD) |
| `OUTCOME_DUMPED_DRAWDOWN` | Hayır (default 80) | Outcome sınıflandırma eşikleri, deploy'da kalibre edilebilir — dumped max-drawdown yüzdesi |
| `OUTCOME_DEAD_VOL` | Hayır (default 100) | Outcome sınıflandırma eşikleri, deploy'da kalibre edilebilir — dead hacim tavanı (USD) |
| `OUTCOME_MIN_LIQ_FLOOR` | Hayır (default 500) | Outcome sınıflandırma eşikleri, deploy'da kalibre edilebilir — minimum likidite tabanı (USD) |
| `OUTCOME_DEAD_AGE_SEC` | Hayır (default 86400) | Outcome sınıflandırma eşikleri, deploy'da kalibre edilebilir — dead yaş eşiği (saniye) |
| `CREATORFILL_ENABLED` | Hayır (default true) | REST creator-backfill worker'ı (WS blokörü baypas). Helius key yoksa başlamaz |
| `CREATORFILL_INTERVAL_SEC` | Hayır (default 30) | Backfill döngüsü aralığı (saniye) |
| `CREATORFILL_LIMIT` | Hayır (default 20) | Döngü başına backfill denenen creator'sız pump.fun token |
| `CREATORFILL_MAX_SIG_PAGES` | Hayır (default 3) | Mint'in en-eski imzasını ararken `getSignaturesForAddress` sayfalama tavanı |

## Railway deploy (KULLANICI ADIMI)
1. Railway'de yeni servis → GitHub repo `furkanatesc/sentinel`, **Root Directory = `apps/api-go`** (nixpacks Go'yu otomatik derler; **Go 1.24** gerekli — `go.mod`'da pinli; start komutu binary'yi çalıştırır).
2. Servise **PostgreSQL** eklentisi ekle → `DATABASE_URL` otomatik enjekte olur.
3. Env: `CORS_ORIGIN=https://sentinel-brown-alpha.vercel.app`, `HELIUS_API_KEY=<Helius API key>` (ingestion worker'ı başlatmak için — bkz repo kökü `api_key_alinacakplatformlar.md`).
4. Deploy sonrası `<railway-url>/healthz`, `/api/strategies`, `/api/events`, `/api/tokens` ve `wss://<railway-url>/ws` doğrula.

## Frontend'i bağlama (KULLANICI ADIMI)
Vercel projesine env ekle: `NEXT_PUBLIC_API_BASE_URL=<railway-url>`, `NEXT_PUBLIC_DATA_SOURCE=http` → redeploy.
Strategies ekranı gerçek API'den, diğer ekranlar mock ile çalışır (hibrit).
