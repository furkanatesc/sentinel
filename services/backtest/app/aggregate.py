"""Saf agregasyon — SimTrade listesinden BacktestResult (I/O YOK, deterministik)."""
from __future__ import annotations

from datetime import datetime, timezone

from app.models import (
    BacktestMetrics,
    BacktestParams,
    BacktestResult,
    BacktestTrade,
    DistributionBucket,
    DrawdownPoint,
    EquityPoint,
    MonthlyReturn,
    ScorePnl,
)
from app.simulate import SimTrade

# profitFactor'da zarar yoksa "sonsuz" yerine sonlu sentinel (JSON inf serileştiremez).
_PF_NO_LOSS = 99.0

_DIST_LABELS = ["<-5", "-5..0", "0..5", ">5"]
_SCORE_BUCKETS = [(0, 25, "0-25"), (25, 50, "25-50"), (50, 75, "50-75"), (75, 101, "75-100")]


def _std(xs: list[float]) -> float:
    if len(xs) < 2:
        return 0.0
    mean = sum(xs) / len(xs)
    var = sum((x - mean) ** 2 for x in xs) / len(xs)
    return var ** 0.5


def _empty_metrics() -> BacktestMetrics:
    return BacktestMetrics(
        netPnlSol=0.0, winRatePct=0.0, profitFactor=0.0, sharpe=0.0, sortino=0.0,
        maxDrawdownPct=0.0, avgTradeSol=0.0, rugExposurePct=0.0, trades=0, avgHoldingHours=0.0,
    )


def _month_label(ts: int) -> str:
    return datetime.fromtimestamp(ts, tz=timezone.utc).strftime("%Y-%m")


def _dist_index(pnl: float) -> int:
    if pnl < -5:
        return 0
    if pnl < 0:
        return 1
    if pnl <= 5:
        return 2
    return 3


def aggregate(trades: list[SimTrade], params: BacktestParams) -> BacktestResult:
    if not trades:
        return BacktestResult(
            metrics=_empty_metrics(), equityCurve=[], drawdown=[], monthlyReturns=[],
            tradeDistribution=[], pnlByScore=[], priceSeries=[], trades=[],
        )

    n = len(trades)
    pnls = [t.pnlSol for t in trades]
    net = sum(pnls)
    wins = [p for p in pnls if p > 0]
    losses = [p for p in pnls if p < 0]
    gross_profit = sum(wins)
    gross_loss = -sum(losses)  # pozitif
    returns = [t.pnlSol / t.size if t.size > 0 else 0.0 for t in trades]
    downside = [r for r in returns if r < 0]
    mean_ret = sum(returns) / n

    # equity + drawdown
    equity: list[EquityPoint] = []
    drawdown: list[DrawdownPoint] = []
    running = params.initialCapitalSol
    peak = running
    max_dd_pct = 0.0
    for t in trades:
        running += t.pnlSol
        equity.append(EquityPoint(t=t.time, v=round(running, 6)))
        if running > peak:
            peak = running
        dd = (running - peak) / peak * 100.0 if peak > 0 else 0.0
        drawdown.append(DrawdownPoint(t=t.time, v=round(dd, 6)))
        if -dd > max_dd_pct:
            max_dd_pct = -dd

    # monthly (label sırası ilk-görülme sırasına göre)
    monthly: dict[str, float] = {}
    order: list[str] = []
    for t in trades:
        lbl = _month_label(t.time)
        if lbl not in monthly:
            monthly[lbl] = 0.0
            order.append(lbl)
        monthly[lbl] += t.pnlSol
    cap = params.initialCapitalSol if params.initialCapitalSol > 0 else 1.0
    monthly_returns = [MonthlyReturn(label=l, pct=round(monthly[l] / cap * 100.0, 6)) for l in order]

    # distribution
    dist_counts = [0, 0, 0, 0]
    for p in pnls:
        dist_counts[_dist_index(p)] += 1
    distribution = [DistributionBucket(label=_DIST_LABELS[i], count=dist_counts[i]) for i in range(4)]

    # pnl by safety score bucket
    by_score: dict[str, float] = {b[2]: 0.0 for b in _SCORE_BUCKETS}
    for t in trades:
        for lo, hi, lbl in _SCORE_BUCKETS:
            if lo <= t.safetyScore < hi:
                by_score[lbl] += t.pnlSol
                break
    pnl_by_score = [ScorePnl(scoreBucket=b[2], pnlSol=round(by_score[b[2]], 6)) for b in _SCORE_BUCKETS]

    ret_std = _std(returns)
    down_std = _std(downside)
    metrics = BacktestMetrics(
        netPnlSol=round(net, 6),
        winRatePct=round(len(wins) / n * 100.0, 6),
        profitFactor=round(gross_profit / gross_loss, 6) if gross_loss > 0 else (_PF_NO_LOSS if gross_profit > 0 else 0.0),
        sharpe=round(mean_ret / ret_std, 6) if ret_std > 0 else 0.0,
        sortino=round(mean_ret / down_std, 6) if down_std > 0 else 0.0,
        maxDrawdownPct=round(max_dd_pct, 6),
        avgTradeSol=round(net / n, 6),
        rugExposurePct=round(sum(1 for t in trades if t.outcome == "rug") / n * 100.0, 6),
        trades=n,
        avgHoldingHours=round(sum(t.holdingHours for t in trades) / n, 6),
    )

    public_trades = [BacktestTrade(time=t.time, price=t.price, side="buy", pnlSol=round(t.pnlSol, 6)) for t in trades]

    return BacktestResult(
        metrics=metrics, equityCurve=equity, drawdown=drawdown, monthlyReturns=monthly_returns,
        tradeDistribution=distribution, pnlByScore=pnl_by_score,
        priceSeries=[],  # outcome-tabanlıda global fiyat yolu yok (dürüst kalıntı, followup)
        trades=public_trades,
    )
