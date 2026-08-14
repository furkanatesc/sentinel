# SENTINEL Backend — Alt-proje 2, Slice 2c: Manipulation Risk (Tasarım)

**Tarih:** 2026-08-14
**Kapsam:** Alt-proje 2 (Scoring & Graph) → **2c (Manipulation Risk)**
**Önceki dilimler:** 2a (Token Safety), 2b-1 (Creator Capture), 2b-2a (Token Outcome), 2b-2b (Creator Reputation) — hepsi MERGED + canlı.
**Bağımlılık:** 1b/1c Enricher'ın çektiği GeckoTerminal havuz verisi + 2a Safety worker'ın çektiği holder dağılımı (owner→amount). **Yeni RPC/ağ yükü yok.**

---

## 1. Amaç

`manipulationRisk` skorunu **saf-DB, kural-tabanlı, agrega-proxy** yaklaşımıyla hesaplamak: token'ın **işlem-akışı davranışından** (alım/satım dengesizliği, işlem/benzersiz-alıcı yoğunluğu, hacim/likidite anomalisi, creator payı) 0-100 **açıklanabilir** bir manipülasyon riski üretmek ve Token Detay ekranındaki şu nötr alanları gerçeğe döndürmek:

- `TokenDetail.scores.manipulationRisk`: `value`/`confidence`/`updatedAt`/`breakdown` (şu an `neutralScores()` → 0/0/"—"/[])
- `TokenDetail.metrics`: `uniqueBuyers`, `buyRatio`, `sellRatio`, `creatorHoldingPct` (şu an hepsi 0; yalnız `top10HolderPct`+`holders` doluydu)

**Frontend'e sıfır dokunuş** — seam (`store.TokenMetrics` + `ScoreDetail`) zaten bu alanları taşıyor; yalnız backend gerçeğe çevirir. **`TokenRow` (liste) DEĞİŞMEZ** — akış yalnız `TokenDetail`/`TokenDetailBase` üzerinden.

**`manipulationRisk` inverted'tır:** frontend `higherIsBetter:false` (score-defs.ts) → yüksek değer = daha çok manipülasyon = **kötü**. `ScoreDetail`'de `riskLevel` string YOKTUR; client `scoreDisplayLevel`'ı `100−value` ile türetir. Bu yüzden 2c yalnız `value`/`confidence`/`breakdown` üretir (2b-2b'deki gibi ayrı `riskLevel` alanı YOK).

---

## 2. Kapsam kararları (brainstorming'de onaylı)

### 2.1 Option A — agrega-tabanlı proxy, saf kural-tabanlı Go
2a Safety / 2b Reputation ile aynı **Option A** deseni: arka plan worker hesaplar + persist eder, okuma yolları DB'den okur, **canlı RPC/HTTP çağrısı yok** → throttle-dayanıklı (Helius/public-RPC + GeckoTerminal paylaşımlı-IP limitlerini tekrar yüklemez). Python DEĞİL — 2a/2b'de kurulan Go kural-tabanlı precedent (ölçülü `apps/scoring-py` YOK).

### 2.2 Trade-davranışı ODAKLI skor (kompozit değil)
Skor yalnız **işlem-akışı** sinyallerinden oluşur. Holder yoğunlaşması (`top10HolderPct`) bilinçle **dışarıda** — o zaten `tokenSafety` (2a) skorunun bir girdisidir; manipülasyona da katmak **çift-sayım** olurdu. `manipulationRisk` ayrı bir lens: "bu token'ın alım/satım aktivitesi sahte/şişirilmiş görünüyor mu?"

### 2.3 Veri kaynağı: mevcut çağrıların yan-ürünü (sıfır ek yük)
- **GeckoTerminal `transactions` objesi** (`attributes.transactions.h24.{buys,sells,buyers,sellers}`) API yanıtında **zaten var** ama `gtPool` struct'ında parse EDİLMİYOR. Enricher aynı `pools/multi` çağrısını yapıyor → ek istek yok, yalnız yeni alan parse+persist.
- **creator_holding_pct**, 2a Safety worker'ının **zaten yaptığı** `HolderDistribution` owner→amount fetch'inden türetilir (creator adresi verilirse aynı map'ten payı çıkar) → **sıfır ek RPC**. Bu bilinçli çapraz-slice dokunuş (kullanıcı 2026-08-14 onayladı).

### 2.4 Ertelenen alanlar (nötr kalır — sahte değil, "henüz yok")
Gerçek per-trade / per-wallet veri gerektiren alanlar framework-hazır bırakılır, 2e (Wallet Graph — trade-flow + funding-graph) doldurur:
- `TokenMetrics.sniperPct` → per-wallet ilk-blok alım verisi → **2e**
- `TokenMetrics.botActivityPct` → per-wallet davranış imzası → **2e**
- `CreatorTokenHistoryItem.creatorSellPct` → creator satış trade-flow'u → **2e**
- `CreatorBehavior.*` (deployFrequency / avgFirstSellMinutes / repeatedFunders / similarMetadata / sameSocial / sameLiquidityPattern) → funding-graph + trade-flow → **2e**

Bu alanlar 0/false/boş-dizi kalır (JSON non-nil). `followups-frontend.md` "2c" bölümüne açıkça yazılır (sessiz düşürme yok).

### 2.5 Dürüstlük notu: skor bir PROXY
`manipulationRisk` gerçek per-trade wash/sniper tespiti değil, **agrega davranış anomalisi**. Breakdown kalemleri bunu şeffaf açıklar (ör. "Alım/satım dengesizliği", "İşlem/benzersiz-alıcı oranı yüksek") ki UI kullanıcısı sinyalin doğasını görsün. 2e trade-flow geldiğinde skor gerçek-tespite yükseltilebilir (OCP — yeni Check eklenir).

---

## 3. Skor formülü (kesin)

**INVERTED — taban 0, red-flag'ler puan EKLER.** Her bileşen önce `[0,1]`'e normalize edilir, sonra ağırlıkla çarpılır; toplam `clamp[0,100]`.

Girdiler (hepsi DB'den, h24): `buys`, `sells`, `buyers` (benzersiz alıcı), `creatorHoldingPct` (%), `vol24h` (USD), `liquidity` (USD).

Türetilenler:
```
txns      = buys + sells
buyRatio  = buys / txns            (txns=0 → 0)
sellRatio = sells / txns           (txns=0 → 0)
```

**Nötr erken-dönüş:** `txns < MANIPULATION_MIN_TXNS` (default 20) → `value=0, confidence=0, breakdown=[]` (dürüst "yeterli aktivite yok"). Erken dönüş (2a/2b deseni).

**Bileşenler (`txns ≥ MIN_TXNS` ise), her biri `[0,1]`:**

1. **Dengesizlik** (tek-yönlü akış = şişirme/dump sinyali):
   `imbalanceNorm = clamp01( |buyRatio − 0.5| · 2 )` → dengede 0, hep-alım/hep-satım 1.

2. **Wash-proxy** (az cüzdan çok işlem = döngüsel/wash sinyali):
   `perBuyer = buys / max(buyers, 1)`
   `washNorm = clamp01( (perBuyer − WASH_MIN) / (WASH_MAX − WASH_MIN) )`
   defaults `WASH_MIN=3`, `WASH_MAX=15` (eşik altı → 0).

3. **Hacim/likidite anomalisi** (likiditeye göre aşırı devir = pump sinyali):
   `volLiq = vol24h / max(liquidity, 1)`
   `volNorm = clamp01( (volLiq − VOL_MIN) / (VOL_MAX − VOL_MIN) )`
   defaults `VOL_MIN=3`, `VOL_MAX=20`.

4. **Creator payı** (hafif — creator büyük pay tutuyorsa dump kaldıracı):
   `creatorNorm = clamp01( creatorHoldingPct / 100 )`. creator_holding bilinmiyorsa (0) → 0 katkı (dürüst).

**Skor:**
```
value = clamp(0, 100,
    W_imbalance · imbalanceNorm
  + W_wash      · washNorm
  + W_volume    · volNorm
  + W_creator   · creatorNorm )
```
Varsayılan ağırlıklar (config, deploy-tunable, tümü-maks = 100):
`W_imbalance=30`, `W_wash=35`, `W_volume=25`, `W_creator=10`.

**Confidence:** `min(1, txns / MANIPULATION_CONF_TXNS)`, default `CONF_TXNS=100`. `confidence == 0` ↔ `txns < MIN_TXNS`.

**Breakdown (şeffaf, `[]ScoreBreakdownItem` — 2a Safety gibi):** yalnız katkısı `> 0` olan bileşenler için birer kalem (`label`/`weight`=puan katkısı/`detail`=ham bağlam, ör. "buyRatio=0.87", "işlem/alıcı=8.2", "vol/likidite=12.4x", "creator payı=%31"). Türkçe açıklayıcı stringler.

---

## 4. Mimari

### 4.1 Yeni paket `internal/manipulation/`
- **`scorer.go` — saf `Scorer` (SRP, ağsız/DB'siz):** `Score(in Inputs) Result`.
  - `Inputs = {Buys, Sells, Buyers int; CreatorHoldingPct, Vol24h, Liquidity float64}`
  - `Result = {Value, Confidence float64; Breakdown []store.ScoreBreakdownItem}`
  - Ağırlıklar/eşikler `Thresholds` struct ile enjekte (config'ten). Saf → tam unit-test.
- **DIP arayüzleri (worker'ın bağımlılıkları):**
  - `ManipulationStore`: `ManipulationTargets(ctx, limit) ([]store.ManipulationTarget, error)` + `UpdateManipulation(ctx, store.ManipulationUpdate) error`. `store.TokenStore` (postgres) + fake karşılar.
- **`worker.go` — `Worker` (safety/outcome Worker deseni, RPC YOK):** ticker; her tik `ManipulationTargets(limit)` → her hedef için `Scorer.Score` → `UpdateManipulation`. Kısmi hata izole (bir token patlarsa diğerleri devam) + ctx-cancel. Log: `logger.Warn("manipulation", ...)`.

### 4.2 Enricher genişletmesi (GeckoTerminal transactions parse)
`geckoterminal.go`:
- `gtPool.Attributes`'a ekle:
  ```go
  Transactions map[string]struct {
      Buys    int `json:"buys"`
      Sells   int `json:"sells"`
      Buyers  int `json:"buyers"`
      Sellers int `json:"sellers"`
  } `json:"transactions"`
  ```
- `market.Pool`'a ekle: `TxnsBuys, TxnsSells, TxnsBuyers, TxnsSellers int`.
- `toPools`'ta `d.Attributes.Transactions["h24"]`'ten doldur (obje yoksa/null → 0, panik yok).

`enricher.go`:
- `store.MarketUpdate`'e ekle: `TxnsBuys, TxnsSells, TxnsBuyers, TxnsSellers int`.
- `tick`'te `UpdateMarket` çağrısına bu 4 alanı geçir (aynı `pools/multi` yanıtından; yeni istek yok).

> **`txns_sellers` notu:** `sellers` yakalanıp persist edilir (buyers ile simetrik) ama 2c Scorer'ı **tüketmez** (wash-proxy yalnız `buyers` kullanır). Bilinçli ileriye-hazırlık: satış-tarafı wash sinyali 2e adayı. Ölü-alan değil, framework-hazır (followups'ta işaretlenir).

### 4.3 Safety worker genişletmesi (creator_holding_pct — sıfır ek RPC)
Safety Scorer **DEĞİŞMEZ** (creator payı safety girdisi değil → çift-sayım yok). Yalnız holder-fetch yan-ürünü olarak creator payı hesaplanıp persist edilir:
- `store.SafetyTarget`'a ekle: `Creator string` (2b-1'in `tokens.creator` kolonundan; `SafetyScoreTargets` SELECT'ine eklenir).
- `safety.Holders` arayüzü + `HeliusHolders.HolderDistribution` imzası:
  ```
  HolderDistribution(ctx, mint, creator string, capN int) (count int, top10Pct, creatorPct float64, capped bool, err error)
  ```
  `byOwner[creator] / total · 100` (creator=="" veya haritada yok → 0). Tek impl + fake güncellenir.
- `safety.OnChainData`'ya ekle: `CreatorHoldingPct float64; CreatorHoldingKnown bool`. `FetchOnChain(ctx, mint, creator string)` creator alır, holder yolu başarılıysa doldurur.
- `store.SafetyUpdate`'e ekle: `CreatorHoldingPct float64; CreatorHoldingKnown bool`. `UpdateSafety` bu payı **yalnız `CreatorHoldingKnown` ise** yazar (aksi mevcut değeri EZMEZ — `creator_holding_pct = CASE WHEN $known THEN $pct ELSE creator_holding_pct END`). Ayrı yazma YOK (aynı satır/tik).

> **Sınır notu:** creator_holding yalnız safety-scored token'larda dolar. Bir token henüz safety-scored değilse `creator_holding_pct=0` → manipülasyon skorunda `creatorNorm=0` (dürüst nötr katkı). Kabul edilen kısmi-veri (2a/2b deseni).

### 4.4 Migration `0010` — tokens'a kolonlar
```sql
-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS txns_buys            INTEGER          NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS txns_sells           INTEGER          NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS txns_buyers          INTEGER          NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS txns_sellers         INTEGER          NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS creator_holding_pct  DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS manipulation_score      DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS manipulation_confidence DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS manipulation_breakdown  TEXT             NOT NULL DEFAULT '';  -- JSON []ScoreBreakdownItem
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS manipulation_scored_ts  BIGINT           NOT NULL DEFAULT 0;
-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS ...;  (9 kolon)
```

### 4.5 Store metotları + tipler
- **`ManipulationTarget`** = `{Mint string; Buys, Sells, Buyers int; CreatorHoldingPct, Vol24h, Liquidity float64}`.
- **`ManipulationTargets(ctx, limit)`** — round-robin (skorlanmamış/eskiden-yeni):
  ```sql
  SELECT mint, txns_buys, txns_sells, txns_buyers, creator_holding_pct, vol_24h, liquidity
  FROM tokens
  WHERE pool_address <> ''            -- yalnız enrichment görmüş token'lar
  ORDER BY manipulation_scored_ts ASC, first_seen_ts DESC
  LIMIT $1;
  ```
- **`ManipulationUpdate`** = `{Mint string; Score, Confidence float64; Breakdown []ScoreBreakdownItem; ScoredTs int64}`.
- **`UpdateManipulation(ctx, u)`** — `UPDATE tokens SET manipulation_score/confidence/breakdown(JSON)/scored_ts WHERE mint=$1`.
- **`UpdateMarket`** — mevcut; +4 txns kolonu yazar (4.2).
- **`UpdateSafety`** — mevcut; +creator_holding_pct koşullu yazar (4.3).
- **`TokenDetailBase`** (token_detail.go) — ekle: `ManipulationScore, ManipulationConfidence float64; ManipulationBreakdown []ScoreBreakdownItem; ManipulationScoredTs int64` + `TxnsBuys, TxnsSells, TxnsBuyers int; CreatorHoldingPct float64`. `TokenDetailBase` SELECT'i bu kolonları okur.

### 4.6 Detail bağlama (`market/detail.go` `Build`)
- `d.Scores["manipulationRisk"]` = `neutralScores()` yerine DB'den (creatorReputation deseni birebir):
  ```go
  updatedAt := "—"; if base.ManipulationScoredTs > 0 { updatedAt = time.Unix(...).Format(RFC3339) }
  d.Scores["manipulationRisk"] = store.ScoreDetail{Key:"manipulationRisk",
      Value: base.ManipulationScore, Confidence: base.ManipulationConfidence,
      UpdatedAt: updatedAt, Breakdown: base.ManipulationBreakdown}
  // nil breakdown → []ScoreBreakdownItem{} (mevcut guard deseni)
  ```
- `d.Metrics`'e ekle (base'den):
  ```go
  txns := base.TxnsBuys + base.TxnsSells
  d.Metrics.UniqueBuyers = base.TxnsBuyers
  if txns > 0 { d.Metrics.BuyRatio = float64(base.TxnsBuys)/float64(txns); d.Metrics.SellRatio = float64(base.TxnsSells)/float64(txns) }
  d.Metrics.CreatorHoldingPct = base.CreatorHoldingPct
  ```
  (`Holders` + `Top10HolderPct` zaten set; `sniperPct`/`botActivityPct` 0 kalır — 2e.)

### 4.7 Config + main wiring
Yeni env (hepsi default'lu, deploy-tunable):
| Env | Default | Anlam |
|---|---|---|
| `MANIPULATION_ENABLED` | true | Worker aç/kapa |
| `MANIPULATION_INTERVAL_SEC` | 60 | Döngü aralığı |
| `MANIPULATION_LIMIT` | 60 | Döngü başına skorlanan token |
| `MANIPULATION_MIN_TXNS` | 20 | Nötr eşiği (altı → skor yok) |
| `MANIPULATION_CONF_TXNS` | 100 | Tam güven için işlem sayısı |
| `MANIPULATION_W_IMBALANCE` | 30 | Alım/satım dengesizliği ağırlığı |
| `MANIPULATION_W_WASH` | 35 | Wash-proxy ağırlığı |
| `MANIPULATION_W_VOLUME` | 25 | Hacim/likidite anomali ağırlığı |
| `MANIPULATION_W_CREATOR` | 10 | Creator payı ağırlığı |
| `MANIPULATION_WASH_MIN` / `_MAX` | 3 / 15 | işlem/alıcı normalleştirme bandı |
| `MANIPULATION_VOL_MIN` / `_MAX` | 3 / 20 | vol/likidite normalleştirme bandı |

`main.go`: `MANIPULATION_ENABLED && bundle.Tokens != nil` gate (RPC/Helius **GEREKMEZ**) → `manipulation.Worker` goroutine (safety/outcome/reputation yanında).

---

## 5. Error handling

- **Worker:** `ManipulationTargets` hatası → döngü atla + `Warn("manipulation", "err", …)`; tek token `UpdateManipulation` hatası → izole (log + continue). ctx-cancel → temiz çıkış.
- **Enricher:** `transactions` objesi eksik/null → 0 (mevcut kısmi-başarı deseni; batch atlanmaz).
- **Safety:** creator=="" veya holder-fetch başarısız → `CreatorHoldingKnown=false` → mevcut creator_holding EZİLMEZ.
- **Okuma:** `TokenDetail` JOIN yok (kolonlar tokens'ta) → manipülasyon skorlanmamışsa `value/confidence=0`, `updatedAt="—"` (dürüst, hata değil). Handler 502/404 mevcut deseni korunur.

---

## 6. Test stratejisi (TDD, SDD ile)

1. **Saf `Scorer` unit:** her bileşeni izole eden fixture'lar — dengesizlik (hep-alım→yüksek, dengeli→0), wash-proxy (band altı=0 / üstü=1 / ara-orantı), vol/likidite (band), creator payı; `txns<MIN_TXNS`→nötr (value 0/conf 0/boş breakdown); conf ramp `min(1,txns/CONF_TXNS)`; clamp[0,100] sınırları; ağırlık/eşik config'ten enjekte → sınır-kayışı testleri; breakdown yalnız katkısı>0 bileşenleri içerir.
2. **Enricher GT parse:** kayıtlı GeckoTerminal JSON fixture ile `transactions.h24.{buys,sells,buyers,sellers}` parse; eksik/null `transactions` → dürüst 0 (panik yok); `Pool` alanları + `UpdateMarket` argümanları.
3. **Safety creator-holding:** `HolderDistribution` owner→amount'tan creator payı doğru (%); creator adresi listede yoksa / creator=="" → 0; `CreatorHoldingKnown` bayrağı; `UpdateSafety` koşullu-yazma (known=false → EZMEZ).
4. **Store parity (fake↔postgres):** `ManipulationTargets` (round-robin sıralama + `pool_address<>''` filtre), `UpdateManipulation` (breakdown JSON round-trip), `UpdateMarket` txns kolonları, `UpdateSafety` creator_holding koşullu, `TokenDetailBase` yeni alanlar.
5. **Worker:** kısmi-hata izolasyon (bir token patlar, diğerleri skorlanır) + ctx-cancel (2a/2b-2a deseni); **RPC yok → ağsız**.
6. **Detail bağlama:** `market/detail.go` `manipulationRisk` gerçek DB'den (nil-breakdown guard); `TokenMetrics.uniqueBuyers/buyRatio/sellRatio/creatorHoldingPct` doldurulur; `sniperPct/botActivityPct` 0 kalır.
7. **Frontend:** DEĞİŞMEZ — seam sabit; mevcut testler (`token.test.ts` vb.) regresyonsuz. Go build/vet/`test -race` yeşil.
8. **Kalibrasyon riski (deploy'da, 1a/2a deseni):** GeckoTerminal `transactions` alan-adı/şekli + eşik/ağırlık değerleri `MANIPULATION_*` ile deploy-tunable (kod değişmez).

---

## 7. Kapsam dışı (bilinçle ertelenen → followups)

- `TokenMetrics.sniperPct` / `botActivityPct` (per-wallet → 2e).
- `CreatorTokenHistoryItem.creatorSellPct` + `CreatorBehavior.*` (trade-flow / funding-graph → 2e).
- Gerçek per-trade wash tespiti (agrega proxy'nin yükseltmesi → 2e, OCP yeni Check).
- `transactions` m5/h1 pencereleri (yalnız h24 kullanılır; YAGNI).
- Presentation-layer "0 = bilinmiyor" formatlama (frontend followup).

---

## 8. Sonraki dilimler

2c → **2d** (Opportunity + Overview: `getKpis`/`getRadar`/`signal` — kompozit `opportunity` skoru diğer 3 skordan) → **2e** (Wallet Graph, god node — trade-flow + funding-graph; ertelenen sniper/bot/creatorSell/behavior gerçek verisi burada gelir).

Süreç: writing-plans → SDD (subagent-driven, per-task review + final opus whole-branch review) → push + PR + `gh pr merge --merge --delete-branch` (kullanıcı onayıyla).
