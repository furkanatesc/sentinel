# SENTINEL API (Go)

Backend Alt-proje 0 (platform iskeleti) + Alt-proje 1 slice 1a (Solana ingestion + WS transport).
Endpoint'ler: `GET /api/strategies`, `GET /api/events`, `GET /api/tokens`, `GET /ws` (WebSocket).

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

## Railway deploy (KULLANICI ADIMI)
1. Railway'de yeni servis → GitHub repo `furkanatesc/sentinel`, **Root Directory = `apps/api-go`** (nixpacks Go'yu otomatik derler; **Go 1.24** gerekli — `go.mod`'da pinli; start komutu binary'yi çalıştırır).
2. Servise **PostgreSQL** eklentisi ekle → `DATABASE_URL` otomatik enjekte olur.
3. Env: `CORS_ORIGIN=https://sentinel-brown-alpha.vercel.app`, `HELIUS_API_KEY=<Helius API key>` (ingestion worker'ı başlatmak için — bkz repo kökü `api_key_alinacakplatformlar.md`).
4. Deploy sonrası `<railway-url>/healthz`, `/api/strategies`, `/api/events`, `/api/tokens` ve `wss://<railway-url>/ws` doğrula.

## Frontend'i bağlama (KULLANICI ADIMI)
Vercel projesine env ekle: `NEXT_PUBLIC_API_BASE_URL=<railway-url>`, `NEXT_PUBLIC_DATA_SOURCE=http` → redeploy.
Strategies ekranı gerçek API'den, diğer ekranlar mock ile çalışır (hibrit).
