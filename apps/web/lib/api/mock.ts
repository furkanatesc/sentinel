import type { SentinelApi } from "./contract";
import type { Kpi, TokenRow, AlertEvent, RadarPoint } from "./types";
import { scoreToLevel, formatUsd, riskMeta } from "@/lib/format";
import type { ScoreKey } from "@/lib/token/score-defs";
import type { RiskSeverity } from "@/lib/format";
import type { ScoreDetail, RiskItem, RiskGroups, SeriesPoint, TokenDetail, EventType, FeedEvent, GraphNode, GraphEdge, WalletGraph, CreatorRow, CreatorProfile, CreatorTokenHistoryItem, CreatorOutcome, LiquidityStatus, StrategyRow, StrategyDetail, StrategyCondition, EquityPoint, StrategyStatus, PortfolioSummary, PortfolioOverview, StrategyPnl, AllocationSlice, WinLossBucket, Position, Candle, MarketData, Order, OrderStatus, Txn, TradeLog, BacktestParams, BacktestResult, BacktestMetrics, DrawdownPoint, BacktestTrade } from "./types";
import { EVENT_SEVERITY } from "@/lib/feed/event-defs";
import { LAUNCHPADS, DEXES } from "@/lib/feed/sources";

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
  { id: "detected", label: "Tespit Edilen Token (24s)", value: "3,412", change: 12.4, spark: spark(3), updated: "12sn önce" },
  { id: "highconf", label: "Yüksek Güvenli Token", value: "184", change: 8.1, spark: spark(7), updated: "12sn önce", tone: "positive" },
  { id: "critical", label: "Kritik Risk Tespiti", value: "97", change: 23.5, spark: spark(11), updated: "8sn önce", tone: "critical" },
  { id: "signals", label: "Aktif Sinyaller", value: "26", change: -4.2, spark: spark(5), updated: "3sn önce" },
  { id: "positions", label: "Açık Pozisyonlar", value: "7", change: 0, spark: spark(9), updated: "1dk önce" },
  { id: "realized", label: "Gerçekleşen K/Z (24s)", value: "+$4,182", change: 6.7, spark: spark(13), updated: "45sn önce", tone: "positive" },
  { id: "unrealized", label: "Gerçekleşmemiş K/Z", value: "-$612", change: -2.1, spark: spark(2), updated: "5sn önce", tone: "warning" },
  { id: "latency", label: "Sistem Gecikmesi", value: "142 ms", change: -11.0, spark: spark(6), updated: "2sn önce" },
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
  { id: "a1", type: "Balina Alımı", token: "PULSE", detail: "4Fk2 cüzdanı 182 SOL aldı", severity: "positive", time: "az önce" },
  { id: "a2", type: "Likidite Çekildi", token: "GFROG", detail: "Üretici havuzun %92'sini çekti", severity: "critical", time: "18sn önce" },
  { id: "a3", type: "Skor Değişti", token: "NOVA", detail: "Güvenlik 62 → 52", severity: "warning", time: "44sn önce" },
  { id: "a4", type: "İlk Likidite", token: "LMN", detail: "$141K havuz oluşturuldu", severity: "info", time: "1dk önce" },
  { id: "a5", type: "Üretici Satışı", token: "ZAP", detail: "Üretici arzın %14'ünü sattı", severity: "warning", time: "2dk önce" },
  { id: "a6", type: "İşlem Doldu", token: "HLS", detail: "3.2 SOL alım @ 0.109", severity: "positive", time: "3dk önce" },
  { id: "a7", type: "Yeni Mint", token: "MCAT", detail: "Pump.fun'da token oluşturuldu", severity: "info", time: "4dk önce" },
  { id: "a8", type: "Şüpheli Küme", token: "GFROG", detail: "5 bağlantılı cüzdan tespit edildi", severity: "critical", time: "5dk önce" },
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

const clamp = (n: number) => Math.max(0, Math.min(100, Math.round(n)));
const seedOf = (s: string) => s.split("").reduce((a, c) => a + c.charCodeAt(0), 0);
const toSeries = (seed: number, len = 24): SeriesPoint[] => spark(seed, len).map((v, t) => ({ t, v }));

function scoreDetail(key: ScoreKey, value: number, seed: number, breakdown: ScoreDetail["breakdown"]): ScoreDetail {
  return { key, value: clamp(value), confidence: clamp(60 + (seed % 35)), updatedAt: "az önce", breakdown };
}

function buildDetail(row: TokenRow): TokenDetail {
  const seed = seedOf(row.symbol);
  const creatorRep = row.creatorScore;
  const safety = row.safetyScore;
  const manip = clamp(100 - safety * 0.7 - creatorRep * 0.3 + 10);
  const opp = clamp(0.35 * row.momentum + 0.35 * creatorRep + 0.3 * safety);

  const scores: Record<ScoreKey, ScoreDetail> = {
    opportunity: scoreDetail("opportunity", opp, seed + 1, [
      { label: "Momentum", weight: 35, detail: `Momentum skoru ${row.momentum}/100` },
      { label: "Üretici itibarı", weight: 35, detail: `Üretici skoru ${creatorRep}/100` },
      { label: "Token güvenliği", weight: 30, detail: `Güvenlik skoru ${safety}/100` },
    ]),
    creatorReputation: scoreDetail("creatorReputation", creatorRep, seed + 2, [
      { label: "Geçmiş performans", weight: 40, detail: creatorRep < 50 ? "Son tokenların çoğu 24s içinde büyük değer kaybetti" : "Geçmiş tokenlar makul performans gösterdi" },
      { label: "Bağlantılı cüzdanlar", weight: 35, detail: creatorRep < 50 ? "Bağlantılı cüzdanlarda likidite çekme paterni" : "Bağlantılı cüzdanlarda anormallik yok" },
      { label: "Cüzdan yaşı", weight: 25, detail: "Fonlayan cüzdan geçmişi incelendi" },
    ]),
    tokenSafety: scoreDetail("tokenSafety", safety, seed + 3, [
      { label: "Kontrat yetkileri", weight: 45, detail: safety < 60 ? "Mint/freeze authority aktif" : "Kritik yetkiler devre dışı" },
      { label: "Metadata", weight: 30, detail: safety < 60 ? "Değiştirilebilir metadata" : "Metadata kilitli" },
      { label: "Likidite kilidi", weight: 25, detail: row.liquidity < 20000 ? "Düşük/kilitsiz likidite" : "Yeterli likidite" },
    ]),
    manipulationRisk: scoreDetail("manipulationRisk", manip, seed + 4, [
      { label: "Holder yoğunluğu", weight: 40, detail: "Top-10 holder oranı izleniyor" },
      { label: "Wash trading", weight: 35, detail: manip > 60 ? "Şüpheli koordineli işlem paterni" : "Belirgin wash trading yok" },
      { label: "Sniper/bot", weight: 25, detail: manip > 60 ? "İlk alıcılarda yüksek bot oranı" : "Bot aktivitesi normal" },
    ]),
  };

  const metrics = {
    holders: row.holders,
    uniqueBuyers: Math.round(row.holders * 0.7),
    buyRatio: clamp(45 + (seed % 30)),
    sellRatio: clamp(55 - (seed % 30)),
    creatorHoldingPct: clamp(2 + (seed % 18)),
    top10HolderPct: clamp(18 + (seed % 40)),
    sniperPct: clamp(manip / 3),
    botActivityPct: clamp(manip / 2.5),
  };

  const risks: RiskGroups = { contract: [], market: [], creator: [] };
  const push = (g: RiskItem[], id: string, title: string, severity: RiskSeverity, description: string, evidence?: string) =>
    g.push({ id, title, severity, description, evidence, firstSeen: "12dk önce", lastSeen: "az önce" });
  if (safety < 70) push(risks.contract, `${row.id}-c1`, "Mint authority aktif", safety < 40 ? "critical" : "high", "Yeni token basılabilir; arz kontrol edilemiyor.", "Mint authority: aktif");
  if (safety < 60) push(risks.contract, `${row.id}-c2`, "Değiştirilebilir metadata", "medium", "Metadata update authority hâlâ açık.", "Update authority: aktif");
  if (row.liquidity < 20000) push(risks.market, `${row.id}-m1`, "Düşük likidite", "high", "Havuz sığ; yüksek price impact riski.", `Likidite: ${formatUsd(row.liquidity)}`);
  if (metrics.top10HolderPct > 45) push(risks.market, `${row.id}-m2`, "Konsantre holder dağılımı", "high", "Arzın büyük kısmı az sayıda cüzdanda.", `Top-10: %${metrics.top10HolderPct}`);
  if (creatorRep < 50) push(risks.creator, `${row.id}-cr1`, "Geçmiş rug bağlantıları", creatorRep < 30 ? "critical" : "high", "Üreticinin geçmiş tokenlarında likidite çekme paterni.", "Bağlantılı cüzdan kümesi tespit edildi");
  if (metrics.creatorHoldingPct > 12) push(risks.creator, `${row.id}-cr2`, "Yüksek üretici payı", "medium", "Üretici arzın önemli kısmını elinde tutuyor.", `Üretici payı: %${metrics.creatorHoldingPct}`);
  if (!risks.contract.length && !risks.market.length && !risks.creator.length)
    push(risks.market, `${row.id}-ok`, "Belirgin risk tespit edilmedi", "info", "Otomatik kontrollerde kritik bir bulgu yok.");

  return {
    id: row.id, name: row.name, symbol: row.symbol, mint: row.mint,
    ageSeconds: row.ageSeconds, price: row.price, priceChange24h: (seed % 40) - 15,
    marketCap: row.liquidity * 4, liquidity: row.liquidity, volume24h: row.vol5m * 12,
    scores, metrics,
    series: { price: toSeries(seed + 5), liquidity: toSeries(seed + 6), volume: toSeries(seed + 7), holders: toSeries(seed + 8) },
    risks,
  };
}

const EVENT_TYPES: EventType[] = [
  "new_mint", "metadata_created", "pool_created", "first_swap", "liquidity_added",
  "liquidity_removed", "creator_sell", "whale_buy", "suspicious_cluster", "score_change", "strategy_signal",
];
const EVENT_DETAIL: Record<EventType, string> = {
  new_mint: "Yeni token basıldı", metadata_created: "Metadata oluşturuldu", pool_created: "İlk havuz açıldı",
  first_swap: "İlk swap gerçekleşti", liquidity_added: "Likidite eklendi", liquidity_removed: "Üretici likidite çekti",
  creator_sell: "Üretici satış yaptı", whale_buy: "Balina alımı", suspicious_cluster: "Bağlantılı cüzdan kümesi",
  score_change: "Risk skoru değişti", strategy_signal: "Strateji sinyali üretildi",
};

function buildEvent(i: number, type: EventType): FeedEvent {
  const t = tokens[i % tokens.length];
  const seed = seedOf(t.symbol) + i;
  return {
    id: `ev-${i}-${type}`, type, symbol: t.symbol, mint: t.mint,
    launchpad: LAUNCHPADS[i % LAUNCHPADS.length], dex: DEXES[i % DEXES.length],
    liquidity: t.liquidity, creatorScore: t.creatorScore,
    riskLevel: scoreToLevel(Math.round((t.creatorScore + t.safetyScore) / 2)),
    tokenAgeSeconds: t.ageSeconds + i * 7, volume5m: t.vol5m,
    holderGrowthPct: clamp(5 + (seed % 60)), severity: EVENT_SEVERITY[type],
    detail: `${t.symbol} · ${EVENT_DETAIL[type]}`, time: i === 0 ? "az önce" : `${i * 4}sn önce`, ts: 1000 - i,
    watchlisted: t.watchlisted,
  };
}
const feedEvents: FeedEvent[] = Array.from({ length: 24 }, (_, i) => buildEvent(i, EVENT_TYPES[i % EVENT_TYPES.length]));

function buildWalletGraph(): WalletGraph {
  const short = (s: string) => `${s}...${s.slice(-2)}`;
  const nodes: GraphNode[] = [
    { id: "F1", type: "funding_wallet", label: "Funder-1", address: short("Fnd1Qk"), riskLevel: "high", balanceSol: 420, firstSeen: "3g önce", lastSeen: "az önce" },
    { id: "F2", type: "funding_wallet", label: "Funder-2", address: short("Fnd2Rp"), riskLevel: "medium", balanceSol: 88, firstSeen: "5g önce", lastSeen: "1g önce" },
    { id: "C1", type: "creator_wallet", label: "Creator-A", address: short("CreAxz"), riskLevel: "high", balanceSol: 12, firstSeen: "2g önce", lastSeen: "az önce" },
    { id: "C2", type: "creator_wallet", label: "Creator-B", address: short("CreBmn"), riskLevel: "medium", balanceSol: 5, firstSeen: "2g önce", lastSeen: "3s önce" },
    { id: "C3", type: "creator_wallet", label: "Creator-C", address: short("CreCqw"), riskLevel: "critical", balanceSol: 1, firstSeen: "1g önce", lastSeen: "az önce" },
    { id: "T1", type: "token", label: "PULSE", address: short("9xQeWv"), riskLevel: "good", firstSeen: "1g önce", lastSeen: "az önce" },
    { id: "T2", type: "token", label: "GFROG", address: short("7mLp2c"), riskLevel: "critical", firstSeen: "12s önce", lastSeen: "az önce" },
    { id: "T3", type: "token", label: "LMN", address: short("Cd93Kf"), riskLevel: "strong", firstSeen: "5s önce", lastSeen: "az önce" },
    { id: "P1", type: "liquidity_pool", label: "Havuz-1", riskLevel: "good", firstSeen: "1g önce", lastSeen: "az önce" },
    { id: "P2", type: "liquidity_pool", label: "Havuz-2", riskLevel: "critical", firstSeen: "12s önce", lastSeen: "az önce" },
    { id: "W1", type: "trader_wallet", label: "Trader-1", address: short("Trd1aa"), riskLevel: "medium", balanceSol: 34, firstSeen: "6g önce", lastSeen: "az önce" },
    { id: "W2", type: "trader_wallet", label: "Trader-2", address: short("Trd2bb"), riskLevel: "good", balanceSol: 51, firstSeen: "8g önce", lastSeen: "2s önce" },
    { id: "S1", type: "smart_wallet", label: "Smart-1", address: short("Smt1cc"), riskLevel: "strong", balanceSol: 210, firstSeen: "40g önce", lastSeen: "az önce" },
    { id: "X1", type: "suspicious_wallet", label: "Şüpheli-1", address: short("Sus1dd"), riskLevel: "critical", balanceSol: 3, firstSeen: "1g önce", lastSeen: "az önce" },
    { id: "X2", type: "suspicious_wallet", label: "Şüpheli-2", address: short("Sus2ee"), riskLevel: "critical", balanceSol: 2, firstSeen: "1g önce", lastSeen: "az önce" },
    { id: "E1", type: "exchange_wallet", label: "Borsa", address: short("Cex1ff"), riskLevel: "good", firstSeen: "200g önce", lastSeen: "az önce" },
  ];
  const edges: GraphEdge[] = [
    { id: "g1", source: "E1", target: "F1", type: "transferred" },
    { id: "g2", source: "F1", target: "C1", type: "funded" },
    { id: "g3", source: "F1", target: "C2", type: "funded" },
    { id: "g4", source: "F2", target: "C3", type: "funded" },
    { id: "g5", source: "F1", target: "F2", type: "shares_funder" },
    { id: "g6", source: "C1", target: "T1", type: "created" },
    { id: "g7", source: "C2", target: "T2", type: "created" },
    { id: "g8", source: "C3", target: "T3", type: "created" },
    { id: "g9", source: "C1", target: "T1", type: "controls_authority" },
    { id: "g10", source: "C1", target: "P1", type: "provided_liquidity" },
    { id: "g11", source: "C2", target: "P2", type: "provided_liquidity" },
    { id: "g12", source: "C2", target: "P2", type: "removed_liquidity" },
    { id: "g13", source: "W1", target: "T1", type: "bought" },
    { id: "g14", source: "W2", target: "T1", type: "bought" },
    { id: "g15", source: "S1", target: "T3", type: "bought" },
    { id: "g16", source: "W1", target: "T2", type: "sold" },
    { id: "g17", source: "X1", target: "T2", type: "bought" },
    { id: "g18", source: "X2", target: "T2", type: "bought" },
    { id: "g19", source: "F1", target: "X1", type: "funded" },
    { id: "g20", source: "F1", target: "X2", type: "funded" },
    { id: "g21", source: "X1", target: "X2", type: "transferred" },
    { id: "g22", source: "C1", target: "T2", type: "sold" },
  ];
  return { nodes, edges };
}
const walletGraph = buildWalletGraph();

const OUTCOMES: CreatorOutcome[] = ["active", "graduated", "dumped", "rug", "dead"];
const LIQS: LiquidityStatus[] = ["locked", "unlocked", "removed"];
const CREATOR_ADDRS = ["CreAxz", "CreBmn", "CreCqw", "Dep7hn", "Dep9kf", "Dep2rt"];

function creatorRow(addr: string): CreatorRow {
  const seed = seedOf(addr);
  const rep = clamp(20 + (seed % 70));
  const total = 4 + (seed % 12);
  return {
    address: addr, reputationScore: rep, riskLevel: scoreToLevel(rep),
    totalTokens: total, activeTokens: (seed % 4), ruggedTokens: Math.min(total, (seed % 6)),
    successRatePct: clamp(rep - 10 + (seed % 20)), realizedPnlSol: (seed % 200) - 60,
  };
}

function creatorHistory(addr: string, n: number): CreatorTokenHistoryItem[] {
  const seed = seedOf(addr);
  return Array.from({ length: n }, (_, i) => {
    const s = seed + i * 13;
    const peak = 20000 + (s % 400) * 1000;
    const cur = Math.round(peak * ((s % 90) / 100));
    return {
      id: `${addr}-h${i}`, symbol: `T${(s % 90).toString(36).toUpperCase()}${i}`, mint: `${addr.slice(0,4)}...${i}`,
      createdAt: `${i + 1}g önce`, peakMarketCap: peak, currentMarketCap: cur,
      maxDrawdownPct: clamp(100 - (cur / peak) * 100), liquidityStatus: LIQS[s % LIQS.length],
      creatorSellPct: clamp(s % 60), outcome: OUTCOMES[s % OUTCOMES.length],
      riskFlags: s % 2 === 0 ? ["Mint authority aktif"] : [],
    };
  });
}

function buildCreator(addr: string): CreatorProfile {
  const seed = seedOf(addr);
  const rep = clamp(20 + (seed % 70));
  const total = 4 + (seed % 12);
  return {
    address: addr, walletAgeDays: 5 + (seed % 60), firstSeen: `${5 + (seed % 60)}g önce`,
    reputation: {
      key: "creatorReputation", value: rep, confidence: clamp(60 + (seed % 35)), updatedAt: "az önce",
      breakdown: [
        { label: "Geçmiş performans", weight: 40, detail: rep < 50 ? `Son ${total} tokenın çoğu 24s içinde büyük değer kaybetti` : "Geçmiş tokenlar makul performans gösterdi" },
        { label: "Bağlantılı cüzdanlar", weight: 25, detail: rep < 50 ? "Bağlantılı cüzdanlarda likidite çekme paterni" : "Anormallik yok" },
        { label: "Cüzdan yaşı ve fonlama", weight: 20, detail: `Cüzdan ${5 + (seed % 60)} günlük` },
        { label: "Tekrarlanan deploy", weight: 15, detail: total > 8 ? "Yüksek frekanslı deploy paterni" : "Düşük frekans" },
      ],
    },
    riskLevel: scoreToLevel(rep),
    metrics: {
      totalTokens: total, activeTokens: (seed % 4), ruggedTokens: Math.min(total, (seed % 6)),
      avgLifetimeHours: 2 + (seed % 72), avgPeakMarketCap: 30000 + (seed % 300) * 1000,
      realizedPnlSol: (seed % 200) - 60, successRatePct: clamp(rep - 10 + (seed % 20)), avgFirstSellMinutes: 3 + (seed % 90),
    },
    history: creatorHistory(addr, 4 + (seed % 4)),
    behavior: {
      deployFrequency: `${total} token / 30 gün`, avgFirstSellMinutes: 3 + (seed % 90),
      repeatedFunders: [`Fnd${seed % 9}...aa`, `Fnd${(seed + 3) % 9}...bb`],
      similarMetadata: seed % 2 === 0, sameSocial: seed % 3 === 0, sameLiquidityPattern: seed % 2 === 1,
    },
  };
}
const creators: CreatorRow[] = CREATOR_ADDRS.map(creatorRow);

// --- Strategies (Increment 6) ---
const STRATEGY_DEFS: { id: string; name: string; status: StrategyStatus; timeframe: string; desc: string }[] = [
  { id: "momentum-scalp", name: "Momentum Scalp", status: "live", timeframe: "1-5 dk", desc: "Yeni mint sonrası ilk dakikalarda güçlü momentum + güvenli creator ararız." },
  { id: "safe-graduation", name: "Güvenli Graduation", status: "paper", timeframe: "15-60 dk", desc: "Yüksek güvenlik skoru + kilitli likidite ile graduation adaylarını izler." },
  { id: "creator-reputation", name: "Creator İtibar Takibi", status: "shadow", timeframe: "5-30 dk", desc: "Kanıtlanmış creator'ların yeni token'larına erken pozisyon." },
  { id: "liquidity-breakout", name: "Likidite Kırılımı", status: "backtesting", timeframe: "1-10 dk", desc: "Ani likidite artışı + holder büyümesi kombinasyonu." },
  { id: "anti-rug-filter", name: "Anti-Rug Filtre", status: "paused", timeframe: "10-45 dk", desc: "Düşük manipülasyon riski ve dağıtık holder tabanı şartı." },
  { id: "legacy-sniper", name: "Eski Sniper v1", status: "archived", timeframe: "0-2 dk", desc: "Emekliye ayrılmış ilk nesil sniper mantığı." },
];

// Note: reuses the existing module-scoped `seedOf(s: string): number` helper (defined above,
// used by buildDetail/buildEvent/creatorRow/etc.) rather than redeclaring a second `seedOf`,
// which would otherwise collide with that existing const declaration.

function equityCurve(seed: number, len = 40): EquityPoint[] {
  const out: EquityPoint[] = [];
  let v = 100;
  for (let i = 0; i < len; i++) {
    v += Math.sin(seed + i * 0.6) * 3 + ((seed * (i + 1)) % 5) - 1.8;
    out.push({ t: i, v: Math.round(Math.max(20, v) * 100) / 100 });
  }
  return out;
}

function strategyEntry(seed: number): StrategyCondition[] {
  return [
    { metric: "creatorScore", op: ">", value: 70 + (seed % 20) },
    { metric: "tokenSafety", op: ">", value: 65 + (seed % 15) },
    { metric: "liquidity", op: ">", value: 20000 + (seed % 30) * 1000, unit: "USD" },
    { metric: "holderGrowth5m", op: ">", value: 30 + (seed % 40), unit: "%" },
  ];
}
function strategyExit(seed: number): StrategyCondition[] {
  return [
    { metric: "manipulationRisk", op: ">", value: 60 + (seed % 20) },
    { metric: "momentum", op: "<", value: 20 + (seed % 15) },
  ];
}

function strategyRow(def: (typeof STRATEGY_DEFS)[number]): StrategyRow {
  const seed = seedOf(def.id);
  return {
    id: def.id, name: def.name, status: def.status, timeframe: def.timeframe,
    winRatePct: 45 + (seed % 40), profitFactor: Math.round((1 + (seed % 25) / 10) * 100) / 100,
    maxDrawdownPct: 8 + (seed % 25), totalTrades: 40 + (seed % 400),
    netPnlSol: Math.round(((seed % 500) - 120) * 10) / 10, lastSignal: `${1 + (seed % 59)} dk önce`,
  };
}

const strategyRows: StrategyRow[] = STRATEGY_DEFS.map(strategyRow);

function buildStrategy(id: string): StrategyDetail {
  const def = STRATEGY_DEFS.find((d) => d.id === id) ?? STRATEGY_DEFS[0];
  const row = strategyRow(def);
  const seed = seedOf(def.id);
  return {
    id: def.id, name: def.name, status: def.status, timeframe: def.timeframe, description: def.desc,
    entry: strategyEntry(seed), exit: strategyExit(seed),
    risk: { riskPerTradePct: 1 + (seed % 4), stopLossPct: 12 + (seed % 18), takeProfitLevels: [25, 60, 120], maxDrawdownStopPct: 20 + (seed % 15) },
    sizing: { model: seed % 2 === 0 ? "Sabit %" : "Kelly kesirli", sizePct: 2 + (seed % 6) },
    supportedLaunchpads: [LAUNCHPADS[seed % LAUNCHPADS.length], LAUNCHPADS[(seed + 2) % LAUNCHPADS.length]],
    minScores: { creator: 60 + (seed % 25), safety: 55 + (seed % 30) },
    performance: {
      winRatePct: row.winRatePct, profitFactor: row.profitFactor, maxDrawdownPct: row.maxDrawdownPct,
      sharpe: Math.round((0.8 + (seed % 20) / 10) * 100) / 100, sortino: Math.round((1 + (seed % 25) / 10) * 100) / 100,
      totalTrades: row.totalTrades, netPnlSol: row.netPnlSol, expectancy: Math.round(((seed % 30) / 10 - 1) * 100) / 100,
    },
    equityCurve: equityCurve(seed),
    backtest: {
      netPnlSol: row.netPnlSol, winRatePct: row.winRatePct, profitFactor: row.profitFactor,
      sharpe: Math.round((0.8 + (seed % 20) / 10) * 100) / 100, maxDrawdownPct: row.maxDrawdownPct,
      trades: row.totalTrades, avgHoldingHours: 1 + (seed % 12), rugExposurePct: (seed % 8),
    },
    versions: [
      { version: "v1.3", date: "2026-07-28", note: "Stop-loss %2 sıkılaştırıldı" },
      { version: "v1.2", date: "2026-07-20", note: "Min creator skoru 70'e çıkarıldı" },
      { version: "v1.0", date: "2026-07-05", note: "İlk yayın" },
    ],
    audit: [
      { time: "2026-07-30 14:22", action: "Duraklatıldı", detail: "Drawdown eşiği aşıldı" },
      { time: "2026-07-29 09:10", action: "Parametre güncellendi", detail: "Take-profit L2 %60" },
      { time: "2026-07-28 18:45", action: "Yayınlandı", detail: "Shadow → Live" },
    ],
  };
}

// --- Portfolio & Positions (Increment 7) ---
function portfolioEquity(seed: number, len = 40): EquityPoint[] {
  const out: EquityPoint[] = [];
  let v = 500;
  for (let i = 0; i < len; i++) {
    v += Math.sin(seed + i * 0.5) * 12 + ((seed * (i + 1)) % 9) - 4;
    out.push({ t: i, v: Math.round(Math.max(50, v) * 100) / 100 });
  }
  return out;
}

const portfolioOverview: PortfolioOverview = (() => {
  const seed = 4242;
  const invested = 640, available = 210;
  const unrealized = 84.5, realized = 312.7, daily = -18.2;
  const summary: PortfolioSummary = {
    totalValueSol: Math.round((invested + available + unrealized) * 10) / 10,
    availableSol: available, investedSol: invested,
    realizedPnlSol: realized, unrealizedPnlSol: unrealized, dailyPnlSol: daily,
    maxDrawdownPct: 22, riskExposurePct: 68, rugExposurePct: 4,
  };
  const pnlByStrategy: StrategyPnl[] = STRATEGY_DEFS.slice(0, 5).map((d, i) => ({
    strategyId: d.id, name: d.name, pnlSol: Math.round((Math.sin(seed + i) * 120 + (i % 3 === 2 ? -60 : 90)) * 10) / 10,
  }));
  const riskAllocation: AllocationSlice[] = [
    { label: riskMeta.strong.label, pct: 34, color: "#2FD98B" },
    { label: riskMeta.good.label, pct: 28, color: "#17B890" },
    { label: riskMeta.medium.label, pct: 22, color: "#3E9BFF" },
    { label: riskMeta.high.label, pct: 12, color: "#FFB020" },
    { label: riskMeta.critical.label, pct: 4, color: "#F0476B" },
  ];
  const winLoss: WinLossBucket[] = [
    { label: "Büyük Kazanç", count: 14 }, { label: "Kazanç", count: 38 },
    { label: "Başabaş", count: 9 }, { label: "Kayıp", count: 21 }, { label: "Büyük Kayıp", count: 6 },
  ];
  return { summary, equityCurve: portfolioEquity(seed), pnlByStrategy, riskAllocation, winLoss };
})();

const POSITION_TOKENS: { symbol: string; mint: string }[] = [
  { symbol: "PULSE", mint: "9xQeWv...4Fk2" }, { symbol: "LMN", mint: "Cd93Kf...6Rt4" },
  { symbol: "HLS", mint: "Hh77Nb...2Kl9" }, { symbol: "PXL", mint: "Ii22Vc...5Dw3" },
  { symbol: "MCAT", mint: "Ff01Xq...3Bn7" }, { symbol: "NOVA", mint: "Ap12Rd...9Zk1" },
  { symbol: "ZAP", mint: "Gg44Lm...8Yu2" }, { symbol: "GFROG", mint: "7mLp2c...1Qw8" },
];

const positions: Position[] = POSITION_TOKENS.map((tok, i) => {
  const strat = STRATEGY_DEFS[i % STRATEGY_DEFS.length];
  const seed = seedOf(tok.symbol + strat.id);
  const entry = Math.round((0.0004 + (seed % 90) / 100000) * 1e6) / 1e6;
  const pnlPct = ((seed % 160) - 60);
  const current = Math.round(entry * (1 + pnlPct / 100) * 1e6) / 1e6;
  const size = 4 + (seed % 30);
  return {
    id: `pos-${i + 1}`, tokenMint: tok.mint, tokenSymbol: tok.symbol,
    strategyId: strat.id, strategyName: strat.name,
    entryPrice: entry, currentPrice: current, sizeSol: size,
    pnlSol: Math.round(size * (pnlPct / 100) * 10) / 10, pnlPct,
    stopLossPct: 12 + (seed % 10), takeProfitPct: 40 + (seed % 60),
    tokenRisk: scoreToLevel(10 + (seed % 85)), creatorRisk: scoreToLevel(35 + ((seed * 3) % 60)),
    ageLabel: `${1 + (seed % 46)} dk`, openedAt: `${1 + (seed % 46)} dk önce`,
  };
});

// --- Trading Terminal (Increment 8) ---
const px = (n: number) => Math.round(n * 1e6) / 1e6;

function buildCandles(mint: string): Candle[] {
  const seed = seedOf(mint);
  const out: Candle[] = [];
  let base = 0.001 + (seed % 60) / 20000;
  for (let i = 0; i < 60; i++) {
    const open = base;
    const drift = Math.sin(seed + i * 0.7) * base * 0.05 + (((seed * (i + 1)) % 7) - 3) * base * 0.01;
    const close = Math.max(base * 0.4, open + drift);
    const high = Math.max(open, close) * (1 + ((seed + i) % 4) / 100);
    const low = Math.min(open, close) * (1 - ((seed + i) % 3) / 100);
    out.push({ time: 1_700_000_000 + i * 300, open: px(open), high: px(high), low: px(low), close: px(close) });
    base = close;
  }
  return out;
}

function buildMarketData(mint: string): MarketData {
  const q = mint.toLowerCase();
  const row = tokens.find((t) => t.mint.toLowerCase() === q || t.symbol.toLowerCase() === q || t.id.toLowerCase() === q) ?? tokens[0];
  const seed = seedOf(row.mint);
  return {
    mint: row.mint, symbol: row.symbol,
    price: row.price, change24hPct: (seed % 40) - 15,
    liquiditySol: Math.round(row.liquidity / 100),
    volume24hSol: Math.round((row.vol5m * 12) / 100),
    marketCapSol: Math.round((row.liquidity * 4) / 100),
    tokenScore: row.safetyScore, creatorScore: row.creatorScore,
  };
}

const ORDER_STATUSES: OrderStatus[] = ["open", "filled", "cancelled"];
const orders: Order[] = POSITION_TOKENS.slice(0, 6).map((t, i) => {
  const seed = seedOf(t.mint) + i;
  return {
    id: `ord-${i + 1}`, tokenSymbol: t.symbol, tokenMint: t.mint,
    side: seed % 2 === 0 ? "buy" : "sell",
    type: seed % 3 === 0 ? "limit" : "market",
    status: ORDER_STATUSES[seed % 3],
    price: px(0.001 + (seed % 40) / 10000),
    amountSol: 5 + (seed % 25), createdAt: `${2 + (seed % 40)} dk önce`,
  };
});
// Guarantee at least one deterministically "open" order so the cancel (İptal) flow
// always has something to exercise, regardless of how the seed-derived status lands.
orders[0].status = "open";

const TXN_KINDS = ["buy", "sell", "approve"] as const;
const TXN_STATUSES = ["success", "pending", "failed"] as const;
const transactions: Txn[] = POSITION_TOKENS.map((t, i) => {
  const seed = seedOf(t.mint) + i * 3;
  return {
    id: `tx-${i + 1}`,
    hash: `${t.mint.slice(0, 4)}${seed.toString(16)}...${(seed * 7).toString(16).slice(-4)}`,
    kind: TXN_KINDS[seed % 3], tokenSymbol: t.symbol, amountSol: 3 + (seed % 30),
    status: TXN_STATUSES[seed % 3], time: `${1 + (seed % 55)} dk önce`,
  };
});

const LOG_MESSAGES = [
  "Sinyal alındı: momentum eşiği aşıldı",
  "Emir simülasyonu tamamlandı",
  "Likidite kontrolü geçti",
  "Creator skoru güncellendi",
  "Slippage toleransı yeniden hesaplandı",
  "Risk limiti kontrolü: uygun",
];
const LOG_LEVELS = ["info", "warn", "error"] as const;
const tradeLogs: TradeLog[] = Array.from({ length: 10 }, (_, i) => {
  const seed = 41 + i * 7;
  return { id: `log-${i + 1}`, level: LOG_LEVELS[seed % 3], message: LOG_MESSAGES[i % LOG_MESSAGES.length], time: `${i * 2 + 1} dk önce` };
});

// --- Backtesting (Increment 9) ---
const BT_MONTHS = ["Oca", "Şub", "Mar", "Nis", "May", "Haz", "Tem", "Ağu"];
const BT_DIST = ["< -5", "-5..0", "0..5", "5..10", "> 10"];
const BT_SCORE_BUCKETS = ["0-24", "25-49", "50-69", "70-84", "85-100"];

function runBacktestResult(p: BacktestParams): BacktestResult {
  // `seedOf` is a shared helper (flat char-code sum) used by many other mock builders, so it is
  // NOT modified here. A plain delimited concatenation would still collide on multiset-preserving
  // param swaps (e.g. minCreatorScore=60/minTokenSafety=55 vs 55/60) because char-code summation is
  // commutative regardless of delimiters. Instead each field's stringified value is repeated
  // (index + 1) times before joining, which folds its ordinal position into the char-sum as a
  // multiplicative weight (charSum(v.repeat(n)) === n * charSum(v)) — a standard transposition-
  // detecting checksum technique — making the seed order-sensitive without touching `seedOf`.
  const btFields = [
    p.strategyId, p.rangePreset, String(p.initialCapitalSol), String(p.maxPositions),
    p.slippageModel, String(p.priorityFee), p.latencyModel, p.liquidityModel,
    String(p.minCreatorScore), String(p.minTokenSafety),
  ];
  const seed = seedOf(btFields.map((v, i) => v.repeat(i + 1)).join("|"));
  const trades = 40 + (seed % 300);
  const netPnl = (seed % 200) - 60 + p.initialCapitalSol * 0.1;
  const r2 = (n: number) => Math.round(n * 100) / 100;
  const metrics: BacktestMetrics = {
    netPnlSol: Math.round(netPnl * 10) / 10,
    winRatePct: 40 + (seed % 45),
    profitFactor: r2(1 + (seed % 250) / 100),
    sharpe: r2((seed % 300) / 100),
    sortino: r2((seed % 350) / 100),
    maxDrawdownPct: 5 + (seed % 35),
    avgTradeSol: r2(netPnl / Math.max(1, trades)),
    rugExposurePct: seed % 10,
    trades,
    avgHoldingHours: 1 + (seed % 12),
  };
  const equityCurve = toSeries(seed, 30);
  const priceSeries = toSeries(seed + 7, 40);
  const drawdown: DrawdownPoint[] = equityCurve.map((pt, i) => ({ t: pt.t, v: -(((seed + i * 3) % 30)) }));
  const monthlyReturns = BT_MONTHS.map((m, i) => ({ label: m, pct: ((seed + i * 13) % 40) - 15 }));
  const tradeDistribution = BT_DIST.map((b, i) => ({ label: b, count: 3 + ((seed + i * 7) % 25) }));
  const pnlByScore = BT_SCORE_BUCKETS.map((b, i) => ({ scoreBucket: b, pnlSol: ((seed + i * 11) % 60) - 20 }));
  const btTrades: BacktestTrade[] = [];
  priceSeries.forEach((pt, i) => {
    if (i % 6 === 2) btTrades.push({ time: pt.t, price: pt.v, side: btTrades.length % 2 === 0 ? "buy" : "sell", pnlSol: ((seed + i) % 20) - 8 });
  });
  return { metrics, equityCurve, drawdown, monthlyReturns, tradeDistribution, pnlByScore, priceSeries, trades: btTrades };
}

export const mockApi: SentinelApi = {
  getKpis: () => delay(kpis),
  getTokens: () => delay(tokens),
  getAlerts: () => delay(alerts),
  getRadar: () => delay(radarFrom(tokens)),
  getEvents: () => delay(feedEvents),
  getWalletGraph: () => delay(walletGraph),
  getAuthorityGraph: () => delay({ nodes: [], edges: [] }),
  getCreators: () => delay(creators),
  getCreator: (address) => delay(buildCreator(address)),
  getStrategies: () => delay(strategyRows),
  getStrategy: (id) => delay(buildStrategy(id)),
  getPortfolio: () => delay(portfolioOverview),
  getPositions: () => delay(positions),
  getCandles: (mint) => delay(buildCandles(mint)),
  getMarketData: (mint) => delay(buildMarketData(mint)),
  getOrders: () => delay(orders),
  getTransactions: () => delay(transactions),
  getTradeLogs: () => delay(tradeLogs),
  runBacktest: (params) => delay(runBacktestResult(params)),

  getToken(idOrMint) {
    const q = idOrMint.toLowerCase();
    const row = tokens.find((t) => t.symbol.toLowerCase() === q || t.id.toLowerCase() === q || t.mint.toLowerCase() === q);
    if (!row) return Promise.reject(new Error(`Token bulunamadı: ${idOrMint}`));
    return delay(buildDetail(row));
  },

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
      { id: "live1", type: "Yeni Mint", token: "AERO", detail: "Raydium'da token oluşturuldu", severity: "info", time: "az önce" },
      { id: "live2", type: "Balina Alımı", token: "LMN", detail: "6Rt4 cüzdanı 90 SOL aldı", severity: "positive", time: "az önce" },
    ];
    let n = 0;
    const id = setInterval(() => {
      cb({ ...pool[n % pool.length], id: `live-${Date.now()}` });
      n++;
    }, 5000);
    return () => clearInterval(id);
  },

  subscribeEvents(cb) {
    let n = 0;
    const id = setInterval(() => {
      const type = EVENT_TYPES[n % EVENT_TYPES.length];
      cb({ ...buildEvent(n % tokens.length, type), id: `live-${n}-${type}`, time: "az önce", ts: 2000 + n });
      n++;
    }, 3000);
    return () => clearInterval(id);
  },
};
