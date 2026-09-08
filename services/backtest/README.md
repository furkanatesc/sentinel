# SENTINEL Backtest Service (Python/FastAPI)

Backend Alt-proje 4 — **outcome-tabanlı** backtest motoru. Repo'nun ilk Python servisi.
Go API `/api/backtest`'i buraya proxy'ler; frontend kontratı değişmez.

## Ne yapar

`POST /backtest` (`BacktestParams`) → tarihsel token'ları `minCreatorScore`/`minTokenSafety` eşikleriyle
filtreler, her token'ın PnL'ini **saklı outcome + peak/drawdown**'dan modeller, `BacktestResult` agregeler.
Aynı Postgres'i (Go servisiyle) `DATABASE_URL` üzerinden okur.

## Yerel çalıştırma

```bash
cd services/backtest
python -m venv .venv && . .venv/Scripts/activate   # Windows; POSIX: . .venv/bin/activate
pip install -r requirements.txt
pytest                                              # saf birim testleri (DB gerekmez)
DATABASE_URL=postgres://... uvicorn app.main:app --reload
```

`pytest` DB'siz çalışır (simulate/aggregate saf; API testi repo'yu monkeypatch'ler).

## Railway deploy (kullanıcı admin adımı)

1. Railway'de yeni servis → **Root Directory = `services/backtest`** (Dockerfile otomatik algılanır).
2. Env: `DATABASE_URL` = Go servisiyle AYNI Postgres (paylaşımlı okuma).
3. Deploy sonrası servis URL'ini al, **Go servisine** `BACKTEST_SERVICE_URL` = o URL ver.
4. `BACKTEST_SERVICE_URL` set edilene kadar Go `/api/backtest` graceful `503` döner (frontend `notReady` dalı).

## Mimari (SRP)

- `app/models.py` — pydantic sözleşme (frontend types.ts ile birebir camelCase).
- `app/repo.py` — Postgres okuma (DB I/O yalnız burada).
- `app/simulate.py` — **saf** entry-filtre + outcome→PnL (I/O yok).
- `app/aggregate.py` — **saf** metrik + eğri agregasyonu.
- `app/main.py` — FastAPI kabuğu (repo → simulate → aggregate).
