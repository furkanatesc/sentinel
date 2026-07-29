import type { SentinelApi } from "./contract";
import type { Kpi, TokenRow, AlertEvent, RadarPoint } from "./types";
import { scoreToLevel } from "@/lib/format";

function spark(seed: number, len = 16): number[] {
  const out: number[] = [];
  let v = 50 + (seed % 20);
  for (let i = 0; i < len; i++) {
    v += Math.sin(seed + i * 1.3) * 8 + ((seed * (i + 1)) % 7) - 3;
    out.push(Math.max(4, Math.round(v)));
  }
  return out;
}

const kpis: Kpi[] = [
  { id: "detected", label: "Tokens Detected (24h)", value: "3,412", change: 12.4, spark: spark(3), updated: "12s ago" },
  { id: "highconf", label: "High Confidence Tokens", value: "184", change: 8.1, spark: spark(7), updated: "12s ago", tone: "positive" },
  { id: "critical", label: "Critical Risk Detections", value: "97", change: 23.5, spark: spark(11), updated: "8s ago", tone: "critical" },
  { id: "signals", label: "Active Signals", value: "26", change: -4.2, spark: spark(5), updated: "3s ago" },
  { id: "positions", label: "Open Positions", value: "7", change: 0, spark: spark(9), updated: "1m ago" },
  { id: "realized", label: "Realized PnL (24h)", value: "+$4,182", change: 6.7, spark: spark(13), updated: "45s ago", tone: "positive" },
  { id: "unrealized", label: "Unrealized PnL", value: "-$612", change: -2.1, spark: spark(2), updated: "5s ago", tone: "warning" },
  { id: "latency", label: "System Latency", value: "142 ms", change: -11.0, spark: spark(6), updated: "2s ago" },
];

const tokens: TokenRow[] = [
  { id: "t1", name: "SolPulse", symbol: "PULSE", mint: "9xQeWv...4Fk2", ageSeconds: 38, price: 0.0042, liquidity: 82400, vol5m: 41200, holders: 312, creatorScore: 82, safetyScore: 78, momentum: 88, spark: spark(21), signal: "buy", watchlisted: true },
  { id: "t2", name: "NovaByte", symbol: "NOVA", mint: "Ap12Rd...9Zk1", ageSeconds: 95, price: 0.00011, liquidity: 15600, vol5m: 8900, holders: 96, creatorScore: 44, safetyScore: 52, momentum: 61, spark: spark(34), signal: "watch", watchlisted: false },
  { id: "t3", name: "GigaFrog", symbol: "GFROG", mint: "7mLp2c...1Qw8", ageSeconds: 12, price: 0.0000009, liquidity: 4200, vol5m: 22100, holders: 41, creatorScore: 17, safetyScore: 22, momentum: 74, spark: spark(48), signal: "avoid", watchlisted: false },
  { id: "t4", name: "Lumen", symbol: "LMN", mint: "Cd93Kf...6Rt4", ageSeconds: 210, price: 0.019, liquidity: 141000, vol5m: 63400, holders: 884, creatorScore: 91, safetyScore: 86, momentum: 79, spark: spark(15), signal: "buy", watchlisted: true },
  { id: "t5", name: "MoonCat", symbol: "MCAT", mint: "Ff01Xq...3Bn7", ageSeconds: 320, price: 0.0007, liquidity: 28900, vol5m: 12300, holders: 210, creatorScore: 63, safetyScore: 58, momentum: 55, spark: spark(27), signal: "watch", watchlisted: false },
  { id: "t6", name: "ZapDog", symbol: "ZAP", mint: "Gg44Lm...8Yu2", ageSeconds: 480, price: 0.0031, liquidity: 9800, vol5m: 3100, holders: 58, creatorScore: 29, safetyScore: 34, momentum: 42, spark: spark(39), signal: "avoid", watchlisted: false },
  { id: "t7", name: "Helios", symbol: "HLS", mint: "Hh77Nb...2Kl9", ageSeconds: 640, price: 0.11, liquidity: 320000, vol5m: 98000, holders: 1420, creatorScore: 88, safetyScore: 90, momentum: 71, spark: spark(19), signal: "buy", watchlisted: true },
  { id: "t8", name: "PixelFi", symbol: "PXL", mint: "Ii22Vc...5Dw3", ageSeconds: 830, price: 0.00054, liquidity: 46700, vol5m: 15600, holders: 340, creatorScore: 71, safetyScore: 66, momentum: 63, spark: spark(31), signal: "watch", watchlisted: false },
];

const alerts: AlertEvent[] = [
  { id: "a1", type: "Whale Buy", token: "PULSE", detail: "Wallet 4Fk2 bought 182 SOL", severity: "positive", time: "just now" },
  { id: "a2", type: "Liquidity Removed", token: "GFROG", detail: "Creator pulled 92% of pool", severity: "critical", time: "18s ago" },
  { id: "a3", type: "Score Change", token: "NOVA", detail: "Safety 62 → 52", severity: "warning", time: "44s ago" },
  { id: "a4", type: "First Liquidity", token: "LMN", detail: "$141K pool created", severity: "info", time: "1m ago" },
  { id: "a5", type: "Creator Sell", token: "ZAP", detail: "Creator sold 14% of supply", severity: "warning", time: "2m ago" },
  { id: "a6", type: "Trade Filled", token: "HLS", detail: "Buy 3.2 SOL @ 0.109", severity: "positive", time: "3m ago" },
  { id: "a7", type: "New Mint", token: "MCAT", detail: "Token created on Pump.fun", severity: "info", time: "4m ago" },
  { id: "a8", type: "Suspicious Cluster", token: "GFROG", detail: "5 linked wallets detected", severity: "critical", time: "5m ago" },
];

function radarFrom(list: TokenRow[]): RadarPoint[] {
  return list.map((t) => ({
    x: t.creatorScore,
    y: t.momentum,
    z: t.liquidity,
    name: t.symbol,
    level: scoreToLevel(Math.round((t.creatorScore + t.safetyScore) / 2)),
  }));
}

const delay = <T,>(v: T) => new Promise<T>((r) => setTimeout(() => r(v), 40));

export const mockApi: SentinelApi = {
  getKpis: () => delay(kpis),
  getTokens: () => delay(tokens),
  getAlerts: () => delay(alerts),
  getRadar: () => delay(radarFrom(tokens)),

  subscribeTokens(cb) {
    // Simulate live feed: nudge the youngest token's age/momentum every 2.5s.
    let live = tokens.map((t) => ({ ...t }));
    const id = setInterval(() => {
      live = live.map((t, i) =>
        i === 0 ? { ...t, ageSeconds: t.ageSeconds + 3, momentum: Math.min(100, t.momentum + 1) } : t
      );
      cb(live);
    }, 2500);
    return () => clearInterval(id);
  },

  subscribeAlerts(cb) {
    const pool: AlertEvent[] = [
      { id: "live1", type: "New Mint", token: "AERO", detail: "Token created on Raydium", severity: "info", time: "just now" },
      { id: "live2", type: "Whale Buy", token: "LMN", detail: "Wallet 6Rt4 bought 90 SOL", severity: "positive", time: "just now" },
    ];
    let n = 0;
    const id = setInterval(() => {
      cb({ ...pool[n % pool.length], id: `live-${Date.now()}` });
      n++;
    }, 5000);
    return () => clearInterval(id);
  },
};
