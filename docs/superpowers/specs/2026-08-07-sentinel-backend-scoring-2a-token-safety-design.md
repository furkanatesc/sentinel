# Slice 2a — Token Safety Scoring (`tokenSafety`) (Tasarım)

> Alt-proje 2 (Scoring & Graph) ilk dilimi. Alt-proje 2, frontend kontratının nötr bıraktığı
> 4 skoru + `getCreators`/`getCreator`/`getWalletGraph` + `getKpis`/`getRadar`'ı gerçeğe çevirir.
> Büyük olduğu için 5 dilime ayrıldı: **2a Token Safety** → 2b Creator Reputation → 2c Manipulation
> Risk → 2d Opportunity + Overview → 2e Wallet Graph. Bu doküman yalnız **2a**'yı kapsar.

## 1. Amaç

`tokenSafety` skorunu (0-100, higherIsBetter) gerçeğe çevir: bir token'ın rug/honeypot/scam
riskine karşı **kural-tabanlı, açıklanabilir** bir güvenlik skoru. Frontend'de bu skor hem token
listesinde (`TokenRow.safetyScore`) hem Token Detay'da (`scores.tokenSafety` — breakdown'lı
`ExplainableScore`) görünür. Skorlama saf/kural-tabanlı olduğu için **Go'da** kalır (Python/ML
gerektiren 2b/2c'ye ertelendi).

## 2. Kapsam

### 2.1 Kapsam içi
- `tokenSafety` skoru: 100'den başlayan, kırmızı bayraklarla düşen ağırlıklı checklist. Her kalem
  bir `ScoreBreakdownItem {label, weight, detail}` (frontend hepsini gösterir).
- v1 kontrolleri (launchpad-farkındalıklı):
  1. **Freeze authority aktif** → honeypot riski (tokenlar dondurulabilir). Büyük düşüş.
  2. **Mint authority aktif** → pump.fun bonding-curve'de cezasız (eğri arzı sabitler);
     graduated/genel token + aktif authority → dilution riski (düşüş).
  3. **Top-10 holder yoğunlaşması** → bantlı düşüş (>%80, %50-80, <%50).
  4. **Holder sayısı çok düşük** → ince/rug-eğilimli (düşüş).
  5. **Likidite tabanı** → eşik altı illikit/rug-eğilimli (düşüş).
- Doldurulan frontend alanları: `TokenRow.safetyScore`; `TokenDetail.scores.tokenSafety`
  (breakdown + confidence); `TokenDetail.risks.contract` (freeze/mint authority) + `.market`
  (yoğunlaşma/likidite); `TokenDetail.metrics.top10HolderPct` (gerçek).
- **Confidence (0-1):** veri tamlığını yansıtır. Authority çekilemezse veya holder verisi
  cap'e/hataya takılırsa düşer. Skorlanmamış token → `confidence:0` ("veri yok"; sahte
  "0=güvensiz" değil). (A1 dersi: `confidence:0` = veri yok.)

### 2.2 Kapsam dışı (ertelendi — sessiz düşürme yok)
- `creatorHoldingPct` → creator adresi gerektirir → **2b** (creator yakalama).
- LP-burn / LP-lock tespiti (graduated token'lar) → sonraki safety iterasyonu.
- Diğer 3 skor: `creatorReputation` (2b), `manipulationRisk` (2c), `opportunity` (2d).
- Davranış metrikleri (sniper%, bot%, buy/sell ratio, uniqueBuyers) → **2c**.
- `TokenRow.signal` → opportunity'den türer → **2d**.
- `getCreators`/`getCreator`/`getWalletGraph`/`getKpis`/`getRadar` → sonraki dilimler.
- Listedeki skorlanmamış token `safetyScore=0` görünür (mevcut nötr-skor davranışı; frontend
  seam'i `number` taşır, "veri yok" işareti detayda `confidence` ile — followup'ta işaretli).

## 3. Skorlama modeli

`tokenSafety = clamp(100 − Σ(düşüşler), 0, 100)`. Her kontrol saf bir **`Check`**: girdiden
(authorities + holder dağılımı + liquidity + launchpad) bir `CheckOutcome {deduction, item,
risk?}` üretir. Scorer tüm Check'leri çalıştırır, düşüşleri toplar, breakdown + risks + score +
confidence döndürür. Ağırlıklar/eşikler **config-lenebilir sabitler** (OCP: yeni Check eklemek
mevcut kodu değiştirmez; registry).

- **Ağırlıklar (başlangıç, deploy'da kalibre edilebilir):** freeze-authority-aktif −35;
  mint-authority-aktif (launchpad-aware) −20; top10 >%80 −30 / %50-80 −15 / <%50 0;
  holder<20 −10; liquidity<eşik −10. (Nihai değerler config sabitleri.)
- **Confidence:** her veri kaynağı (authorities, holders) başarıyla alındıysa tam ağırlık;
  biri eksikse ilgili Check atlanır ve confidence orantılı düşer. Hiç veri yoksa `confidence:0`,
  score nötr 0.
- **Risks:** kırmızı bayraklar `RiskItem` de üretir → `risks.contract` (authority) / `.market`
  (yoğunlaşma/likidite). Severity bayrağa göre (`high`/`medium`/`low`).

## 4. Bileşenler (SOLID)

Yeni paket **`internal/safety/`** (SRP: skorlama domain'i).

### 4.1 `SafetyDataProvider` arayüzü (DIP)
```go
type SafetyInputs struct {
    MintAuthorityActive, FreezeAuthorityActive bool
    AuthoritiesKnown                           bool // getAccountInfo başarılı mı
    HolderCount                                int
    Top10Pct                                   float64
    HoldersKnown                               bool // getTokenAccounts başarılı mı
}
type SafetyDataProvider interface {
    SafetyInputs(ctx context.Context, mint string) (SafetyInputs, error)
}
```
Somut impl `internal/ingest/` Helius genişlemelerini kullanır (aşağıda 4.4). Scorer'ı Helius'tan
izole eder (test'te fake).

### 4.2 `Scorer` (saf, SRP — `internal/safety/scorer.go`)
`Score(in SafetyInputs, liquidity float64, launchpad string) SafetyResult`. I/O yok → tam test
edilebilir. `SafetyResult{Score, Confidence, Top10Pct, Breakdown []store.ScoreBreakdownItem,
Risks []store.RiskItem}`. Check registry (OCP) launchpad-aware mantığı içerir.

### 4.3 `Worker` goroutine (Enricher deseni — `internal/safety/worker.go`)
Periyodik: `SafetyScoreTargets` (pool'lu, en eski-skorlanan önce → adil round-robin) → provider →
scorer → `store.UpdateSafety`. Config-gated (`SAFETY_ENABLED` default true, interval, limit).
Helius key yoksa **noop** (dürüst — skorlanmaz). **v1'de kendi WS broadcast'i YOK:** `RecentTokens`
zaten `safety_score`'u okuyor (aşağıda 4.5), dolayısıyla mevcut Enricher broadcast'i (~30s'de bir
`RecentTokens` snapshot'ı yayınlar) güncel `safety_score`'u canlı WS istemcilerine bir enrichment
döngüsü içinde taşır — ayrı bir safety broadcast'ine gerek yok (sadelik).

### 4.4 Helius genişlemesi (`internal/ingest/`)
- `getTokenAccounts` parse'ı `amount`+`owner` döndürsün → owner'a göre topla, sırala, top-10 %
  ve holder sayısı hesapla. **Mevcut `HoldersCount` ile aynı endpoint**; yeni bir
  `HolderDistribution(ctx, mint, cap) (count int, top10Pct float64, capped bool, err error)`
  metodu (veya mevcut çağrıyı genişlet). Cap deseni korunur.
- Yeni `MintAuthorities(ctx, mint) (mintActive, freezeActive bool, err error)` →
  `getAccountInfo(mint, {encoding: jsonParsed})` → `result.value.data.parsed.info.mintAuthority`
  / `freezeAuthority` (null = revoked). Mevcut `rpcURL` reuse.

### 4.5 Store (`internal/store/`)
- Migration `0005`: `tokens`'a `safety_breakdown TEXT DEFAULT ''` (JSON `[]ScoreBreakdownItem`),
  `safety_risks TEXT DEFAULT ''` (JSON `[]RiskItem`), `safety_confidence DOUBLE DEFAULT 0`,
  `top10_holder_pct DOUBLE DEFAULT 0`, `safety_scored_ts BIGINT DEFAULT 0`. (`safety_score`
  0002'de zaten var.)
- `UpdateSafety(ctx, SafetyUpdate)` — skorlama sonucunu yazar.
- `SafetyScoreTargets(ctx, limit)` — skorlanacak token'lar (pool'lu; en eski `safety_scored_ts`
  önce → adil round-robin).
- `RecentTokens` `safety_score`'u **zaten** TokenRow'a okuyor (`tokens.go` SELECT'inde
  `safety_score` var → `t.SafetyScore`) → liste `safetyScore` worker yazınca otomatik dolar,
  ek değişiklik yok. `TokenDetailBase` (veya yeni okuma metodu) safety breakdown/risks/confidence/
  top10'u detail için taşır.
- Fake store paritesi (aynı davranış) + postgres integration testinde round-trip.

### 4.6 Detail + Liste bağlama
- `TokenDetailService.Build`: `scores["tokenSafety"]`'i DB breakdown/confidence/score'dan kurar
  (diğer 3 skor nötr kalır); `risks.contract/market`'ı DB safety_risks'ten (JSON parse); 
  `metrics.top10HolderPct`'i DB'den. **Canlı Helius çağrısı yok** (Option A deseni — throttle-dayanıklı).
- Liste (`RecentTokens`): `safety_score` → `TokenRow.safetyScore`.

## 5. Config

- `SAFETY_ENABLED` (default true) — false ise worker başlamaz.
- `SAFETY_INTERVAL_SEC` (default 60) — skorlama döngüsü.
- `SAFETY_LIMIT` (default 40) — döngü başına skorlanan token.
- `SAFETY_HOLDERS_CAP` (default `HOLDERS_CAP`=5000) — dağılım için sayfalama tavanı.
- Ağırlık/eşik sabitleri kod içinde (v1). Helius key gerekli (yoksa noop).

## 6. Test

- **`Scorer` (saf):** tablo-tabanlı — her Check, launchpad dalları (pump.fun vs genel mint
  authority), top-10 bantları, holder<20, liquidity eşiği, confidence (kısmi veri), clamp [0,100].
- **Provider (Helius):** httptest fixture — `getAccountInfo` (jsonParsed, null/aktif authority) +
  `getTokenAccounts` (amount/owner → top-10 hesabı, cap). 1a decoder fixture deseni.
- **Worker:** fake provider + fake store — persist doğru, interval, key-yok→noop, kısmi-hata izole.
- **Store:** fake/postgres parity; JSON breakdown/risks round-trip (postgres DATABASE_URL'liyse).
- **Detail:** safety alanları DB'den doğru eşlenir (mevcut detail testlerini genişlet); diğer skorlar
  nötr kalır (regresyon).
- Go build/vet/`test -race` yeşil; frontend testleri **değişmez** (seam sabit — saf backend).

## 7. Clean Code & SOLID

- **SRP:** `Scorer` (saf skorlama) / `SafetyDataProvider` (veri) / `Worker` (zamanlama) / store (kalıcılık) ayrı.
- **DIP:** `Worker` somut Helius'a değil `SafetyDataProvider` + `SafetyStore` soyutlamalarına bağımlı.
- **OCP:** Check registry — yeni güvenlik kontrolü eklemek Scorer'ı değiştirmeden (2b creatorHolding%,
  LP-burn ileride buraya eklenir).
- **ISP:** dar arayüzler (SafetyDataProvider tek amaç; store yalnız gereken metotlar).
- **Clean:** anlamlı isimler, saf fonksiyon çekirdeği (I/O kenarında), TDD, breakdown açıklanabilir.

## 8. Kabul kriterleri (deploy'da doğrulanır)

- Go build/vet/`test -race` + frontend testleri yeşil.
- Deploy sonrası: skorlanan token'lar `/api/tokens`'da gerçek `safetyScore` (0 dışı, çeşitli);
  `/api/token/{mint}` `scores.tokenSafety` breakdown + confidence + `risks` + `top10HolderPct`
  gerçek döndürür. Diğer skorlar nötr kalır.
- **Kalibrasyon riski (deploy'da):** Helius `getAccountInfo` (jsonParsed authority alan adları) +
  `getTokenAccounts` (`amount`/`owner` şekli) — canlı şekil farklıysa parse+fixture birlikte
  güncellenir (1a/1c deploy-hotfix deseni).
- **Yeni ücret/key YOK** (mevcut `HELIUS_API_KEY`). Frontend'e dokunulmaz (seam sabit).

## 9. Kullanıcı aksiyonları

- Merge sonrası Railway otomatik deploy (migration 0005 goose ile). Yeni env opsiyonel (default'lar
  iş görür); `HELIUS_API_KEY` zaten var.
- Helius çağrı hacmi artar (worker: ~2 çağrı/token/döngü). Free-tier baskısı olursa interval/limit
  ile kısılır (config). Sürekli WS blokörü (1a) bu dilimi etkilemez (REST getAccountInfo/getTokenAccounts).
