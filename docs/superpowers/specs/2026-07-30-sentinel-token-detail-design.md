# SENTINEL Frontend — Increment 2 Design Spec
### Token Detail (Header + 4 Skor + Overview + Risk Analizi + Açıklanabilir Skor)

- **Tarih:** 2026-07-30
- **Durum:** Onay bekliyor
- **Önkoşul:** Increment 1 (shell + seam + Overview) master'da. Bu artım onun üzerine kurulur.
- **Kaynaklar:** `docs/design/sentinel-ui-ux-design.md` (Ekran 3), `ROADMAP.md` (skorlar, risk kategorileri), `docs/progress.md`.

---

## 1. Amaç ve kapsam

Kullanıcı feed'de bir tokena tıklayınca açılan, tokenı derinlemesine analiz eden ekran. Bu artımda:
**Header + 4 ana skor kartı + Overview sekmesi + Risk Analizi sekmesi + Açıklanabilir Skor paneli.** Grafikler **Recharts**. Kalan 8 sekme placeholder.

Rota: `app/(app)/tokens/[mint]/page.tsx`. Overview'daki **LiveTokenFeed satırları tıklanabilir** → bu ekran. Mock mint'ler kısaltılmış olduğundan mock `getToken(param)` param'ı **symbol** ile (case-insensitive) eşler; backend gelince param gerçek mint olur, seam değişmez.

---

## 2. Clean Code & SOLID ilkeleri (bu artımın ölçütü)

Kullanıcı talebi: clean code + SOLID'e sıkı bağlılık. Bu artımda somut karşılıkları:

- **SRP (Tek Sorumluluk):** Her bileşen tek iş yapar — `TokenHeader` kimlik+metrik, `ScoreCard` tek skor, `OverviewTab` grafikler+metrik tile, `RiskAnalysisTab` risk listeleri, `ExplainableScore` breakdown, `TokenTabs` sekme yönetimi. **Türetme/hesap mantığı bileşende değil** `lib/` içinde saf fonksiyonlarda (ör. skor seviyesini/rengini hesaplama, seri formatlama).
- **OCP (Açık/Kapalı):** Skorlar tek tek elle yazılmaz; `SCORE_DEFS` config dizisi üzerinden map'lenir. Yeni skor eklemek = diziye satır eklemek (bileşen değişmez). Sekmeler `TAB_DEFS` registry'sinden gelir; yeni sekme = registry'ye entry. Risk kategorileri de config'ten map'lenir.
- **DIP (Bağımlılığın Tersine Çevrilmesi):** Bileşenler somut `mock`'a değil `SentinelApi` soyutlamasına, `useToken()` hook'u üzerinden bağımlı. **Hiçbir bileşen `lib/api/mock`'u import etmez.** Backend gelince yalnız `http.ts` değişir.
- **ISP (Arayüz Ayrımı):** `ScoreCard` bütün `TokenDetail`'i değil dar `ScoreDetail`'i alır; `RiskAnalysisTab` yalnız `RiskGroups` alır; her tab yalnız ihtiyacı olan veriyi alır. Şişkin prop yok.
- **LSP:** `mockApi` ↔ `httpApi` `SentinelApi`'yi ikame edebilir (kontrat testiyle korunur).
- **Clean code:** küçük odaklı dosyalar (bir dosya büyürse böl), anlamlı isimler, DRY (ama erken soyutlama yok), pristine test çıktısı, TDD.

---

## 3. Mimari

### 3.1 Veri seam genişlemesi (`lib/api`)
Yeni tipler `lib/api/types.ts`'e; `getToken` `SentinelApi`'ye eklenir; mock `lib/api/mock.ts`'te üretir.

```ts
export type ScoreKey = "opportunity" | "creatorReputation" | "tokenSafety" | "manipulationRisk";

export interface ScoreBreakdownItem { label: string; weight: number; detail: string; }
export interface ScoreDetail {
  key: ScoreKey;
  value: number;        // 0-100
  confidence: number;   // 0-100
  updatedAt: string;
  breakdown: ScoreBreakdownItem[];
}

export type RiskSeverity = "critical" | "high" | "medium" | "info";
export interface RiskItem {
  id: string; title: string; severity: RiskSeverity;
  description: string; evidence?: string; firstSeen: string; lastSeen: string;
}
export interface RiskGroups { contract: RiskItem[]; market: RiskItem[]; creator: RiskItem[]; }

export interface SeriesPoint { t: number; v: number; }
export interface TokenMetrics {
  holders: number; uniqueBuyers: number; buyRatio: number; sellRatio: number;
  creatorHoldingPct: number; top10HolderPct: number; sniperPct: number; botActivityPct: number;
}
export interface TokenDetail {
  id: string; name: string; symbol: string; mint: string;
  ageSeconds: number; price: number; priceChange24h: number;
  marketCap: number; liquidity: number; volume24h: number;
  scores: Record<ScoreKey, ScoreDetail>;
  metrics: TokenMetrics;
  series: { price: SeriesPoint[]; liquidity: SeriesPoint[]; volume: SeriesPoint[]; holders: SeriesPoint[] };
  risks: RiskGroups;
}
```

- `SentinelApi.getToken(idOrMint: string): Promise<TokenDetail>` — mock mevcut `tokens` seed'inden **deterministik** üretir (symbol/mint eşleme; skorlar mevcut creator/safety + türetme; seriler `spark`-benzeri seed; riskler skora göre kurallı). Bulunamazsa reject.
- Hook: `lib/hooks/queries.ts`'e `useToken(mint)`; `qk.token = (mint) => ["token", mint]`.
- `httpApi.getToken` → `notReady` stub (mevcut desen).

### 3.2 Saf mantık (`lib/`) — SRP/clean
- `lib/format.ts`'e: `riskSeverityMeta: Record<RiskSeverity, {label,color,bg,border}>` (mevcut paletle DRY), `formatPct`, `formatCompactUsd` (varsa `formatUsd` yeniden kullan).
- `lib/token/score-defs.ts`: `SCORE_DEFS: { key: ScoreKey; label: string; higherIsBetter: boolean }[]`
  - opportunity (better), creatorReputation (better), tokenSafety (better), **manipulationRisk (worse)**.
  - **Önemli semantik:** Manipulation Risk'te yüksek = kötü. `ScoreCard` rengi `higherIsBetter ? scoreToLevel(value) : scoreToLevel(100 - value)` ile hesaplar (ters çevirme config'te, bileşende hardcode değil).

### 3.3 Bileşen yapısı (`components/token/`)
Her dosya tek sorumluluk:
```
components/token/
├─ TokenHeader.tsx          # kimlik + metrikler + aksiyon butonları (UI+toast)
├─ TokenActions.tsx         # İzle/Telegram/Simüle/Al/Sat buton grubu (Header'dan ayrık)
├─ ScoreCard.tsx            # tek skor: ring/meter + confidence + "Neden bu skor?"
├─ ScoreRow.tsx             # SCORE_DEFS üzerinden 4 ScoreCard map'ler (OCP)
├─ ExplainableScore.tsx     # seçili skorun breakdown paneli
├─ TokenTabs.tsx            # TAB_DEFS registry; Overview+Risk dolu, gerisi placeholder
├─ tabs/OverviewTab.tsx     # Recharts grafikler + metrik tile'ları
├─ tabs/RiskAnalysisTab.tsx # RiskGroups → kategorili liste
├─ metric-tile.tsx          # küçük metrik gösterimi (reusable)
└─ tab-defs.ts              # TAB_DEFS: { key,label,build? } (OCP registry)
```
- `app/(app)/tokens/[mint]/page.tsx` — server component: `getToken` prefetch + `dehydrate` + `HydrationBoundary`, `<TokenDetailContent mint=.../>` (client) render eder.
- `TokenDetailContent.tsx` — `useToken(mint)` + kompozisyon (Header + ScoreRow + TokenTabs + seçili skora göre ExplainableScore). Loading/empty state (shadcn Skeleton).

### 3.4 Feed → Detail bağlama
- `LiveTokenFeed` satırındaki token hücresi/Analyze butonu `next/link` ile `/tokens/${symbol}`'e gider. (Analyze artık no-op toast değil, navigasyon.)

---

## 4. UI davranışı

- **Header:** avatar, ad + symbol, mint (`WalletAddress` — copy), yaş (`formatAge`), fiyat + %değişim (yeşil/kırmızı), market cap / likidite / 24s hacim (mono). Aksiyonlar: **İzle, Telegram Alarmı, Simüle Et, Al, Sat**. Al/Sat/Simüle sadece UI + toast + (Live modda) uyarı — **gerçek trade yok** (güvenlik UX'i, ROADMAP §7).
- **ScoreRow (4 kart):** her kart 0–100 skor + seviye etiketi (renk+metin), progress ring veya yatay meter, confidence, son hesaplama zamanı, "Neden bu skor?" → `ExplainableScore`'u o skora ayarlar. Manipulation Risk rengi ters (yüksek = kritik).
- **ExplainableScore:** seçili skorun ağırlıklı, kanıta dayalı breakdown'ı (ROADMAP örneği: "Creator 27/100 — son 12 tokenın 8'i…"). Yalnız renk değil metin.
- **TokenTabs:** Overview (varsayılan aktif) + Risk Analizi dolu; diğerleri "yakında" placeholder içerik.
- **OverviewTab:** Recharts area/line — fiyat, likidite, hacim, holder büyümesi; metrik tile'ları — unique buyer, buy/sell oranı, creator holding %, top10 holder %, sniper %, bot activity %.
- **RiskAnalysisTab:** Contract / Market / Creator kategorileri; her risk: `riskSeverityMeta` badge + başlık + açıklama + kanıt + ilk/son görülme. Boşsa empty state.

---

## 5. Kapsam dışı (bilinçli — sessiz düşürme yok)
Market, Holders, Creator, Wallet Graph, Transactions, Social, Strategy Signals, Audit Log sekmeleri → **placeholder** (sonraki artımlar). Gerçek trade/simülasyon motoru, lightweight-charts (Trading Terminal artımı), auth/wallet-connect, `/tokens` liste ekranı (hâlâ placeholder).

## 6. Test stratejisi (TDD)
- `mockApi.getToken` kontratı: bilinen symbol için `TokenDetail` döndürür (4 skor, seriler dolu, riskler gruplu); bilinmeyen için reject. `getApi().getToken` seam.
- `useToken` hook mock veriyi yükler.
- `lib/format.ts`: `riskSeverityMeta` tam; `formatPct`.
- `score-defs`: manipulationRisk `higherIsBetter=false`; `ScoreCard` yüksek manipulation'ı kritik renkte gösterir (ters mantık testi).
- `RiskAnalysisTab`: kategori + severity render; empty state.
- `ScoreRow`: 4 kart SCORE_DEFS'ten render.
- Feed→detail: token linki doğru `/tokens/<symbol>` href'i.

## 7. Kabul kriterleri
1. Feed'de bir tokena tıklayınca `/tokens/<symbol>` açılır; SSR prefetch + hydration.
2. Header tüm kimlik/metrikleri + aksiyon butonlarını gösterir (Al/Sat toast + Live uyarısı).
3. 4 skor kartı doğru renk/seviye (Manipulation Risk ters), confidence, "Neden bu skor?" ExplainableScore'u besler.
4. Overview sekmesi: 4 grafik + 6 metrik tile mock veriyle.
5. Risk Analizi sekmesi: 3 kategori, severity'li kalemler; boş kategori empty state.
6. Diğer 8 sekme placeholder; tab navigasyonu çalışır.
7. Tüm veri `component → useToken → getApi() → mock`; hiçbir bileşen mock import etmez.
8. Testler yeşil; build başarılı; SOLID/clean-code review ölçütü karşılanır.
