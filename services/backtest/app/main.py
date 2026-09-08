"""FastAPI kabuğu — repo → simulate → aggregate akışını bağlar (ince I/O katmanı).

DB bağlantısı (psycopg) `fetch_tokens` içinde LAZY import edilir → saf modüller ve testler
psycopg gerektirmez (test `fetch_tokens`'ı monkeypatch'ler).
"""
from __future__ import annotations

import os
import time

from fastapi import FastAPI

from app import repo
from app.aggregate import aggregate
from app.models import BacktestParams, BacktestResult
from app.simulate import run

app = FastAPI(title="SENTINEL Backtest Service")

# rangePreset → geriye dönük gün sayısı ("all"/bilinmeyen → 0 = tüm geçmiş).
_RANGE_DAYS = {"7d": 7, "30d": 30, "90d": 90, "180d": 180, "1y": 365}


def _now() -> int:
    return int(time.time())


def _range_start_ts(preset: str, now: int) -> int:
    days = _RANGE_DAYS.get(preset, 0)
    return 0 if days == 0 else now - days * 86400


def fetch_tokens(range_start_ts: int) -> list[repo.TokenRow]:
    """Postgres'ten token'ları yükler (lazy psycopg). Testte monkeypatch edilir."""
    import psycopg  # lazy: saf modüller/testler psycopg'siz çalışsın

    dsn = os.environ["DATABASE_URL"]
    with psycopg.connect(dsn) as conn:
        return repo.load_tokens(conn, range_start_ts)


@app.get("/healthz")
def healthz() -> dict:
    return {"ok": True}


@app.post("/backtest", response_model=BacktestResult)
def backtest(params: BacktestParams) -> BacktestResult:
    start = _range_start_ts(params.rangePreset, _now())
    tokens = fetch_tokens(start)
    trades = run(tokens, params)
    return aggregate(trades, params)
