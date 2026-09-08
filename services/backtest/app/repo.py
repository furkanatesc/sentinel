"""Postgres okuma katmanı — DB I/O YALNIZ burada (saf simulate/aggregate izole kalsın).

Go tarafındaki tokens/creators şemasıyla parity: creator_score = creators.reputation_score
(LEFT JOIN, skorlanmamış/creator'sız → 0), outcome/peak/drawdown 0007 migration'ından.
"""
from __future__ import annotations

from dataclasses import dataclass


@dataclass
class TokenRow:
    mint: str
    firstSeenTs: int
    creatorScore: float
    safetyScore: float
    outcome: str
    peakMarketCap: float
    maxDrawdownPct: float
    liquidity: float
    price: float
    marketCap: float


_QUERY = """
SELECT t.mint, t.first_seen_ts, COALESCE(c.reputation_score, 0), t.safety_score,
       t.outcome, t.peak_market_cap, t.max_drawdown_pct, t.liquidity, t.price, t.market_cap_usd
FROM tokens t
LEFT JOIN creators c ON c.address = t.creator
WHERE t.first_seen_ts >= %s
ORDER BY t.first_seen_ts ASC
"""


def load_tokens(conn, range_start_ts: int) -> list[TokenRow]:
    """range_start_ts'ten (unix sn) itibaren keşfedilmiş token'ları kronolojik döndürür."""
    with conn.cursor() as cur:
        cur.execute(_QUERY, (range_start_ts,))
        rows = cur.fetchall()
    return [
        TokenRow(
            mint=r[0], firstSeenTs=int(r[1]), creatorScore=float(r[2]), safetyScore=float(r[3]),
            outcome=r[4], peakMarketCap=float(r[5]), maxDrawdownPct=float(r[6]),
            liquidity=float(r[7]), price=float(r[8]), marketCap=float(r[9]),
        )
        for r in rows
    ]
