"""Saf entry-filtre + outcome→PnL modeli (I/O YOK, deterministik, test edilebilir).

Sabitler adlandırılmış kod-sabiti; deploy'da gerçek dağılımla kalibre edilebilir (followup:
kalibrasyon-env). Model outcome-tabanlı — tick-replay değil (bkz spec).
"""
from __future__ import annotations

from dataclasses import dataclass

from app.models import BacktestParams
from app.repo import TokenRow

# outcome → getiri çarpanı sabitleri (size'a oran).
GRAD_RET_DEFAULT = 2.0   # marketCap bilinmiyorsa graduated varsayılan +%200
GRAD_CAP = 5.0           # peak-tabanlı graduated getirisi tavanı (+%500)
ACTIVE_RET = 0.2         # active → ılımlı +%20
RUG_LOSS = 0.95          # rugged → -%95
DEAD_LOSS = 0.5          # dead → -%50

# slippageModel → oran (size'a). realistic varsayılan.
_SLIPPAGE = {"optimistic": 0.005, "realistic": 0.02, "pessimistic": 0.05}
_SLIPPAGE_DEFAULT = 0.02

# outcome → nominal tutma süresi (saat) — outcome-tabanlıda gerçek süre yok, model.
_HOLDING_HOURS = {"graduated": 48.0, "active": 24.0, "dumped": 6.0, "rugged": 1.0, "dead": 72.0}


@dataclass
class SimTrade:
    """Dahili işlem — agregasyon için zengin alanlar (public BacktestTrade alt kümesi)."""
    time: int
    price: float
    pnlSol: float
    size: float
    outcome: str
    safetyScore: float
    holdingHours: float


def _return_multiplier(t: TokenRow) -> float | None:
    """outcome → getiri çarpanı. Bilinmeyen/skorsuz → None (işlem yok)."""
    o = t.outcome
    if o == "graduated":
        if t.marketCap > 0 and t.peakMarketCap > t.marketCap:
            return min(t.peakMarketCap / t.marketCap - 1.0, GRAD_CAP)
        return GRAD_RET_DEFAULT
    if o == "active":
        return ACTIVE_RET
    if o == "dumped":
        return -(t.maxDrawdownPct / 100.0)
    if o == "rugged":
        return -RUG_LOSS
    if o == "dead":
        return -DEAD_LOSS
    return None  # "" / bilinmeyen → skorlanmamış, işlem yok


def _slippage_rate(model: str) -> float:
    return _SLIPPAGE.get(model, _SLIPPAGE_DEFAULT)


def simulate_trade(t: TokenRow, params: BacktestParams) -> SimTrade | None:
    """Entry filtresi + outcome→PnL. Geçmez/skorsuz → None."""
    if t.creatorScore < params.minCreatorScore or t.safetyScore < params.minTokenSafety:
        return None
    ret = _return_multiplier(t)
    if ret is None:
        return None
    size = params.initialCapitalSol / params.maxPositions if params.maxPositions > 0 else params.initialCapitalSol
    gross = size * ret
    cost = size * _slippage_rate(params.slippageModel) + params.priorityFee
    return SimTrade(
        time=t.firstSeenTs, price=t.price, pnlSol=gross - cost, size=size,
        outcome=t.outcome, safetyScore=t.safetyScore,
        holdingHours=_HOLDING_HOURS.get(t.outcome, 24.0),
    )


def run(tokens: list[TokenRow], params: BacktestParams) -> list[SimTrade]:
    """Tüm token'ları filtrele+simüle; nitelikli olanları kronolojik (firstSeenTs) döndür.

    Not: maxPositions POZİSYON BÜYÜKLÜĞÜ bölücüsüdür (eş-ağırlık; eşzamanlı-pozisyon vekili),
    işlem SAYISI cap'i DEĞİL — tarihsel taramada tüm nitelikli token'lar işlem olur.
    """
    trades = [st for t in tokens if (st := simulate_trade(t, params)) is not None]
    trades.sort(key=lambda s: s.time)
    return trades
