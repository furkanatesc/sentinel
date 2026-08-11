# SENTINEL Backend — Alt-proje 2, Slice 2b-2b: Creator Reputation Score (Tasarım)

**Tarih:** 2026-08-11
**Kapsam:** Alt-proje 2 (Scoring & Graph) → 2b (Creator Reputation) → **2b-2b (İtibar Skoru)**
**Önceki dilimler:** 2a (Token Safety), 2b-1 (Creator Capture), 2b-2a (Token Outcome Detection) — hepsi MERGED + canlı.
**Bağımlılık:** 2b-2a'nın ürettiği per-token `outcome`/`peak_market_cap`/`max_drawdown_pct`/`liquidity_status` + 2b-1'in yakaladığı `tokens.creator`.

---

## 1. Amaç

Creator (üretici) itibar skorunu **saf-DB, kural-tabanlı** hesaplamak: 2b-2a token outcome'larını creator başına agrega ederek 0-100 **açıklanabilir** bir `reputationScore` + `riskLevel` + `CreatorMetrics` üretmek ve şu nötr alanları gerçeğe döndürmek:

- `CreatorRow`: `reputationScore`, `riskLevel`, `activeTokens`, `ruggedTokens`, `successRatePct`
- `CreatorProfile`: `reputation` (value/confidence/breakdown), `riskLevel`, `metrics` (aşağıdaki alt küme)
- `CreatorTokenHistoryItem`: `riskFlags` (per-token)
- `TokenRow`: `creatorScore`
- `TokenDetail`: `scores.creatorReputation`

**Frontend'e sıfır dokunuş** — seam (`apps/web/lib/api/`) zaten bu alanları taşıyor; yalnız backend gerçeğe çevirir.

---

## 2. Kapsam kararları (brainstorming'de onaylı)

### 2.1 Saf-DB (yeni RPC YOK)
2b-2b yalnızca **DB'de mevcut** 2b-2a outcome verisinden hesaplar. 2a Safety'nin **Option A** deseni: arka plan worker hesaplar + persist eder, okuma yolları DB'den okur, **canlı RPC çağrısı yok** → throttle-dayanıklı. (Bu, yeni çözülen Helius/public-RPC `getSignaturesForAddress` blokörünü tekrar yüklememek için bilinçli karardır.)

### 2.2 Ertelenen alanlar (nötr kalır — sahte değil, "henüz yok")
- `walletAgeDays` → creator'ın en-eski tx'i (RPC `getSignaturesForAddress`) gerektirir → **ileri dilim** (RPC-zenginleştirme).
- `realizedPnlSol` + `avgFirstSellMinutes` → creator trade-flow persist gerektirir → **2c**.
- `CreatorBehavior` (deployFrequency/repeatedFunders/similarMetadata/…) → davranış analizi → **2c/2e**.
- `CreatorTokenHistoryItem.creatorSellPct` → trade-flow → **2c**.

Bu alanlar 0/false/boş-dizi kalır (JSON non-nil). Presentation-layer'da "0" gerçek sıfır gibi görünür — bu 2b-1'den beri var olan bir **followup** (backend 2b-2b kapsamı dışı).

---

## 3. Skor formülü (kesin)

Bir creator'ın tokenları üzerinden:

- `total` = toplam token
- `active`, `graduated`, `dumped`, `rug`, `dead` = outcome'a göre sayımlar (2b-2a'nın 5'li outcome'u)
- **`resolved = total − active`** (rug+dumped+dead+graduated). `active` = "henüz yargılanmadı", oran/güven hesabına girmez.

**resolved == 0 ise:** `confidence = 0` → **nötr itibar** (skor 0/nötr gösterilir, `riskLevel = "medium"`, breakdown boş). Erken dönüş (2a deseni).

**resolved > 0 ise oranlar (payda `resolved`):**
```
rugRate  = rug / resolved
failRate = (dumped + dead) / resolved
gradRate = graduated / resolved
```

**Skor:**
```
score = clamp(0, 100, 50 − W_rug·rugRate − W_fail·failRate + W_grad·gradRate)
```
Varsayılan ağırlıklar (config, deploy-tunable): `W_rug=50`, `W_fail=20`, `W_grad=40`.
Örnek: hepsi rug → 0; hepsi graduated → 90; hepsi dumped/dead → 30; karışım orantılı.
(Ayrı `avgPeakMarketCap` traction ödülü YOK — graduated zaten mcap≥69k ile tanımlı olduğundan çift-sayım olurdu; bilinçle elendi.)

**Confidence:** `min(1, resolved / K)`, `K = REPUTATION_MIN_RESOLVED` (default 5). 1 çözülmüş → 0.2; 5+ → 1.0. `confidence == 0` ↔ `resolved == 0`.

**Breakdown (şeffaf, `ScoreBreakdownItem` — 2a Safety gibi):** taban 50 + üç kalem ("Rug oranı −X", "Başarısız (dump/dead) −Y", "Graduated +Z"), her biri çözülmüş sayı bağlamıyla.

**riskLevel (frontend `scoreToLevel` bantları, Go'da yeniden):**
```
score ≤ 24 → critical
score ≤ 49 → high
score ≤ 69 → medium
score ≤ 84 → good
score > 84 → strong
```
`RiskLevel` = `critical|high|medium|good|strong` (frontend `lib/format.ts`); **"low" YOK**. `higherIsBetter=true` (score-defs.ts) ile uyumlu → yüksek itibar = düşük risk. Nötr durumda `"medium"`.

**successRatePct = gradRate · 100** (gerçek, resolved üzerinden).

---

## 4. Mimari

### 4.1 Yeni paket `internal/reputation/`
- **`scorer.go` — saf `Scorer` (SRP, ağsız/DB'siz):** `Score(agg CreatorAgg) Reputation`, `Reputation = {Score, Confidence, RiskLevel, Breakdown, ActiveTokens, RuggedTokens, GraduatedTokens, SuccessRatePct, AvgPeakMarketCap, AvgLifetimeHours}`. Eşikler/ağırlıklar `Thresholds` struct ile enjekte (config'ten). Saf → tam unit-test.
- **DIP arayüzleri:**
  - `ReputationDataProvider`: `CreatorAggregates(ctx, limit) ([]store.CreatorAgg, error)`
  - `ReputationStore`: `UpsertReputation(ctx, store.CreatorReputation) error`
  - Her ikisini de `store.TokenStore` (postgres) + fake karşılar.
- **`worker.go` — `Worker` (safety/outcome Worker deseni, RPC YOK):** ticker; her tik `CreatorAggregates(limit)` → her agrega için `Scorer.Score` → `UpsertReputation`. Kısmi hata izole (bir creator patlarsa diğerleri devam) + ctx-cancel. Log deseni: `"reputation"` warn.

### 4.2 Migration `0009` — yeni `creators` tablosu (türetilmiş agrega)
```sql
CREATE TABLE IF NOT EXISTS creators (
  address             TEXT PRIMARY KEY,
  reputation_score    DOUBLE PRECISION NOT NULL DEFAULT 0,
  confidence          DOUBLE PRECISION NOT NULL DEFAULT 0,
  risk_level          TEXT NOT NULL DEFAULT 'medium',
  breakdown           TEXT NOT NULL DEFAULT '',        -- JSON []ScoreBreakdownItem
  total_tokens        INTEGER NOT NULL DEFAULT 0,
  active_tokens       INTEGER NOT NULL DEFAULT 0,
  rugged_tokens       INTEGER NOT NULL DEFAULT 0,
  graduated_tokens    INTEGER NOT NULL DEFAULT 0,
  avg_peak_market_cap DOUBLE PRECISION NOT NULL DEFAULT 0,
  avg_lifetime_hours  DOUBLE PRECISION NOT NULL DEFAULT 0,
  success_rate_pct    DOUBLE PRECISION NOT NULL DEFAULT 0,
  scored_ts           BIGINT NOT NULL DEFAULT 0
);
```
`creators` = 2b-1'in "2b-2 için ekleyebilir" dediği hesaplanmış-skor tablosu. `tokens.creator` FK değil (soft; creator token'ları silinmez).

### 4.3 Store metotları
- **`CreatorAggregates(ctx, limit)` — tek SQL:**
  ```sql
  SELECT t.creator, COUNT(*) AS total,
    SUM(CASE WHEN t.outcome='active'    THEN 1 ELSE 0 END) AS active,
    SUM(CASE WHEN t.outcome='rug'       THEN 1 ELSE 0 END) AS rug,
    SUM(CASE WHEN t.outcome='dumped'    THEN 1 ELSE 0 END) AS dumped,
    SUM(CASE WHEN t.outcome='dead'      THEN 1 ELSE 0 END) AS dead,
    SUM(CASE WHEN t.outcome='graduated' THEN 1 ELSE 0 END) AS graduated,
    COALESCE(AVG(NULLIF(t.peak_market_cap,0)),0) AS avg_peak,
    COALESCE(AVG(CASE WHEN t.outcome<>'active' AND t.outcome_scored_ts>0
      THEN (t.outcome_scored_ts - t.first_seen_ts)/3600.0 END),0) AS avg_life_hours
  FROM tokens t
  LEFT JOIN creators c ON c.address = t.creator
  WHERE t.creator <> ''
  GROUP BY t.creator, c.scored_ts
  ORDER BY c.scored_ts ASC NULLS FIRST   -- skorlanmamış önce, sonra en-eski
  LIMIT $1;
  ```
  `avg_life_hours` = çözülmüş token'larda `(outcome_scored_ts − first_seen_ts)/3600` **yaklaşık proxy** (gerçek ölüm anı değil, worker'ın sınıflandırdığı an; ≤60s pencere). Spec'te açıkça yaklaşık.
- **`UpsertReputation(ctx, rep)`** — `INSERT ... ON CONFLICT (address) DO UPDATE` (tüm hesaplanan kolonlar + `scored_ts`).
- **`Creators(limit)`** — `tokens GROUP BY creator LEFT JOIN creators c` (RecentTokens deseni): total gerçek token'lardan, reputationScore/riskLevel/activeTokens/ruggedTokens/successRatePct `COALESCE(c.…, nötr)` ile. **Yakalanmış ama henüz skorlanmamış creator'lar da listede kalır** (nötr reputation, 2b-1 completeness'i korunur — yalnız `creators` tablosundan okumak onları düşürürdü). Sıralama `total DESC, MIN(first_seen_ts) ASC` (2b-1 ile aynı). `realizedPnlSol=0` (ertelenmiş).
- **`CreatorDetail(addr)`** — `creators` (skor+metrik+breakdown) + `tokens` (token geçmişi, per-token riskFlags türetilir). Creator `creators`'ta yoksa (henüz skorlanmamış) → nötr reputation + `tokens`'tan total/history (mevcut 2b-1 davranışına degrade). `walletAgeDays=0`, `avgFirstSellMinutes=0`, `realizedPnlSol=0`, `behavior` nötr (ertelenmiş).
- **`RecentTokens`** — `tokens LEFT JOIN creators c ON c.address=tokens.creator`; `TokenRow.creatorScore = COALESCE(c.reputation_score, 0)` (creator'sız/skorlanmamış → 0, dürüst).
- **`TokenDetailBase`** (Slice 1c) — o token'ın creator'ı için `creators.reputation_score`/`confidence`/`breakdown` → `TokenDetail.scores.creatorReputation`.

### 4.4 per-token riskFlags (ek veri/persist YOK)
`CreatorDetail`'de `newHistoryItem` içinde, o token'ın mevcut alanlarından saf türetme:
- `outcome=="rug"` → `"Rug çekildi"`
- `outcome=="dumped"` → `"Dump edildi"`
- `outcome=="dead"` → `"Ölü (hacim yok)"`
- `liquidity_status=="removed"` → `"Likidite çekildi"`
- `max_drawdown_pct ≥ REPUTATION_HIGH_DRAWDOWN` (default 80) → `"Yüksek düşüş (%N)"`
- graduated/active → bayrak yok (boş dizi, non-nil)

Frontend `riskFlags` serbest-metin çip render eder (sabit enum yok; `CreatorTokenHistoryTable.tsx`), Türkçe açıklayıcı stringler uygun.

### 4.5 Config + main wiring
Yeni env (hepsi default'lu, deploy-tunable):
| Env | Default | Anlam |
|---|---|---|
| `REPUTATION_ENABLED` | true | Worker aç/kapa |
| `REPUTATION_INTERVAL_SEC` | 60 | Döngü aralığı |
| `REPUTATION_LIMIT` | 60 | Döngü başına skorlanan creator |
| `REPUTATION_MIN_RESOLVED` | 5 | Confidence K (tam güven için çözülmüş token) |
| `REPUTATION_W_RUG` | 50 | Rug ceza ağırlığı |
| `REPUTATION_W_FAIL` | 20 | Dump/dead ceza ağırlığı |
| `REPUTATION_W_GRAD` | 40 | Graduated ödül ağırlığı |
| `REPUTATION_HIGH_DRAWDOWN` | 80 | per-token "yüksek düşüş" bayrağı eşiği (%) |

`main.go`: `REPUTATION_ENABLED && bundle.Tokens != nil` gate (RPC/Helius **GEREKMEZ**) → `reputation.Worker` goroutine (safety/outcome yanında).

---

## 5. Error handling

- **Worker:** `CreatorAggregates` hatası → döngü atla + `logger.Warn("reputation", "err", …)`; tek creator `UpsertReputation` hatası → izole (log + continue). ctx-cancel → temiz çıkış.
- **Config-gate:** RPC yok → yalnız `Tokens != nil`. Kapalıysa (`REPUTATION_ENABLED=false`) worker başlamaz, okuma yolları nötr/degrade döner.
- **Okuma:** `Creators`/`CreatorDetail` err→502, not-found→404 (mevcut handler deseni korunur). `RecentTokens`/`TokenDetail` JOIN → creators yoksa `creatorScore/creatorReputation = 0` (dürüst, hata değil).

---

## 6. Test stratejisi (TDD, SDD ile)

- **Saf `Scorer` unit:** outcome-karışımı senaryoları → beklenen score/confidence/riskLevel/breakdown; edge: `resolved=0`→nötr(conf 0), hepsi-rug→0, hepsi-graduated→90, dump/dead→30, clamp sınırları, K güven eğrisi (1/K → 0.2, ≥K → 1.0).
- **Store parity (fake↔postgres):** `CreatorAggregates` (outcome sayımları + avg_peak + avg_life proxy + skorlanmamış-önce sıralama), `UpsertReputation` (insert+conflict-update), `Creators`/`CreatorDetail` gerçek metrik okuma, `RecentTokens` JOIN `creatorScore`, `TokenDetailBase` creatorReputation.
- **per-token riskFlags:** `newHistoryItem` her outcome/liq/drawdown → beklenen bayrak seti (boş=non-nil).
- **Worker:** kısmi-hata izolasyon (bir creator patlar, diğerleri skorlanır) + ctx-cancel (2b-2a deseni).
- **Frontend:** DEĞİŞMEZ — seam sabit; mevcut testler (`creators.test.ts`, `token.test.ts` vb.) regresyonsuz. Go build/vet/`test -race` yeşil.

---

## 7. Kapsam dışı (bilinçle ertelenen)

- `walletAgeDays` (RPC), `realizedPnlSol`/`avgFirstSellMinutes`/`creatorSellPct` (trade-flow → 2c), `CreatorBehavior` (2c/2e).
- `avgLifetimeHours` = worker-scored-time proxy (gerçek ölüm anı değil; kabul edilen yaklaşım).
- Gerçek graduation (marketCap proxy'si — 2b-2a'dan miras).
- Presentation-layer "0 = bilinmiyor" formatlama (frontend followup).
- creators tablosu için silme/GC (creator token'ları kalıcı; YAGNI).

---

## 8. Sonraki dilimler

2b-2b → **2c** (Manipulation Risk, Python ML — trade-flow persist gerektirir) → **2d** (Opportunity + Overview: getKpis/getRadar/signal) → **2e** (Wallet Graph, god node).

Süreç: writing-plans → SDD (subagent-driven, per-task review + final opus whole-branch review) → push + PR + `gh pr merge --merge --delete-branch` (kullanıcı onayıyla).
