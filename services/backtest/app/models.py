"""Pydantic modelleri — frontend types.ts ile birebir JSON (camelCase alanlar).

Alan adları kasıtlı camelCase: frontend BacktestParams/BacktestResult sözleşmesiyle
serileştirme parity'si için (alias makinesi yerine doğrudan camelCase — basit + DRY).
"""
from __future__ import annotations

from pydantic import BaseModel


class BacktestParams(BaseModel):
    strategyId: str
    rangePreset: str
    initialCapitalSol: float
    maxPositions: int
    slippageModel: str
    priorityFee: float
    latencyModel: str
    liquidityModel: str
    minCreatorScore: float
    minTokenSafety: float


class EquityPoint(BaseModel):
    t: int
    v: float


class DrawdownPoint(BaseModel):
    t: int
    v: float


class MonthlyReturn(BaseModel):
    label: str
    pct: float


class DistributionBucket(BaseModel):
    label: str
    count: int


class ScorePnl(BaseModel):
    scoreBucket: str
    pnlSol: float


class BacktestTrade(BaseModel):
    time: int
    price: float
    side: str  # "buy" | "sell"
    pnlSol: float


class BacktestMetrics(BaseModel):
    netPnlSol: float
    winRatePct: float
    profitFactor: float
    sharpe: float
    sortino: float
    maxDrawdownPct: float
    avgTradeSol: float
    rugExposurePct: float
    trades: int
    avgHoldingHours: float


class BacktestResult(BaseModel):
    metrics: BacktestMetrics
    equityCurve: list[EquityPoint]
    drawdown: list[DrawdownPoint]
    monthlyReturns: list[MonthlyReturn]
    tradeDistribution: list[DistributionBucket]
    pnlByScore: list[ScorePnl]
    priceSeries: list[EquityPoint]
    trades: list[BacktestTrade]
