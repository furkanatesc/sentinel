# Backtest Motoru (Python/FastAPI) Implementation Plan

> **For agentic workers:** subagent-driven-development veya executing-plans. Steps checkbox.

**Goal:** `/api/backtest`'i gerçeğe döndür — Python FastAPI servisi (outcome-tabanlı) + Go proxy + frontend seam.

**Architecture:** Ayrı `services/backtest` (FastAPI, Postgres okur) + Go `/api/backtest` proxy + frontend LIVE_ENDPOINTS. Saf `simulate`/`aggregate` çekirdek + ince I/O kabuğu.

**Tech Stack:** Python 3.11+ (FastAPI, pydantic v2, psycopg3), Go (chi), Next.js.

**Spec:** `docs/superpowers/specs/2026-09-08-sentinel-backtest-engine-design.md`

## Global Constraints
- Clean code & SOLID: saf `simulate_trade`/`aggregate` (I/O yok), ince `repo`/`main` kabuğu; SRP dosyalar.
- Deterministik (aynı params+veri → aynı sonuç). Frontend kontratı (BacktestResult JSON) DEĞİŞMEZ.
- Go proxy graceful: `BACKTEST_SERVICE_URL` yok → 503 (frontend notReady dalı).
- Python testleri `pytest` yeşil; Go `go test ./...` yeşil; frontend seam kırılmaz.
- Commit sonu: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`

---

## Task 1: Python servis iskelesi + models + repo

**Files:** `services/backtest/`: `requirements.txt`, `app/__init__.py`, `app/models.py`, `app/repo.py`, `Dockerfile`, `README.md`, `tests/__init__.py`.

- [ ] Step 1: `requirements.txt` (fastapi, uvicorn[standard], pydantic>=2, psycopg[binary]>=3, pytest, httpx).
- [ ] Step 2: `app/models.py` — pydantic: `BacktestParams` (strategyId, rangePreset, initialCapitalSol, maxPositions, slippageModel, priorityFee, latencyModel, liquidityModel, minCreatorScore, minTokenSafety); `EquityPoint{t,v}`, `DrawdownPoint{t,v}`, `MonthlyReturn{label,pct}`, `DistributionBucket{label,count}`, `ScorePnl{scoreBucket,pnlSol}`, `BacktestTrade{time,price,side,pnlSol}`, `BacktestMetrics{...10 alan}`, `BacktestResult{metrics,equityCurve,drawdown,monthlyReturns,tradeDistribution,pnlByScore,priceSeries,trades}`. JSON alan adları frontend types.ts ile birebir (camelCase — pydantic `alias`/`populate_by_name` ya da doğrudan camelCase alanlar).
- [ ] Step 3: `app/repo.py` — `TokenRow` dataclass (mint, firstSeenTs, creatorScore, safetyScore, outcome, peakMarketCap, maxDrawdownPct, liquidity, price, marketCap). `load_tokens(conn, range_start_ts) -> list[TokenRow]` (SELECT tokens LEFT JOIN creators ... WHERE first_seen_ts>=range_start_ts). DB I/O yalnız burada.
- [ ] Step 4: `Dockerfile` (python:3.11-slim, pip install, uvicorn app.main:app) + `README.md` (Railway: root=services/backtest, DATABASE_URL, PORT).
- [ ] Step 5: Commit.

## Task 2: Saf `simulate` + test

**Files:** `services/backtest/app/simulate.py`, `tests/test_simulate.py`.

- [ ] Step 1: Failing pytest — `simulate_trade(token, params)`: entry-filtre reddi (skor düşük → None); graduated → pozitif pnl; rugged → ~-0.95*size; dumped → -maxDD; dead → negatif; slippage+priorityFee düşülür; bilinmeyen outcome → None. `run(tokens, params)` maxPositions cap + eşit sizing.
- [ ] Step 2: Run → fail (`cd services/backtest && pytest tests/test_simulate.py`).
- [ ] Step 3: Implement `simulate.py` (spec §PnL modeli; adlandırılmış sabitler; saf, I/O yok).
- [ ] Step 4: Run → pass.
- [ ] Step 5: Commit.

## Task 3: Saf `aggregate` + test

**Files:** `services/backtest/app/aggregate.py`, `tests/test_aggregate.py`.

- [ ] Step 1: Failing pytest — sabit trade listesi → beklenen metrics (winRate/profitFactor/avgTrade/rugExposure/trades), equityCurve kümülatif, drawdown, monthlyReturns (ay grupları), tradeDistribution kovaları, pnlByScore; boş liste → sıfır/boş geçerli sonuç.
- [ ] Step 2: Run → fail.
- [ ] Step 3: Implement `aggregate.py` (saf; sharpe/sortino getiri=pnl/size; maxDD equity'den).
- [ ] Step 4: Run → pass.
- [ ] Step 5: Commit.

## Task 4: FastAPI `main` + endpoint wiring + test

**Files:** `services/backtest/app/main.py`, `tests/test_api.py`.

- [ ] Step 1: Failing pytest (FastAPI TestClient) — `POST /backtest` sahte repo (monkeypatch `load_tokens`) ile → 200 + `BacktestResult` şekli; `GET /healthz` → 200.
- [ ] Step 2: Run → fail.
- [ ] Step 3: Implement `main.py`: FastAPI app; `/healthz`; `POST /backtest` → `range_start_ts` (rangePreset'ten) → `load_tokens` → `simulate.run` → `aggregate` → `BacktestResult`. DB bağlantısı env `DATABASE_URL` (psycopg). repo çağrısı test'te monkeypatch edilebilir (modül-düzey fonksiyon).
- [ ] Step 4: Run → pass.
- [ ] Step 5: Commit.

## Task 5: Go proxy + config + router + test

**Files:** Modify `internal/config/config.go`, `internal/api/router.go`, `cmd/server/main.go`; Create `internal/api/backtest.go`, `internal/api/backtest_test.go`.

- [ ] Step 1: Failing Go test — `backtestHandler("")` → 503; fake upstream httptest server → gövde iletildi + cevap passthrough + status 200.
- [ ] Step 2: Run → fail.
- [ ] Step 3: Implement: config `BacktestServiceURL` (`BACKTEST_SERVICE_URL`, default ""); `backtest.go` `backtestHandler(serviceURL string, timeout time.Duration)`: boş → `writeJSON(503, {"error":"backtest service not configured"})`; aksi `http.Post(serviceURL+"/backtest", body)` → cevabı passthrough (status+gövde). router `r.Post("/api/backtest", backtestHandler(d.BacktestServiceURL, ...))` + `RouterDeps.BacktestServiceURL`; main.go `BacktestServiceURL: cfg.BacktestServiceURL`.
- [ ] Step 4: Run → pass + `go build ./... && go test ./... -race && go vet ./...`.
- [ ] Step 5: Commit.

## Task 6: Frontend seam (LIVE_ENDPOINTS + httpApi)

**Files:** Modify `apps/web/lib/api/live-endpoints.ts`, `apps/web/lib/api/http.ts`.

- [ ] Step 1: `live-endpoints.ts`'e `"runBacktest"` ekle.
- [ ] Step 2: `http.ts` `runBacktest` `notReady` → `postJson<BacktestResult>("/api/backtest", params)` (getJson deseni; POST helper yoksa ekle).
- [ ] Step 3: `npx tsc --noEmit` (non-test temiz) + `npx vitest run lib/api/` + `npm run build` yeşil.
- [ ] Step 4: Commit.

## Task 7: Review + doküman + Railway yönergesi

- [ ] Step 1: pytest + `go test ./... -race` + frontend build yeşil.
- [ ] Step 2: Whole-branch review (opus) → receiving-code-review ile ele al.
- [ ] Step 3: Yaşayan dokümanlar — progress.md + MEMORY.md + followups (PnL kalibrasyon, priceSeries, tick-replay).
- [ ] Step 4: Railway kurulum yönergesi kullanıcıya (2. servis + BACKTEST_SERVICE_URL) — DUR-noktası (kullanıcı admin).
- [ ] Step 5: Merge/push.

---

## Self-Review
**Spec coverage:** models/repo→T1; simulate→T2; aggregate→T3; api→T4; Go proxy→T5; frontend→T6; review/deploy→T7 ✅.
**Type consistency:** `TokenRow` T1→T2/T4; `BacktestParams/Result` T1(py)↔frontend types.ts; `simulate_trade`/`run` T2→T4; `aggregate` T3→T4; `BacktestServiceURL` T5 config↔router↔main; `backtestHandler` T5. ✅
