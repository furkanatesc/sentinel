from datetime import datetime, timezone

from app.aggregate import aggregate
from app.models import BacktestParams
from app.simulate import SimTrade


def _params(**kw):
    base = dict(
        strategyId="s1", rangePreset="30d", initialCapitalSol=100.0, maxPositions=4,
        slippageModel="realistic", priorityFee=0.0, latencyModel="realistic",
        liquidityModel="realistic", minCreatorScore=50.0, minTokenSafety=50.0,
    )
    base.update(kw)
    return BacktestParams(**base)


def _ts(y, m, d):
    return int(datetime(y, m, d, tzinfo=timezone.utc).timestamp())


def _fixture():
    return [
        SimTrade(time=_ts(2026, 1, 15), price=1.0, pnlSol=10.0, size=25.0, outcome="graduated", safetyScore=80.0, holdingHours=48.0),
        SimTrade(time=_ts(2026, 1, 20), price=1.0, pnlSol=-5.0, size=25.0, outcome="dumped", safetyScore=60.0, holdingHours=6.0),
        SimTrade(time=_ts(2026, 2, 5), price=1.0, pnlSol=-20.0, size=25.0, outcome="rug", safetyScore=55.0, holdingHours=1.0),
        SimTrade(time=_ts(2026, 2, 10), price=1.0, pnlSol=15.0, size=25.0, outcome="active", safetyScore=90.0, holdingHours=24.0),
    ]


def test_metrics_deterministic():
    r = aggregate(_fixture(), _params())
    m = r.metrics
    assert m.trades == 4
    assert m.winRatePct == 50.0            # 2/4
    assert m.netPnlSol == 0.0              # 10-5-20+15
    assert m.rugExposurePct == 25.0        # 1/4 rugged
    assert m.profitFactor == 1.0           # gross 25 / gross 25
    assert m.avgTradeSol == 0.0            # 0/4
    assert abs(m.avgHoldingHours - 19.75) < 1e-9  # (48+6+1+24)/4
    assert 22.0 < m.maxDrawdownPct < 23.0  # 110→85 tepe-altı ≈ %22.73


def test_curves_and_buckets():
    r = aggregate(_fixture(), _params())
    assert len(r.equityCurve) == 4
    assert r.equityCurve[-1].v == 100.0     # 100+10-5-20+15
    assert len(r.drawdown) == 4
    assert len(r.monthlyReturns) == 2       # Ocak + Şubat
    assert len(r.trades) == 4
    assert sum(b.count for b in r.tradeDistribution) == 4
    hi = next(s for s in r.pnlByScore if s.scoreBucket == "75-100")
    assert hi.pnlSol == 25.0                # t1(+10)+t4(+15)


def test_empty_trades_valid_result():
    r = aggregate([], _params())
    assert r.metrics.trades == 0
    assert r.metrics.netPnlSol == 0.0
    assert r.metrics.winRatePct == 0.0
    assert r.equityCurve == []
    assert r.trades == []
