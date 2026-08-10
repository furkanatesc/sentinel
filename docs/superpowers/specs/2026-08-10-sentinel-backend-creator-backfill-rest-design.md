# Slice: REST Creator Backfill (WS blokörünü baypas) (Tasarım)

- **Tarih:** 2026-08-10
- **Bağlam:** Alt-proje 1 (ingestion) + Alt-proje 2 (creator/scoring) ortak blokörü
- **Branch (planlanan):** `feat/backend-creator-backfill`
- **Önceki:** 2b-1 (Creator Capture, WS-tabanlı decoder) + 2b-2a (Outcome) merged
- **Sonraki:** 2b-2b (Creator Reputation) — bu dilim onun canlı verisini açar

## 1. Amaç

Creator verisi şu an **dormant**: pump.fun creator'ı yalnız Helius WS `logsSubscribe` yolu (2b-1) yakalar, ama Helius free-tier WS kısa patlamadan sonra teslimatı kesiyor (bilinen sağlayıcı/altyapı blokörü — kullanıcı 2026-08-05 "şimdilik böyle bırak" demişti). Sonuç: `getCreators`/`getCreator` canlıda boş, 2b-1/2b-2a creator-bağlı alanları + 2b-2b reputation dormant.

**Gözlem:** WS'in tek sağladığı şey **creator**. Token KEŞFİ zaten GeckoTerminal REST ile çalışıyor (canlıda 200 token, creator'sız — `launchpad='Pump.fun'`, `creator=''`). Yani boşluk dar: *GeckoTerminal'in keşfettiği pump.fun token'larına, flaky WS'e güvenmeden creator eklemek.*

**Çözüm (kullanıcı onaylı):** WS'i baypas et — arka plan **REST creator-backfill** enricher'ı, creator'sız pump.fun token'ları için creator'ı **create transaction'ının CreateEvent'inden** getirir (mevcut Helius free-tier RPC, bounded çağrı, yeni key/ücret yok). Bu, backend'i baştan beri şekillendiren desenle aynı: **1b GeckoTerminal WS-keşif-blokörünü REST ile atladı; Option A GeckoTerminal-throttle'ı DB'den-sun ile atladı.**

## 2. Kapsam

### 2.1 Kapsam içi

1. **Helius genişletme:** `getSignaturesForAddress` (mint → sinyaller, en eskiye sayfalama, **cap'li**) + tx fetch `logMessages` döndürür + `CreatorResolver` (mint → creator via create tx decode).
2. **Backfill worker** (yeni `internal/creatorfill/`, 2a/Enricher deseni): creator'sız pump.fun token hedeflerini çeker → her biri için resolver → creator persist (mevcut merge kuralı) + denendi-damgası.
3. **Şema (migration 0008):** `tokens.creator_backfill_ts` (hedef sıralaması + "denendi" işareti — outcome_scored_ts deseni).
4. **Config + main wiring:** `CREATORFILL_*` env + config-gated goroutine.

### 2.2 Kapsam dışı (ertelendi/bilinçli)

- **WS worker SİLİNMEZ** — tamamlayıcı kalır (Helius patlama yaptığında real-time creator yakalar; backfill zaten-creator'lı token'ları atlar). Silmek fazladan iş + real-time yolu kaybı, dürüst gerekçe yok.
- **DAS `getAsset` ile creator** → ELENDİ: pump.fun on-chain Metaplex `creators`'ı genelde doldurmaz (metadata off-chain) → güvenilmez. Tek güvenilir kaynak create tx'in CreateEvent'i (`user`/dev pubkey kesin orada). YAGNI (spekülatif ucuz yol eklenmez).
- **Non-pump.fun launchpad backfill** → kapsam dışı (creator yalnız pump.fun CreateEvent'inde; Raydium/diğer decoder creator vermez, 2b-1 deseni).
- **Ücretli/alternatif WS sağlayıcı** → gereksiz (REST backfill çözüyor); ileride isteğe bağlı.

## 3. Mekanizma (create tx → creator)

`CreatorResolver.ResolveCreator(ctx, mint) (creator string, found bool, err error)`:

1. **En eski sig'i bul:** `getSignaturesForAddress(mint, {limit:1000, before:cursor})` newest-first döndürür; `before` ile sayfalayarak en eskiye in (bir sayfa < limit → en eskiye ulaşıldı, ya da `MaxSigPages` cap). En eski sig = mint'in **create tx**'i (mint hesabı orada oluşturulur).
2. **Tx'i çek:** `getTransaction(oldestSig)` → `meta.logMessages`.
3. **Decode:** mevcut `pumpFunDecoder` iç mantığı (`hasCreateInstruction` + `programDataAll` + `parseCreateEvent`) logMessages üzerinde birebir çalışır (WS ile aynı "Program data:" formatı) → `createEvent.creator` (mint+64 offset, 2b-1 Task 1). CreateEvent tanınmazsa `found=false`.
4. **Dürüst boş:** cap'e takılıp en eski create'e ulaşılamazsa ya da decode başarısızsa → `found=false` (creator boş; damgalanır, tekrar taranmaz kısa vadede).

**Reuse:** resolver `internal/ingest/`'te (decoder iç mantığı + Helius client orada). `parseCreateEvent`/`programDataAll` zaten var (2b-1); logMessages'a uygulanır. `GetTransaction` `meta.logMessages` erişimi eklenir (şu an yalnız AccountKeys çıkarıyor).

**Cost:** taze token (GeckoTerminal dakikalar içinde keşfeder) = az sig = 1 sayfa + 1 getTransaction = ucuz. `MaxSigPages` cap runaway'i önler. Her token bir kez denenir (damga); bulunamayan en düşük öncelikte nadiren retry.

## 4. Bileşenler (SOLID)

### 4.1 Helius (`internal/ingest/helius.go` + `helius_signatures.go`)

- `NewHeliusSignatures(rpcURL)` → `getSignaturesForAddress` (solana-go `GetSignaturesForAddressWithOpts`, `Before`/`Limit` sayfalama).
- `GetTransaction` genişletilir (ya da yeni `GetTransactionLogs`): `out.Meta.LogMessages` döndürür. (Mevcut `TxInfo{AccountKeys}` Raydium için korunur — yeni alan/metot eklenir, kırılmaz.)
- `NewCreatorResolver(rpcURL string, maxSigPages int) CreatorResolver` — yukarıdaki adımları birleştirir; decoder reuse.

### 4.2 `CreatorResolver` arayüzü (DIP — `internal/creatorfill/`)

```go
type CreatorResolver interface {
	ResolveCreator(ctx context.Context, mint string) (creator string, found bool, err error)
}
```
Arayüz `creatorfill`'te tanımlı; somut impl `ingest.NewCreatorResolver(...)` (yapısal karşılar). main wire eder (döngüsel import yok: creatorfill↛ingest, ingest↛creatorfill).

### 4.3 `CreatorFillStore` (DIP — `internal/store/`)

Dar arayüz (ISP):
```go
type CreatorFillTarget struct { Mint string }
type CreatorFillStore interface {
	CreatorFillTargets(ctx, limit) ([]CreatorFillTarget, error)
	SetCreatorBackfill(ctx, mint, creator string, backfillTs int64) error
}
```
- `CreatorFillTargets` SQL: `SELECT mint FROM tokens WHERE launchpad='Pump.fun' AND creator='' ORDER BY creator_backfill_ts ASC, first_seen_ts DESC LIMIT $1` (en eski-denenen önce; GeckoTerminal `launchpad='Pump.fun'` zaten set ediyor — 2a doğrulaması).
- `SetCreatorBackfill` SQL: `UPDATE tokens SET creator=COALESCE(NULLIF($2,''),creator), creator_backfill_ts=$3 WHERE mint=$1` (boş creator mevcut gerçek'i ezmez — 2b-1 merge deseni; damga her denemede güncellenir). postgres + fake parity.

### 4.4 `Worker` (Enricher/2a deseni — `internal/creatorfill/worker.go`)

`WorkerDeps{Store CreatorFillStore, Resolver CreatorResolver, Interval, Limit, Now, Logger}`. `Run(ctx)` ticker; `fillOnce`: `CreatorFillTargets(limit)` → her hedef `ResolveCreator` → `SetCreatorBackfill(mint, creator, now)` (bulundu/bulunamadı fark etmez, damgala). Partial-hata izole (ctx-cancel + continue). Rate-limit doğal (interval + limit; her token birkaç RPC).

### 4.5 main wiring

`main.go`: `if cfg.CreatorFillEnabled && bundle.Tokens != nil && rpcURL != "" { resolver := ingest.NewCreatorResolver(rpcURL, cfg.CreatorFillMaxSigPages); w := creatorfill.NewWorker(...); go w.Run(ctx) }` (safety worker deseni; rpcURL GEREKLİ — Helius RPC).

## 5. Şema (migration 0008)

```sql
-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS creator_backfill_ts BIGINT NOT NULL DEFAULT 0;
-- +goose Down
ALTER TABLE tokens DROP COLUMN IF EXISTS creator_backfill_ts;
```

## 6. Config

`CREATORFILL_ENABLED`(true), `CREATORFILL_INTERVAL_SEC`(30), `CREATORFILL_LIMIT`(20), `CREATORFILL_MAX_SIG_PAGES`(3). Hepsi default'lu. **Yeni key/ücret YOK** — mevcut `HELIUS_API_KEY` free-tier RPC (bounded: interval başına ≤ limit token × birkaç RPC).

## 7. Test

- **Resolver (ingest):** kayıtlı getSignaturesForAddress + getTransaction cevabı (fixture) → create logMessages'tan doğru creator; CreateEvent yoksa `found=false`; cap davranışı. (HTTP mock ya da arayüz-seviyesi fake.)
- **Decode reuse:** mevcut CreateEvent fixture logMessages olarak verilince 2b-1 ile aynı creator.
- **Store:** `CreatorFillTargets` (yalnız pump.fun + creator='' + en eski-denenen önce) + `SetCreatorBackfill` (bulunan set eder, boş ezmez, damga güncellenir) fake↔postgres parity.
- **Worker:** hedefleri resolve edip persist eder; resolver-hata izole; ctx-cancel; bulunamayan yine damgalanır (sonsuz döngü yok).
- Canlı getSignaturesForAddress/getTransaction + DB round-trip yalnız **deploy'da** (yerel Postgres/RPC key yok — 0/1/2a/2b deseni).

## 8. Clean Code & SOLID

- **SRP:** resolver yalnız mint→creator; worker yalnız orkestra; store yalnız persist; decode mevcut 2b-1 mantığı (yeni parse yok).
- **DIP:** worker `CreatorResolver` + `CreatorFillStore` arayüzlerine bağımlı; somutlar (Helius/postgres) main'de wire.
- **OCP:** resolver arayüzü ileride başka kaynak (paid provider/DAS) ile genişletilebilir; worker değişmez.
- **ISP:** dar store + resolver arayüzleri.
- **DRY:** mevcut `parseCreateEvent`/`programDataAll` reuse (WS ve REST aynı decode); merge kuralı tek yerde (COALESCE).
- **Dürüstlük:** bulunamayan creator boş kalır (sahte değil); WS worker korunur (tamamlayıcı, sessiz silme yok); cap + damga runaway/sonsuz-retry önler.

## 9. Kabul kriterleri (deploy'da doğrulanır)

1. Migration 0008 goose ile (yeni zorunlu env yok).
2. Backfill worker creator'sız pump.fun token'ları için create tx'ini bulup creator'ı **gerçek base58** olarak yazar.
3. `/api/creators` **canlıda dolmaya başlar** (GeckoTerminal-keşif pump.fun token'larından creator'lar); `/api/creator/{addr}` gerçek token geçmişi.
4. Bulunamayan token'lar (cap/decode fail) creator='' + damgalı → sonsuz retry yok, log gürültüsü sınırlı.
5. WS worker hâlâ çalışır (patlama olursa real-time creator; backfill onları atlar — çift-yazım yok, merge idempotent).
6. RPC bütçesi makul (bounded; free-tier limitine takılırsa `CREATORFILL_LIMIT`/`_INTERVAL_SEC` ile kısılır — kod değişmez).

## 10. Kullanıcı aksiyonları

- **Merge → Railway deploy** (master push otomatik; migration 0008 goose ile).
- Yeni key / ücret / harici panel **YOK** (mevcut `HELIUS_API_KEY` RPC yeterli).
- Deploy sonrası: creator birikmeye başlar → 2b-1/2b-2a creator alanları + (sonra) 2b-2b reputation canlı anlam kazanır. Free-tier RPC limiti gözlenirse `CREATORFILL_*` ile ayarlanır.
