from app.models import BacktestParams
from app.repo import TokenRow
from app.simulate import run, simulate_trade


def _params(**kw):
    base = dict(
        strategyId="s1", rangePreset="30d", initialCapitalSol=100.0, maxPositions=10,
        slippageModel="realistic", priorityFee=0.0, latencyModel="realistic",
        liquidityModel="realistic", minCreatorScore=50.0, minTokenSafety=50.0,
    )
    base.update(kw)
    return BacktestParams(**base)


def _tok(outcome="active", creator=80.0, safety=80.0, **kw):
    base = dict(
        mint="M", firstSeenTs=1000, creatorScore=creator, safetyScore=safety,
        outcome=outcome, peakMarketCap=0.0, maxDrawdownPct=0.0, liquidity=1000.0,
        price=1.0, marketCap=0.0,
    )
    base.update(kw)
    return TokenRow(**base)


def test_entry_filter_rejects_low_score():
    p = _params()
    assert simulate_trade(_tok(creator=10), p) is None
    assert simulate_trade(_tok(safety=10), p) is None


def test_graduated_is_profitable():
    t = simulate_trade(_tok(outcome="graduated"), _params(priorityFee=0.0))
    assert t is not None
    assert t.pnlSol > 0
    assert t.size == 10.0  # 100 / maxPositions(10)


def test_rugged_is_large_loss():
    t = simulate_trade(_tok(outcome="rugged"), _params(slippageModel="optimistic", priorityFee=0.0))
    assert t is not None
    # ~ -0.95 * size (küçük slippage düşülür), belirgin negatif
    assert -10.0 < t.pnlSol < -8.0


def test_dumped_uses_drawdown():
    t = simulate_trade(_tok(outcome="dumped", maxDrawdownPct=40.0), _params(slippageModel="optimistic", priorityFee=0.0))
    assert t is not None
    assert t.pnlSol < 0  # -40% * size - maliyet


def test_unknown_outcome_no_trade():
    assert simulate_trade(_tok(outcome=""), _params()) is None


def test_costs_reduce_pnl():
    hi = simulate_trade(_tok(outcome="graduated"), _params(slippageModel="optimistic", priorityFee=0.0))
    lo = simulate_trade(_tok(outcome="graduated"), _params(slippageModel="pessimistic", priorityFee=1.0))
    assert lo.pnlSol < hi.pnlSol  # daha çok slippage + priorityFee → daha az pnl


def test_run_filters_and_sizes():
    p = _params(maxPositions=4, initialCapitalSol=100.0)
    toks = [
        _tok(mint="A", outcome="graduated"),
        _tok(mint="B", outcome="rugged"),
        _tok(mint="C", creator=10),  # elenir
        _tok(mint="D", outcome=""),  # skorsuz → işlem yok
    ]
    trades = run(toks, p)
    assert len(trades) == 2  # A + B
    assert all(t.size == 25.0 for t in trades)  # 100/4
