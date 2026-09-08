from fastapi.testclient import TestClient

from app import main
from app.repo import TokenRow

client = TestClient(main.app)

_PARAMS = {
    "strategyId": "s1", "rangePreset": "30d", "initialCapitalSol": 100.0, "maxPositions": 4,
    "slippageModel": "realistic", "priorityFee": 0.0, "latencyModel": "realistic",
    "liquidityModel": "realistic", "minCreatorScore": 50.0, "minTokenSafety": 50.0,
}


def _tok(mint, outcome, creator=80.0, safety=80.0):
    return TokenRow(
        mint=mint, firstSeenTs=1000, creatorScore=creator, safetyScore=safety,
        outcome=outcome, peakMarketCap=0.0, maxDrawdownPct=0.0, liquidity=1000.0, price=1.0, marketCap=0.0,
    )


def test_healthz():
    r = client.get("/healthz")
    assert r.status_code == 200
    assert r.json()["ok"] is True


def test_backtest_shape(monkeypatch):
    monkeypatch.setattr(main, "fetch_tokens", lambda _start: [
        _tok("A", "graduated"), _tok("B", "rugged"), _tok("C", "active", creator=10),
    ])
    r = client.post("/backtest", json=_PARAMS)
    assert r.status_code == 200
    body = r.json()
    # sözleşme şekli (frontend BacktestResult)
    for key in ("metrics", "equityCurve", "drawdown", "monthlyReturns", "tradeDistribution", "pnlByScore", "priceSeries", "trades"):
        assert key in body
    assert body["metrics"]["trades"] == 2  # A + B (C elenir)
    assert isinstance(body["equityCurve"], list)
    assert body["trades"][0]["side"] == "buy"


def test_backtest_empty(monkeypatch):
    monkeypatch.setattr(main, "fetch_tokens", lambda _start: [])
    r = client.post("/backtest", json=_PARAMS)
    assert r.status_code == 200
    assert r.json()["metrics"]["trades"] == 0
