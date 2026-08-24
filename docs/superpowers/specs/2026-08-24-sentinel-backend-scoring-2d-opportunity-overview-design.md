# SENTINEL Backend — Alt-proje 2, Slice 2d: Opportunity + Overview (Tasarım)

**Tarih:** 2026-08-24
**Kapsam:** Alt-proje 2 (Scoring & Graph) → **2d (Opportunity + Overview)**
**Önceki dilimler:** 2a (Token Safety), 2b-1 (Creator Capture), 2b-2a (Token Outcome), 2b-2b (Creator Reputation), 2c (Manipulation Risk) — hepsi MERGED + canlı.
**Bağımlılık:** Yalnız zaten persist edilmiş skorlar (safety/creator-reputation/manipulation/momentum). **Yeni RPC/ağ yükü YOK, yeni key/ücret YOK.**

---

## 1. Amaç

Alt-proje 2'nin son skor-birleştirme dilimi. İki şey:

1. **Kompozit `opportunity` skoru** — token'ın diğer üç skorunu (+ momentum) **confidence-ağırlıklı** birleştirip 0-100 açıklanabilir bir "fırsat" skoru üretir. Şu nötr alanları gerçeğe döndürür:
   - `TokenDetail.scores.opportunity`: `value`/`confidence`/`updatedAt`/`breakdown` (şu an `neutralScores()`)
   - `TokenRow.signal`: `"buy"|"watch"|"avoid"|null` (şu an backend `null` döndürüyor)
2. **Overview ekranı** — iki yeni endpoint gerçeğe döner:
   - `getKpis()` → `Kpi[]` (Overview KPI kartları)
   - `getRadar()` → `RadarPoint[]` (Overview radar scatter)

**Frontend'e minimum dokunuş** — `getKpis`/`getRadar` gerçek fetch'e alınır + `LIVE_ENDPOINTS`+2; `TokenRow.signal` + `TokenDetail.scores.opportunity` seam zaten taşıyor. **UI bileşenleri DEĞİŞMEZ** (1c/2b-1 deseni).

**`opportunity` higherIsBetter:true** (score-defs.ts) → yüksek = daha iyi fırsat. `ScoreDetail`'de ayrı `riskLevel` string YOK; client `scoreDisplayLevel`'ı `value`'dan türetir.

---

## 2. Kapsam kararları (brainstorming'de onaylı)

### 2.1 Option A — saf-DB kompozit, kural-tabanlı Go
2a/2b/2c ile aynı **Option A** deseni: arka plan worker hesaplar + persist eder, okuma yolları (liste/detay/kpi/radar) DB'den okur, **canlı RPC/HTTP çağrısı YOK** → throttle-dayanıklı. opportunity **saf DB fonksiyonu** — zaten persist edilmiş skorları birleştirir, sıfır ek yük. Python DEĞİL (Go kural-tabanlı precedent).

### 2.2 Confidence-ağırlıklı kompozit (onaylı felsefe)
Girdilerin çoğu şu an degraded/seyrek (safety conf 0.5 çünkü holders DAS ertelendi; creator reputation seyrek çünkü WS dormant; manipulation txn-verisine bağlı). Kompozit bu gerçeği **dürüstçe taşır**: her girdi kendi confidence'ıyla ağırlıklanır, genel opportunity confidence girdi conf'larından türer. Girdi zayıfsa opportunity **düşük-conf** olur (nötr-erken-dönüş: tüm conf=0 ise opportunity nötr — **sahte skor yok**).

### 2.3 KPI: türetilebilir gerçek + trading/ops nötr placeholder (onaylı)
DB'den türetilebilen 4 KPI **gerçek**; trading (Alt-proje 5) + ops KPI'ları açıkça **nötr placeholder** (`value:"—"`, `tone:"neutral"`, `change:0`, `spark:[]`). 8 kart korunur (Overview görsel tam), ertelenen açık işaretli. **Sahte trading verisi ÜRETİLMEZ.**

### 2.4 getRadar: saf projeksiyon
Mevcut token skorlarının görsel izdüşümü — yeni veri toplamaz. Mock `radarFrom` birebir kopyalanır (aşağıda kesin).

### 2.5 Ertelenen alanlar (açık işaretli → followups)
- Trading/ops KPI'ları (Açık Pozisyonlar, Gerçekleşen/Gerçekleşmemiş K/Z, Sistem Gecikmesi) → Alt-proje 5 / ops telemetri.
- KPI `change` + `spark` tarihsel serisi (trend) → şimdilik gerçek KPI'larda `change:0`/`spark:[]` (anlık sayım gerçek, trend verisi ayrı bir zaman-serisi işi).
- Radar zaman-serisi / animasyon.

### 2.6 Dürüstlük notu: opportunity bir TÜREV
opportunity kendi başına yeni bir ölçüm yapmaz — üç skorun + momentum'un ağırlıklı birleşimidir. Girdiler iyileştikçe (holders DAS gelince safety conf→1.0; WS gelince creator dolunca) opportunity otomatik keskinleşir. Bu bilinçli: kompozit lens, alt-skorların kalitesini miras alır.

---

## 3. Opportunity formülü (kesin)

Dört girdi, hepsi **opportunity yönünde** (yüksek = daha iyi fırsat) normalize:

| Girdi | Ham değer (0-100) | Confidence | Ağırlık wᵢ |
|---|---|---|---|
| tokenSafety | `safety_score` (yüksek-iyi) | `safety_confidence` | 0.30 |
| creatorReputation | `creators.reputation_score` (yüksek-iyi, JOIN) | `creators.confidence` | 0.25 |
| manipülasyon-ters | `100 − manipulation_score` (manip yüksek-kötü → ters) | `manipulation_confidence` | 0.25 |
| momentum | `momentum` (yüksek-iyi) | market-proxy (bkz aşağı) | 0.20 |

**Momentum confidence proxy:** momentum kolonu default 0'dır ve enrichment yapılmamışsa 0 kalır (bu "çok ayı" DEĞİL, "bilinmiyor"). Bu yüzden `momentumConf = 1.0 if liquidity > 0 else 0.0` (enriched proxy) — enrichment yoksa momentum katkı vermez, opportunity'yi haksız düşürmez.

**Kompozit (confidence-ağırlıklı):**
```
W  = Σ (wᵢ × confᵢ)              // etkin ağırlık toplamı
opportunity            = W > 0 ? Σ(scoreᵢ × wᵢ × confᵢ) / W : 0
opportunityConfidence  = W / Σ(wᵢ)      // = W / 1.00 (ağırlıklar toplamı 1.0)
```
- **Nötr-erken-dönüş:** `W == 0` (tüm girdi conf=0) → `value=0, confidence=0, breakdown=[]` (2a/2c deseni — sahte skor yok, "henüz yok").
- **Clamp:** `value` = `clamp(round(...), 0, 100)` (girdiler zaten 0-100 olduğu için taşma olmaz; defansif).
- **Ağırlıklar kalibrasyon** — config'te sabit değil, kod-sabiti (YAGNI; ileride config'e alınabilir). Toplamları 1.00.

**Breakdown** (her katkıda bulunan girdi için bir kalem; conf=0 girdiler atlanır):
```
{ label: "Token güvenliği", weight: <katkı puanı>, detail: "72/100 (conf %50)" }
{ label: "Üretici itibarı",  weight: <katkı puanı>, detail: "60/100 (conf %100)" }
{ label: "Manipülasyon (ters)", weight: <katkı>, detail: "100−15=85 (conf %80)" }
{ label: "Momentum",         weight: <katkı>, detail: "63/100" }
```
`weight` = o girdinin nihai skora katkısı (`scoreᵢ × wᵢ × confᵢ / W`, yuvarlanmış) — kullanıcı hangi girdinin skoru sürüklediğini görür.

---

## 4. Signal türetme (kesin)

`TokenRow.signal` (`"buy"|"watch"|"avoid"|null`), opportunity `value` + `confidence`'tan türer:

```
if opportunityConfidence < SIGNAL_MIN_CONF (0.25):   signal = null    // yetersiz veri, dürüst
else if value >= 70:                                  signal = "buy"
else if value >= 45:                                  signal = "watch"
else:                                                 signal = "avoid"
```
- Eşikler kod-sabiti (kalibrasyon; ağırlıklar gibi). `SIGNAL_MIN_CONF` düşük-conf token'ı `null` yapar → "buy" ancak yeterli veriyle verilir (dürüst, agresif değil). **DÜZELTME 2026-08-24:** min-conf 0.35→**0.25** — en ağır tek-girdi ağırlığı (safety 0.30) altında olmalı, aksi hiçbir tek-girdi token'ı sinyal alamaz; 0.25 ≥ bir confident girdi (ör. manip-only conf 0.25) sinyal verebilir, null gerçekten seyrek/kısmi-conf token'lar için.
- `last_signal` kolonu tokens'ta **YOKTU** (DÜZELTME 2026-08-24: 0002 tokens CREATE'inde yok; "last_signal" yalnız 0001'deki AYRI `strategies` tablosunda — farklı domain). → migration `0011` `tokens.last_signal TEXT NOT NULL DEFAULT ''` ekler; opportunity worker `last_signal`'ı yazar.

---

## 5. getKpis (kesin)

`Kpi[]` — 4 gerçek (DB agrega) + 4 nötr placeholder. Sıra mock ile aynı (Overview düzeni korunur):

| id | label | value | Kaynak |
|---|---|---|---|
| `detected` | Tespit Edilen Token (24s) | COUNT(first_seen_ts ≥ now−24h) | gerçek |
| `highconf` | Yüksek Güvenli Token | COUNT(safety_score ≥ 70 AND safety_confidence ≥ 0.5) | gerçek, tone:positive |
| `critical` | Kritik Risk Tespiti | COUNT( `(manipulation_score ≥ 70 AND manipulation_confidence ≥ 0.5)` **OR** `(safety_score ≤ 30 AND safety_confidence ≥ 0.5)` ) — parantezler SQL'de zorunlu | gerçek, tone:critical |
| `signals` | Aktif Sinyaller | COUNT(last_signal IN ('buy','watch')) | gerçek |
| `positions` | Açık Pozisyonlar | "—" | placeholder (A5), tone:neutral |
| `realized` | Gerçekleşen K/Z (24s) | "—" | placeholder (A5), tone:neutral |
| `unrealized` | Gerçekleşmemiş K/Z | "—" | placeholder (A5), tone:neutral |
| `latency` | Sistem Gecikmesi | "—" | placeholder (ops), tone:neutral |

- Gerçek KPI'larda `value` = sayının string'i (ör. `"184"`); `change:0`, `spark:[]`, `updated:<ISO now>` (trend ertelendi §2.5). Placeholder'larda `value:"—"`.
- **Eşikler** (70/0.5 vb.) kod-sabiti. `Kpis()` tek SQL agrega (birden çok COUNT, tek sorgu) döndürür → handler `Kpi[]`'e map eder.

---

## 6. getRadar (kesin — saf projeksiyon)

Mock `radarFrom` birebir, DB'deki `RecentTokens` çıktısından:
```
RadarPoint {
  x:     creatorScore    (= creators.reputation_score, JOIN)
  y:     momentum
  z:     liquidity
  name:  symbol
  level: scoreToLevel(round((creatorScore + safetyScore) / 2))   // backend Go karşılığı
}
```
- `level` (`RiskLevel`) backend Go'da hesaplanır (frontend `scoreToLevel` mantığının Go dengi — `internal/store` veya scorer'da küçük yardımcı). Eşikler frontend `format.ts` `scoreToLevel` ile birebir (kalibrasyon parity: low/medium/high/critical).
- `Radar()` mevcut `RecentTokens` verisini yeniden kullanır → ayrı sorgu gerekmez ama temizlik için store `Radar(limit)` metodu RecentTokens'ı çağırıp map eder (SRP; handler ince kalır).

---

## 7. Mimari

### 7.1 Yeni paket `internal/opportunity/`
- `scorer.go` — saf `Score(Inputs) Result` (SRP, DI yok, ağsız). `Inputs`: dört skor + conf + liquidity (momentum-proxy). `Result`: value/confidence/breakdown + türetilmiş `Signal`. **Ağırlıklar/eşikler paket-sabiti.**
- `signal.go` (veya scorer içinde) — `deriveSignal(value, conf) string`.
- `worker.go` — `Worker` (2c Worker deseni): `OpportunityScoreTargets(limit)` çeker → her birini `Score` → `UpdateOpportunity` persist. ctx-cancel, kısmi-hata izole, tik-özet log (observability deseni — 2a/PR#11 gibi `opportunity cycle` targets/scored). **Canlı çağrı YOK** (saf DB).
- `provider` GEREKMEZ — girdiler DB'den gelir (RPC yok), worker doğrudan store'dan okur.

### 7.2 Migration `0011` — tokens'a opportunity kolonları
```sql
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS opportunity_score      DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS opportunity_confidence DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS opportunity_breakdown  TEXT             NOT NULL DEFAULT '';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS opportunity_scored_ts  BIGINT           NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS last_signal            TEXT             NOT NULL DEFAULT ''; -- DÜZELTME: tokens'ta yoktu (0001 strategies'te var, farklı domain)
```
Down: dört kolonu DROP.

### 7.3 Store metotları + tipler
- `OpportunityScoreTargets(ctx, limit) []OpportunityTarget` — tokens LEFT JOIN creators: `safety_score, safety_confidence, reputation_score+confidence (JOIN), manipulation_score+confidence, momentum, liquidity, mint`. (RecentTokens JOIN desenini yeniden kullanır.)
- `UpdateOpportunity(ctx, mint, score, confidence, breakdownJSON, signal, scoredTs)` — dört kolon + `last_signal` UPDATE.
- `Kpis(ctx) KpiCounts` — tek agrega sorgu (detected/highconf/critical/signals COUNT'ları).
- `Radar(ctx, limit) []RadarPoint` — RecentTokens map (Go `level` hesabı).
- `TokenDetailBase` + `RecentTokens`: opportunity_score/confidence/breakdown + last_signal okur → `TokenDetail.scores.opportunity` doldurulur, `TokenRow.signal` set edilir.
- **Fake + Postgres parity** (2a/2b deseni): fake store aynı JOIN/agrega semantiğini taklit eder.

### 7.4 Detail/liste bağlama
- `market/detail.go` `Build`: `scores.opportunity`'yi `neutralScores()` yerine base'den (opportunity_* kolonları) doldurur; breakdown JSON parse (2c manipulation_breakdown deseni).
- `RecentTokens` → `TokenRow.signal = last_signal` ("" → `null`; frontend `null` bekliyor).

### 7.5 API handlers
- `GET /api/kpis` → `store.Kpis` → `Kpi[]` (nil-guard, err→502). Placeholder'ları handler ekler (statik, deterministik).
- `GET /api/radar` → `store.Radar` → `RadarPoint[]` (nil-guard, err→502).
- RouterDeps/Bundle/main wiring (2b-1 creators handler deseni).

### 7.6 Config + main wiring
- `OPPORTUNITY_ENABLED` (default true), `OPPORTUNITY_INTERVAL_SEC` (default 60), `OPPORTUNITY_LIMIT` (default 100).
- main.go: `if cfg.OpportunityEnabled && bundle.Tokens != nil` → worker goroutine (Helius/RPC GEREKMEZ — saf DB, reputation/outcome worker gibi koşulsuz-RPC).

### 7.7 Frontend
- `http.ts`: `getKpis`/`getRadar` gerçek fetch (`notReady` yerine).
- `live-endpoints.ts`: `LIVE_ENDPOINTS`+2 (`getKpis`, `getRadar`).
- `TokenRow.signal` + `TokenDetail.scores.opportunity` zaten seam'de → ekstra frontend dosyası yok. **UI bileşeni dokunulmaz.** (3 dosya sınıfı: http.ts + live-endpoints.ts.)

---

## 8. Error handling

- Worker: `OpportunityScoreTargets` hata → tik atla + WARN (döngü ölmez). Per-target `UpdateOpportunity` hata → izole (o token atla, devam), tik-özet logda sayılır.
- Kısmi girdi: eksik skor conf=0 taşır (nötr-erken-dönüş formülde). Total-fail (W==0) → nötr persist (0/0) — bu **doğru**, gerçek "0 fırsat" değil "yeterli veri yok"; confidence=0 bunu taşır (frontend conf'a göre gösterir).
- API: store nil → 502 önce; boş sonuç → boş dizi (200, `[]`), UI çökmez.

---

## 9. Test stratejisi (TDD, SDD ile)

- **Scorer** (saf, ağsız): confidence-ağırlıklı kompozit doğru; nötr-erken-dönüş (tüm conf=0); manipülasyon-ters (yüksek manip → düşük katkı); momentum-proxy (liquidity=0 → momentum atlanır); clamp; breakdown katkı-doğruluğu.
- **Signal**: eşik sınırları (69/70, 44/45, min-conf altı → null).
- **Worker**: kısmi-hata izole; ctx-cancel; tik-özet.
- **Store agrega**: `Kpis` sayımları (fixture); `Radar` projeksiyon + `level` parity; `OpportunityScoreTargets` JOIN; **fake/postgres parity**.
- **API**: `/api/kpis` (4 gerçek + 4 placeholder şekli), `/api/radar` (nil-guard, boş→[]).
- **Frontend**: `getKpis`/`getRadar` gerçek fetch; `getApi()` hibrit (LIVE_ENDPOINTS); mevcut testler yeşil.
- `go build`/`vet`/`test -race ./...` + frontend suite.

---

## 10. Kapsam dışı (bilinçle ertelenen → followups)

- Trading/ops KPI'ları (positions/realized/unrealized/latency) → Alt-proje 5 + ops telemetri.
- KPI `change`/`spark` trend serisi → zaman-serisi altyapısı (ayrı iş).
- Radar zaman-serisi/animasyon.
- Opportunity ağırlık/eşiklerini config'e alma (şimdilik kod-sabiti, YAGNI).
- Girdi kalitesi: opportunity, alt-skorların degraded durumunu miras alır (safety conf 0.5 holders DAS'a kadar; creator seyrek WS'e kadar) — bunlar ayrı işler, 2d onları düzeltmez, dürüstçe taşır.

---

## 11. Sonraki dilimler

- **2e Wallet Graph** (`getWalletGraph`) — Alt-proje 2'nin son + en ağır dilimi (god node, cluster tespiti; muhtemel Python).
- Alt-proje 2 tamamlanınca: Alt-proje 3 (Alerts & Telegram) / 4 (Strategies & backtest) / 5 (Trading).
- Paralel açık iş: safety holders→DAS (Parça 2, opportunity safety-conf'unu 1.0'a çıkarır).
