# SENTINEL API (Go)

Backend Alt-proje 0 — platform iskeleti. Şimdilik tek gerçek endpoint: `GET /api/strategies`.

## Yerel çalıştırma
```bash
cd apps/api-go
go run ./cmd/server           # DATABASE_URL yoksa in-memory fake store
# Postgres ile:
DATABASE_URL=postgres://user:pass@localhost:5432/sentinel PORT=8080 \
  CORS_ORIGIN=http://localhost:3000 go run ./cmd/server
```
`GET http://localhost:8080/healthz` → `{"status":"ok"}`
`GET http://localhost:8080/api/strategies` → 6 strateji.

## Railway deploy (KULLANICI ADIMI)
1. Railway'de yeni servis → GitHub repo `furkanatesc/sentinel`, **Root Directory = `apps/api-go`** (nixpacks Go'yu otomatik derler; start komutu binary'yi çalıştırır).
2. Servise **PostgreSQL** eklentisi ekle → `DATABASE_URL` otomatik enjekte olur.
3. Env: `CORS_ORIGIN=https://sentinel-brown-alpha.vercel.app`.
4. Deploy sonrası `<railway-url>/healthz` ve `/api/strategies` doğrula.

## Frontend'i bağlama (KULLANICI ADIMI)
Vercel projesine env ekle: `NEXT_PUBLIC_API_BASE_URL=<railway-url>`, `NEXT_PUBLIC_DATA_SOURCE=http` → redeploy.
Strategies ekranı gerçek API'den, diğer ekranlar mock ile çalışır (hibrit).
