# SENTINEL Backend — Backtest Motoru (Python/FastAPI) Tasarım Spec'i

**Tarih:** 2026-09-08
**Dilim:** Backend Alt-proje 4 — Backtest motoru. Repo'nun **ilk Python servisi**.
**Durum:** Spec — implementasyon öncesi (kullanıcı onayladı, "devam durma").

## Amaç

`POST /api/backtest`'i gerçeğe döndürmek — Backtesting ekranını canlıya almak. **Outcome-tabanlı**
(kullanıcı kararı 2026-09-07): tarihsel token'ları `BacktestParams` eşikleriyle filtreler, her token'ın
PnL'ini **saklı outcome + peak/drawdown**'dan modeller. **Entegrasyon-gerektirmez** (yalnız mevcut Postgres);
harici veri-sağlayıcı yok.

## Kapsam

**Dahil:**
- **`services/backtest`** — Python FastAPI servisi (AYNI Postgres'i okur). `POST /backtest` → `BacktestResult`.
- Go API `/api/backtest` → Python'a **proxy** (`BACKTEST_SERVICE_URL`; unset → `notReady` graceful).
- Frontend `runBacktest` `LIVE_ENDPOINTS`'e eklenir (UI dokunulmaz).

**Kapsam dışı (bilinçli):**
- Tick-tick replay (per-token tarihsel metrik/fiyat serisi yok → gelecekte veri-toplama gerekir).
- Tam strateji entry/exit koşulları (backend'de yok → `minCreatorScore`/`minTokenSafety` eşik-filtresi).
- `priceSeries` gerçek global fiyat yolu (outcome-tabanlıda yok → temsili/boş; followup).
- Gerçek deploy: Railway 2. servis kurulumu (kullanıcı admin adımı).

## Mimari

**Topoloji (kullanıcı onaylı):** ayrı FastAPI servisi + Go proxy → frontend kontratı değişmez.

```
Frontend runBacktest ──HTTP──▶ Go /api/backtest ──proxy HTTP──▶ Python /backtest ──SQL──▶ Postgres
```

### Python servisi (`services/backtest/`)

Dizin (SRP modüller, saf çekirdek + ince I/O kabuğu):
- `app/models.py` — pydantic `BacktestParams` / `BacktestResult` (+ alt tipler), frontend `types.ts` ile birebir JSON.
- `app/repo.py` — Postgres okuma (`psycopg`): `load_tokens(conn, range_start_ts) -> list[TokenRow]` (tokens⋈outcome⋈creators).
- `app/simulate.py` — **saf** `simulate_trade(token, params) -> Trade | None` (entry filtresi + outcome→PnL) + `run(tokens, params) -> list[Trade]`.
- `app/aggregate.py` — **saf** `aggregate(trades, params) -> BacktestResult` (metrikler + eğriler).
- `app/main.py` — FastAPI app; `POST /backtest` → repo.load → simulate.run → aggregate; `GET /healthz`.
- `tests/` — pytest (saf modül birim testleri).
- `requirements.txt`, `Dockerfile` (veya Railway nixpacks), `README.md`.

### Entry filtresi + PnL modeli (saf, açıklanabilir, deterministik)

`simulate_trade(token, params)`:
1. **Entry:** `token.creator_score >= params.minCreatorScore AND token.safety_score >= params.minTokenSafety`.
   Geçmezse `None` (işlem yok).
2. **Pozisyon büyüklüğü:** `size = initialCapitalSol / maxPositions` (SOL bazlı; basit eşit-ağırlık).
3. **Getiri (outcome→retMultiplier):**
   - `graduated` → `+GRAD_RET` (varsayılan +2.0 = %200) veya peak-tabanlı: `min(peak/entry - 1, GRAD_CAP)`.
   - `active` → `momentum/peak`-tabanlı ılımlı getiri (varsayılan +0.2).
   - `dumped` → `-(max_drawdown_pct/100)`.
   - `rugged` → `-RUG_LOSS` (varsayılan -0.95; likidite kurtarma payı sonra).
   - `dead` → `-DEAD_LOSS` (varsayılan -0.5).
   - bilinmeyen/`""` → işlem yok (skorlanmamış).
4. **Maliyet:** `pnl = size * retMultiplier - slippageCost - priorityFee`; `slippageCost = size * SLIPPAGE_BPS(slippageModel)`.
5. **Trade:** `{time: first_seen_ts, price: entry_price, side: "buy", pnlSol: pnl}` (+ dahili: outcome, scoreBucket, holdingHours).

Sabitler `simulate.py`'de adlandırılmış (kalibrasyon-env değil, kod sabiti; deploy'da gerçek dağılımla ayarlanabilir → followup).

### Agregasyon (`aggregate.py`)

- **metrics:** winRate=(pnl>0)/n; profitFactor=Σkâr/|Σzarar|; sharpe/sortino=mean(ret)/std(ret) (getiri = pnl/size); maxDrawdownPct=equity eğrisinden; avgTradeSol; rugExposurePct=(outcome=rugged)/n; trades=n; avgHoldingHours.
- **equityCurve:** first_seen'e göre sıralı kümülatif PnL (`initialCapitalSol` + Σ).
- **drawdown:** equity tepe-altı yüzde serisi.
- **monthlyReturns:** first_seen ayına göre gruplu Σpnl.
- **tradeDistribution:** PnL kovaları (ör. <-1, -1..0, 0..1, >1).
- **pnlByScore:** safety skor kovalarına (0-25/25-50/…) göre Σpnl.
- **priceSeries:** temsili (boş `[]` ya da equity yansıması) — dürüst kalıntı.

### Go proxy (`internal/api/backtest.go`)

`backtestHandler(serviceURL string) http.HandlerFunc`: `serviceURL==""` → 503 `notReady` (graceful). Aksi halde
istek gövdesini `POST {serviceURL}/backtest`'e iletir, cevabı aynen döndürür (timeout'lu). Config `BacktestServiceURL`
(`BACKTEST_SERVICE_URL`). Router wiring + `RouterDeps.BacktestServiceURL`.

### Frontend

`lib/api/live-endpoints.ts`'e `"runBacktest"` eklenir → httpApi `runBacktest` POST `/api/backtest` (mevcut httpApi'de
`notReady`; gerçek POST'a çevrilir). `NEXT_PUBLIC_DATA_SOURCE=http` modunda gerçek; mock modda mock. UI değişmez.

## Veri akışı

Frontend form submit → `runBacktest(params)` → Go `/api/backtest` → Python `/backtest` → Postgres oku → filtrele +
simüle + agregele → `BacktestResult` → Go geri döner → React Query render. Deterministik (aynı params+veri → aynı sonuç).

## Hata yönetimi

- Go proxy: `BACKTEST_SERVICE_URL` yok → 503 (frontend `notReady` gibi ele alır); Python 5xx/timeout → 502.
- Python: DB hatası → 500 + log; boş token seti → boş ama geçerli `BacktestResult` (0 trade, boş eğriler).
- Frontend: mevcut `BacktestContent` loading/error dalları (zaten var).

## Test

- **pytest** (Python): `simulate_trade` her outcome dalı + entry-filtre reddi + maliyet; `run` çoklu token; `aggregate`
  sabit trade fixture'larından deterministik metrikler (winRate/profitFactor/maxDD/monthlyReturns/pnlByScore); boş-set.
- **Go:** `backtestHandler` — serviceURL boş → 503; fake upstream (httptest) → gövde proxy + cevap passthrough.
- **Frontend:** seam flip opsiyonel (LIVE_ENDPOINTS'e ekleme + httpApi POST şekli).
- CI: Python testleri `services/backtest` içinde `pytest`; Go `go test ./...`.

## Deploy sonrası (kullanıcı admin adımı)

Railway'de `services/backtest` için 2. servis (Python, root=`services/backtest`, `DATABASE_URL` paylaşımlı);
Go servisine `BACKTEST_SERVICE_URL`=Python servis URL'i. Bunlar KOD'da değil Railway panelinde — kod hazır olunca
kullanıcı kurar; kurulana dek `/api/backtest` graceful `notReady` döner (frontend zaten bu dalı ele alıyor).

## Followup

- PnL model sabitlerini kalibrasyon-env'e taşı; likidite kurtarma payı (rug); `priceSeries` temsili → gerçek.
- Tick-replay için per-token tarihsel metrik serisi toplama (ayrı büyük dilim).
- Strateji entry/exit koşullarını backend'e taşıyıp eşik-filtresi yerine tam koşul (ayrı dilim).
