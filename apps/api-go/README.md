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
| `SOLANA_RPC_URL` | Hayır | Creator-backfill resolver'ı için alternatif genel RPC (ör. QuickNode/Alchemy/Chainstack free). Helius free-tier `getSignaturesForAddress`'i bloke ettiğinden gerekir. Set edilirse resolver bunu kullanır (standart getSignaturesForAddress+getTransaction); boşsa Helius rpcURL'e düşer. WS + DAS (holders/safety) yine Helius'ta kalır |
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
| `CREATORFILL_RATE_PER_MIN` | Hayır (default 120) | Helius RPC paylaşılan istek bütçesi (istek/dk); free-tier 429 burst'ünü önler. Per-key limit → client-side kısıtlama etkili |
| `CREATORFILL_BURST` | Hayır (default 2) | Rate-limiter token-bucket burst kapasitesi |
| `REPUTATION_ENABLED` | Hayır (default true) | Creator itibar scorer'ı (2b-2b). Saf DB, RPC gerektirmez |
| `REPUTATION_INTERVAL_SEC` | Hayır (default 60) | Skorlama döngüsü aralığı (saniye) |
| `REPUTATION_LIMIT` | Hayır (default 60) | Döngü başına skorlanan creator |
| `REPUTATION_MIN_RESOLVED` | Hayır (default 5) | Tam güven için gereken çözülmüş token (confidence K) |
| `REPUTATION_W_RUG` | Hayır (default 50) | Rug oranı ceza ağırlığı |
| `REPUTATION_W_FAIL` | Hayır (default 20) | Dump/dead oranı ceza ağırlığı |
| `REPUTATION_W_GRAD` | Hayır (default 40) | Graduated oranı ödül ağırlığı |
| `REPUTATION_HIGH_DRAWDOWN` | Hayır (default 80) | per-token "yüksek düşüş" bayrağı eşiği (%) |
| `MANIPULATION_ENABLED` | Hayır (default true) | Manipülasyon riski scorer'ı (2c). Saf DB, RPC gerektirmez |
| `MANIPULATION_INTERVAL_SEC` | Hayır (default 60) | Skorlama döngüsü aralığı (saniye) |
| `MANIPULATION_LIMIT` | Hayır (default 60) | Döngü başına skorlanan token |
| `MANIPULATION_MIN_TXNS` | Hayır (default 20) | Skor üretmek için gereken minimum h24 işlem sayısı (altında skor=0/conf=0) |
| `MANIPULATION_CONF_TXNS` | Hayır (default 100) | Tam güven için gereken h24 işlem sayısı (confidence tavanı) |
| `MANIPULATION_W_IMBALANCE` | Hayır (default 30) | Alım/satım dengesizliği ceza ağırlığı |
| `MANIPULATION_W_WASH` | Hayır (default 35) | Wash-trading proxy (işlem/alıcı oranı) ceza ağırlığı |
| `MANIPULATION_W_VOLUME` | Hayır (default 25) | Hacim/likidite oranı ceza ağırlığı |
| `MANIPULATION_W_CREATOR` | Hayır (default 10) | Creator holding oranı ceza ağırlığı |
| `MANIPULATION_WASH_MIN` | Hayır (default 3) | Wash-proxy band alt sınırı (bu değerin altı ceza almaz; `getenvFloat >0` şartıyla 0 girilemez — YAGNI sınırı) |
| `MANIPULATION_WASH_MAX` | Hayır (default 15) | Wash-proxy band üst sınırı (bu değerin üstü tam ceza alır) |
| `MANIPULATION_VOL_MIN` | Hayır (default 3) | Hacim/likidite band alt sınırı |
| `MANIPULATION_VOL_MAX` | Hayır (default 20) | Hacim/likidite band üst sınırı |

## Manipülasyon riski (2c)
Kural-tabanlı, saf-DB agrega-proxy scorer — RPC yok, per-wallet/trade-flow analiz yok. Girdi: GeckoTerminal
`transactions.h24` (buys/sells/buyers) + safety `creator_holding` + market hacim/likidite; `manipulation.Scorer`
bu agregaları eşiklere (`MANIPULATION_*`) karşı puanlayıp inverted (yüksek risk = düşük güven token) bir skor +
breakdown üretir. `MANIPULATION_MIN_TXNS` altındaki token'larda skor=0/confidence=0 döner (düşük-aktivite için
dürüst "veri yok", "risksiz" değil). Per-wallet sniper/bot-activity/creator-sell tespiti (trade-flow gerektirir)
kasıtlı olarak 2e'ye ertelendi — bu slice yalnız agrega-proxy sinyaller kullanır.

## Railway deploy (KULLANICI ADIMI)
1. Railway'de yeni servis → GitHub repo `furkanatesc/sentinel`, **Root Directory = `apps/api-go`** (nixpacks Go'yu otomatik derler; **Go 1.24** gerekli — `go.mod`'da pinli; start komutu binary'yi çalıştırır).
2. Servise **PostgreSQL** eklentisi ekle → `DATABASE_URL` otomatik enjekte olur.
3. Env: `CORS_ORIGIN=https://sentinel-brown-alpha.vercel.app`, `HELIUS_API_KEY=<Helius API key>` (ingestion worker'ı başlatmak için — bkz repo kökü `api_key_alinacakplatformlar.md`).
4. Deploy sonrası `<railway-url>/healthz`, `/api/strategies`, `/api/events`, `/api/tokens` ve `wss://<railway-url>/ws` doğrula.

## Frontend'i bağlama (KULLANICI ADIMI)
Vercel projesine env ekle: `NEXT_PUBLIC_API_BASE_URL=<railway-url>`, `NEXT_PUBLIC_DATA_SOURCE=http` → redeploy.
Strategies ekranı gerçek API'den, diğer ekranlar mock ile çalışır (hibrit).
