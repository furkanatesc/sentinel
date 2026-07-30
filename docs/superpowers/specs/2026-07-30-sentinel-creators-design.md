# SENTINEL Frontend — Increment 5 Design Spec
### Creators (Creator Profile + Liste)

- **Tarih:** 2026-07-30
- **Durum:** Onay bekliyor
- **Önkoşul:** Increment 1–4 master'da.
- **Kaynaklar:** `docs/design/sentinel-ui-ux-design.md` (Ekran 4), `ROADMAP.md` (§3 creator trust score), `docs/progress.md`.

---

## 1. Amaç ve kapsam

Bir deployer/creator cüzdanının derin analiz ekranı: geçmiş token performansı, reputation skoru, davranış paterni. Bu artımda:
- **`/creators/[address]`** tam profil (header + reputation + 8 metrik + token geçmişi tablosu + davranış paterni).
- **`/creators`** basit creator listesi (tablo → profile linki).
- **Wallet Graph** creator node detay paneline "Creator Detayına Git" linki.

Ağır görselleştirme yok — tablo + metrik + (yeniden kullanılan) skor bileşenleri. Backend gelince aynı seam gerçek API'ye geçer.

---

## 2. Clean Code & SOLID + yeniden kullanım (ölçüt)

- **DRY / mevcut bileşenleri kullan:** Reputation skoru için **Token Detail'in `ScoreCard` + `ExplainableScore`** bileşenleri yeniden kullanılır (reputation = tek `ScoreDetail`, key `creatorReputation` — `SCORE_DEFS`'te var). Metrikler için `MetricTile` yeniden kullanılır (bu artımda `components/token/MetricTile` → **`components/sentinel/MetricTile`** taşınır, paylaşılan primitive; OverviewTab importu güncellenir). `WalletAddress`, `ScoreBadge`, `formatUsd/formatPct`, `riskMeta` reuse.
- **SRP:** `CreatorHeader`, `CreatorMetrics`, `CreatorTokenHistoryTable`, `CreatorBehaviorPanel`, `CreatorsList` ayrı; hesap/türetme `lib/`'te (mock) veya reuse edilen saf fonksiyonlarda.
- **OCP:** outcome/liquidity durum rozetleri config'ten (`OUTCOME_DEFS`); metrik listesi bir config dizisinden map'lenir.
- **DIP:** bileşenler `useCreators`/`useCreator`/`getApi()`; mock import etmez.
- **ISP:** `CreatorTokenHistoryTable` yalnız `history`; `CreatorBehaviorPanel` yalnız `behavior`; dar prop'lar.

---

## 3. Mimari

### 3.1 Veri seam genişlemesi (`lib/api`)
```ts
export type CreatorOutcome = "active" | "graduated" | "dumped" | "rug" | "dead";
export type LiquidityStatus = "locked" | "unlocked" | "removed";

export interface CreatorRow {
  address: string; label?: string;
  reputationScore: number; riskLevel: RiskLevel;
  totalTokens: number; activeTokens: number; ruggedTokens: number;
  successRatePct: number; realizedPnlSol: number;
}
export interface CreatorTokenHistoryItem {
  id: string; symbol: string; mint: string; createdAt: string;
  peakMarketCap: number; currentMarketCap: number; maxDrawdownPct: number;
  liquidityStatus: LiquidityStatus; creatorSellPct: number;
  outcome: CreatorOutcome; riskFlags: string[];
}
export interface CreatorBehavior {
  deployFrequency: string;        // "12 token / 30 gün"
  avgFirstSellMinutes: number;
  repeatedFunders: string[];      // kısaltılmış adresler
  similarMetadata: boolean; sameSocial: boolean; sameLiquidityPattern: boolean;
}
export interface CreatorMetrics {
  totalTokens: number; activeTokens: number; ruggedTokens: number;
  avgLifetimeHours: number; avgPeakMarketCap: number;
  realizedPnlSol: number; successRatePct: number; avgFirstSellMinutes: number;
}
export interface CreatorProfile {
  address: string; label?: string; walletAgeDays: number; firstSeen: string;
  reputation: ScoreDetail;        // reuse; key = "creatorReputation"
  riskLevel: RiskLevel;
  metrics: CreatorMetrics;
  history: CreatorTokenHistoryItem[];
  behavior: CreatorBehavior;
}
```
- `SentinelApi.getCreators(): Promise<CreatorRow[]>` (mock ~6-8 creator).
- `SentinelApi.getCreator(address: string): Promise<CreatorProfile>` (mock: adresten deterministik üretir; reputation `ScoreDetail` breakdown'lı; history ~5-8 token).
- `httpApi` → `notReady`. `qk.creators`, `qk.creator(address)`; `useCreators()`, `useCreator(address)`.

### 3.2 Config (OCP)
`lib/creator/outcome-defs.ts`:
```ts
export const OUTCOME_DEFS: Record<CreatorOutcome, { label: string; color: string }>;   // active/graduated/dumped/rug/dead
export const LIQUIDITY_DEFS: Record<LiquidityStatus, { label: string; color: string }>; // locked/unlocked/removed
```

### 3.3 Bileşenler (`components/creator/`)
```
components/sentinel/MetricTile.tsx     # token/'den TAŞINIR (paylaşılan primitive)
components/creator/
├─ CreatorHeader.tsx        # adres(WalletAddress) + wallet yaşı + ilk görülme + risk + Watch/Telegram (toast)
├─ CreatorMetrics.tsx       # 8 MetricTile (config-driven)
├─ CreatorTokenHistoryTable.tsx  # token geçmişi tablosu; outcome/liquidity rozetleri; her token → /tokens/[symbol]
├─ CreatorBehaviorPanel.tsx # davranış paterni (frekans, ilk satış, funder'lar, benzerlik flag'leri)
├─ CreatorsList.tsx         # /creators tablosu → profile linkleri
└─ CreatorProfileContent.tsx # kompozisyon: header + Reputation(ScoreCard+ExplainableScore) + metrics + history + behavior; useCreator
app/(app)/creators/page.tsx           # server: getCreators prefetch + CreatorsList
app/(app)/creators/[address]/page.tsx # server: getCreator prefetch + CreatorProfileContent
```
- **CreatorProfileContent:** `useCreator(address)`; reputation için `SCORE_DEFS.find(d=>d.key==="creatorReputation")` def'i ile `<ScoreCard>` + `<ExplainableScore>` (reuse); loading (Skeleton)/error state.
- **CreatorsList:** `useCreators()`; tablo (adres, reputation ScoreBadge, toplam/aktif/rug token, başarı oranı, risk) → satır `/creators/[address]`.
- **Wallet Graph entegrasyonu:** `components/graph/NodeDetailPanel.tsx`'e — node creator tipindeyse (`creator_wallet`) "Creator Detayına Git" → `/creators/${node.address ?? node.id}`.

### 3.4 Sayfalar
- `/creators` (liste) ve `/creators/[address]` (profil) — server components, RSC prefetch + hydration. `/creators` placeholder değişir.

---

## 4. Kapsam dışı (bilinçli)
Creator karşılaştırma, funder cluster derin analizi (Wallet Graph'a bırakıldı), gerçek trade/watch persistanı, gerçek backend. Token Detail'in "Üretici" sekmesi hâlâ placeholder (bu artım standalone creator ekranı; sekme entegrasyonu sonraki artım). Smart-money/copy-trade.

## 5. Test stratejisi (TDD)
- `getCreators`/`getCreator` adapter (deterministik; reputation breakdown dolu; history bağlı); `useCreators`/`useCreator` hook.
- `outcome-defs`: her outcome/liquidity durumu tanımlı.
- `CreatorTokenHistoryTable`: satırlar + outcome rozeti + token linki (`/tokens/<symbol>`).
- `CreatorBehaviorPanel`: flag'ler + funder listesi render.
- `CreatorHeader`: adres/yaş/risk + Watch/Telegram toast.
- `CreatorMetrics`: 8 tile.
- `CreatorsList`: satır → `/creators/<address>` linki.
- `CreatorProfileContent`: reputation ScoreCard + ExplainableScore reuse render (smoke).
- MetricTile taşıma: OverviewTab importu + mevcut testler yeşil kalır.
- Wallet Graph `NodeDetailPanel`: creator node → "Creator Detayına Git" `/creators/<address>` (mevcut testi güncelle).

## 6. Kabul kriterleri
1. Sidebar "Üreticiler" → `/creators`: creator listesi; satıra tıkla → `/creators/[address]` profil.
2. Profil: header + Reputation skoru (ring + "neden bu skor?" breakdown, reuse) + 8 metrik + token geçmişi tablosu (outcome/liquidity rozetleri, token linkleri) + davranış paterni.
3. Wallet Graph'ta creator node seçilince "Creator Detayına Git" → creator profili.
4. Token geçmişindeki her token → Token Detail'e gider.
5. Tüm veri `component → useCreators/useCreator → getApi() → mock`; hiçbir bileşen mock import etmez.
6. `MetricTile` paylaşılan konuma taşındı; Overview hâlâ çalışır (testler yeşil).
7. Testler yeşil; build başarılı; SOLID/clean + reuse ölçütü.
