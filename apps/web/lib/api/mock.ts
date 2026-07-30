import type { SentinelApi } from "./contract";
import type { Kpi, TokenRow, AlertEvent, RadarPoint } from "./types";
import { scoreToLevel, formatUsd } from "@/lib/format";
import type { ScoreKey } from "@/lib/token/score-defs";
import type { RiskSeverity } from "@/lib/format";
import type { ScoreDetail, RiskItem, RiskGroups, SeriesPoint, TokenDetail, EventType, FeedEvent } from "./types";
import { EVENT_SEVERITY } from "@/lib/feed/event-defs";

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

const LAUNCHPADS = ["Pump.fun", "Raydium", "Moonshot", "Meteora"];
const DEXES = ["Raydium", "Meteora", "Orca", "Jupiter"];
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

export const mockApi: SentinelApi = {
  getKpis: () => delay(kpis),
  getTokens: () => delay(tokens),
  getAlerts: () => delay(alerts),
  getRadar: () => delay(radarFrom(tokens)),
  getEvents: () => delay(feedEvents),

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
