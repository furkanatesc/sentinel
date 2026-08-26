# SENTINEL Backend — Alt-proje 2, Slice 2e-2: Authority Graph — controls_authority Kümeleme (Tasarım)

**Tarih:** 2026-08-26
**Kapsam:** Alt-proje 2 (Scoring & Graph) → **2e (Wallet Graph)** → **2e-2 (controls_authority slice — authority-kontrol kümeleme)**
**Önceki dilimler:** 2a (Token Safety), 2b-1 (Creator Capture), 2b-2a (Outcome), 2b-2b (Creator Reputation), 2c (Manipulation), 2d (Opportunity+Overview), **2e-1 (Creator-Funding Kümeleme / Bundler Tespiti)** — hepsi MERGED + canlı.
**Bağımlılık:** 2a safety'nin `getAccountInfo` (authorities) RPC altyapısı — **zaten canlı** (PR #12). **Yeni key/ücret/RPC YOK** (authority pubkey aynı çağrıdan gelir, şu an atılıyor).

---

## 1. Amaç

2e-1 "tek cüzdan kaç launch'ı **fonluyor**?" sorusunu cevapladı. 2e-2 farklı bir kontrol eksenini yüzeye çıkarır: **"tek cüzdan kaç token'ın mint/freeze authority'sini kontrol ediyor?"** — aynı cüzdan ≥K token'ın **on-chain yetkisini** elinde tutuyorsa, bu seri-rug / arz-enflasyonu / honeypot operatörü sinyalidir (mint authority = ek arz basma; freeze authority = hesap dondurma).

`getAuthorityGraph()` **parametresizdir** → tek bir graph döndürür: yalnız **≥K token'ın authority'sini kontrol eden** `authority_wallet` god node'ları (bilinen-program/derece-filtreli) + kontrol ettikleri token'lar.

**2e Wallet Graph dilimleri:** 2e-1 (creator-funding, MERGED) → **2e-2 (bu — controls_authority slice)** → 2e-2-kalanı (trade/holder edge'leri, veri-önkoşullarına DEFER) → 2e-3 (ileri kümeleme/god-node skorlama, muhtemel Python).

---

## 2. Kapsam kararları (brainstorming'de onaylı, 2026-08-26)

### 2.1 2e-2 üç edge'e bölündü — yalnız controls_authority bu turda
2e-2'nin hedeflediği üç edge'in **hiçbiri** mevcut persist'li veriden hazır değildi (bağlam keşfi):

| Edge | Veri durumu | Karar |
|------|------------|-------|
| **bought/sold** (trade) | Bireysel trader cüzdanı yok (yalnız GeckoTerminal agrega `txns_*` sayıları) → yeni harici kaynak şart (Helius enhanced-tx / Bitquery / swap decode) | **DEFER** (veri kaynağı önkoşulu) |
| **holder co-holding** | Per-owner holding persist yok + `getTokenAccounts` DAS-gated (Helius-safety **Parça2** blocker) | **DEFER** (Parça2'ye bağlı) |
| **controls_authority** | Authority pubkey'i canlı authorities RPC **zaten parse ediyor** (`ingest/authorities.go:48-49`) ama bool'a indirgeyip **atıyor** → yakalamak **sıfır ek RPC** | **BU TUR** |

Gerekçe: en yüksek değer/maliyet oranı. Veri neredeyse bedava (canlı RPC reuse), 2e-1 saf-DB desenini birebir taklit eder, gerçek yeni bir küme tipi teslim eder ve — safety worker canlı olduğu için — **prod'da gerçek veri döndürür** (2e-1 wallet-graph'ın aksine; o funder worker `WALLET_GRAPH_ENABLED=false` ile kapalı).

### 2.2 Option A — saf-DB okuma + safety piggyback yakalama (onaylı)
2a/2b/2c/2d/2e-1 ile aynı **Option A**: veri arka planda yakalanır+persist edilir, `getAuthorityGraph` DB'den küme kurar (canlı RPC yok → throttle-dayanıklı). **Yakalama = safety worker'a piggyback** (§3): ayrı worker YOK — authorities RPC zaten safety için çağrılıyor; dönen pubkey atılmak yerine persist edilir. **Stack: Go** (deterministik graph + SQL agrega; ML yok).

### 2.3 Edge modeli: birleşik `controls_authority` + rol etiketi (onaylı)
Tek edge tipi `controls_authority`; hub = ≥K token'ın **mint VEYA freeze** authority'sini tutan cüzdan. Edge bir **rol** taşır: `mint` / `freeze` / `both` (aynı cüzdan hem mint hem freeze authority ise). Ayrı `mint_authority`/`freeze_authority` edge tipleri YAGNI — rol etiketi ayrımı korur, model ikiye katlanmaz. `GraphEdge`'e opsiyonel `Role string \`json:"role,omitempty"\`` eklenir (mevcut funded/created edge'leri boş bırakır → JSON'da omit → frontend kırılmaz, OCP).

### 2.4 Bilinen-program allowlist dışlaması (ZORUNLU — 2e-1 CEX analoğu, onaylı)
pump.fun/launchpad token'ları sıklıkla **paylaşılan program/PDA authority** taşır → tek adres yüzlerce token'ın authority'si → **devasa sahte god-node**. İki katmanlı filtre (2e-1 CEX deseni birebir):
- (a) **Bilinen-program allowlist** — pump.fun mint-authority PDA, SPL Token Program, Token-2022, System Program, null/boş → dışlanır (`walletgraph/authority_exclude.go`, CEX gibi web-doğrulanır).
- (b) **Derece tavanı** (`MaxDegree`) — allowlist'te olmayan ama aşırı-yüksek-dereceli (≥ MaxDegree) authority = muhtemel bilinmeyen program/sistem → dışlanır (emniyet ağı).

### 2.5 Şüpheli-küme odağı (onaylı)
Parametresiz endpoint = **yalnız kontrol kümeleri**: derece ≥ K (MinCluster) authority'ler. İzole tek-token authority'leri (normal) grafı şişirir, sinyal vermez → dışarıda.

### 2.6 Ertelenen (açık işaretli → DEFER)
- **trade edge'leri** (`bought`/`sold`, `trader_wallet`/`smart_wallet` node): per-trade cüzdan kaynağı gelince (2e-2-kalanı).
- **holder edge'leri** (co-holding, `liquidity_pool` node): Helius-safety **Parça2** (DAS sağlayıcı) + per-owner persist gelince.
- `balanceSol` node alanı (getBalance ek RPC) → 2e-1 gibi ertelendi.
- İleri kümeleme / çok-eksenli birleşik graph → 2e-3.

### 2.7 Dürüstlük notu
Graph **gerçek ama seyrek** olabilir — çoğu pump.fun token'ı bonding-curve sonrası authority'sini **iptal eder** (null); aktif+bireysel-authority tutan token'lar azınlıktır. Küme yoksa `{nodes:[], edges:[]}` (dürüst boş, sahte node DEĞİL). Mevcut token'lar safety worker onları **yeniden skorlayana** kadar authority kolonları boştur (dürüst; zamanla dolar — 2e-1 backfill deseni).

---

## 3. Veri yakalama — safety worker piggyback (kesin)

**Arayüz değişikliği (`internal/safety`):**
`Authorities.MintAuthorities` bool yerine pubkey döndürür:
```go
// önce: (mintActive, freezeActive bool, err error)
// sonra:
MintAuthorities(ctx context.Context, mint string) (mintAuthority, freezeAuthority *string, err error)
```
- `ingest/authorities.go`: satır 66 `return info.MintAuthority != nil, ...` → `return info.MintAuthority, info.FreezeAuthority, nil` (pubkey'i atmayı bırak; `*string`, nil = iptal).
- **Safety scorer/provider mantığı korunur:** `active := (mintAuthority != nil)` türetilir → mevcut `OnChainData.MintAuthorityActive/FreezeAuthorityActive` aynen dolar, safety skoru+testleri değişmez.
- **Persist (karar — kod okunduktan sonra netleşti):** ayrı `SetAuthorities` metodu DEĞİL; `OnChainData`'ya `MintAuthorityAddr/FreezeAuthorityAddr string` + `SafetyUpdate`'e `MintAuthority/FreezeAuthority string` + `AuthoritiesKnown bool` eklenir, mevcut `UpdateSafety` yazımıyla persist edilir. Bu, kod tabanındaki **`CreatorHoldingPct` piggyback desenini birebir** taklit eder (provider.go:8-10 / worker.go:91: "Scorer'a girmez, sadece persist edilir") → tek round-trip, tek metod, mevcut desen. `AuthoritiesKnown=false` (RPC fail) → mevcut değer EZİLMEZ (`CreatorHoldingKnown` guard'ı); nil pubkey (iptal) → `''`.

**Not:** yakalama safety worker'a bağlı → `WALLET_GRAPH_ENABLED` bunu gate ETMEZ (safety her zaman çalışır). Bu bilinçli: toplama bedava olduğundan her zaman açık; gating yalnız okuma/endpoint katmanında gerekirse (§6.4).

---

## 4. Depolama (migration 0013)

`tokens` tablosuna iki kolon (authority token'a **1:1** → ayrı tablo YAGNI; safety kolonları gibi token'da yaşar):
```sql
-- +goose Up
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS mint_authority   TEXT NOT NULL DEFAULT '';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS freeze_authority TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_tokens_mint_authority   ON tokens (mint_authority)   WHERE mint_authority   <> '';
CREATE INDEX IF NOT EXISTS idx_tokens_freeze_authority ON tokens (freeze_authority) WHERE freeze_authority <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_tokens_freeze_authority;
DROP INDEX IF EXISTS idx_tokens_mint_authority;
ALTER TABLE tokens DROP COLUMN IF EXISTS freeze_authority;
ALTER TABLE tokens DROP COLUMN IF EXISTS mint_authority;
```
- `''` = iptal edilmiş VEYA henüz skorlanmamış (ikisi de küme adayı değil → doğru davranış).
- Derece SAKLANMAZ → sorgu-zamanı `GROUP BY authority`.

---

## 5. getAuthorityGraph sorgusu + graph kurma (kesin)

`store.AuthorityGraphClusters(ctx, minCluster, maxDegree int) ([]AuthorityRow, error)`:

1. **Küme adayları (unpivot mint+freeze):** her token'ın dolu authority'lerini `(authority, mint, symbol, role, safety_score, first_seen_ts)` satırlarına aç, authority başına distinct token say:
   ```sql
   WITH auth AS (
     SELECT mint_authority AS authority, mint, symbol, 'mint'   AS role, safety_score, first_seen_ts
       FROM tokens WHERE mint_authority   <> ''
     UNION ALL
     SELECT freeze_authority AS authority, mint, symbol, 'freeze' AS role, safety_score, first_seen_ts
       FROM tokens WHERE freeze_authority <> ''
   )
   SELECT authority, mint, symbol, role, safety_score, first_seen_ts
   FROM auth
   WHERE authority IN (
     SELECT authority FROM auth
     GROUP BY authority
     HAVING COUNT(DISTINCT mint) >= $minCluster AND COUNT(DISTINCT mint) <= $maxDegree
   )
   ORDER BY authority, mint;
   ```
   (Bilinen-program allowlist filtresi Go tarafında: `authority NOT IN programSet` — küçük set, 2e-1 CEX deseni.)
2. **Graph kurma (Go, saf) — `BuildAuthorityGraph(rows) WalletGraphResult`:**
   - Node: her authority → `authority_wallet` (id=`auth:<addr>`, label kısaltılmış addr, riskLevel dereceden: `ScoreToLevel(100 - min(degree*20,100))` — 2e-1 hub deseni); her token → `token` (id=`tok:<mint>`, label symbol, riskLevel safety'den). **Rol birleştirme:** aynı (authority,token) hem mint hem freeze satırıysa rol=`both`.
   - Edge: `controls_authority` (authority→token, `Role:"mint"/"freeze"/"both"`). id=`e:ctrl:<authority>:<mint>`.
   - `firstSeen`/`lastSeen`: token first_seen (RFC3339 veya "—").
3. **Boş küme → `{nodes:[], edges:[]}`.**

> **Karar (allowlist yeri):** Program dışlaması Go tarafında (`IsProgramAuthority`) — 2e-1 CEX deseniyle simetrik; küçük set SQL'e gömmek yerine kodda test edilebilir kalır. Derece tavanı SQL'de (mega-hub'ı erkenden eler, transfer yükünü azaltır).

---

## 6. Mimari

### 6.1 `internal/walletgraph/` (mevcut paket genişler)
- `authority_graph.go` — saf `BuildAuthorityGraph(rows []store.AuthorityRow) store.WalletGraphResult` (SRP, ağsız/DB'siz). `shortAddr`/`rfc3339`/`mapNodes`/`mapEdges` reuse (graph.go).
- `authority_exclude.go` — `knownProgramAuthority map[string]string` + `IsProgramAuthority(addr) bool` (cex.go deseni; web-doğrulanmış program adresleri).

### 6.2 Store (migration 0013 + metotlar)
- **Persist:** ayrı metod yok — `SafetyUpdate` genişletilir (`MintAuthority/FreezeAuthority/AuthoritiesKnown`), mevcut `UpdateSafety` (fake+pg) iki kolonu koşullu yazar (`CASE WHEN AuthoritiesKnown`; §3 kararı, CreatorHolding deseni).
- `AuthorityGraphClusters(ctx, minCluster, maxDegree int) ([]AuthorityRow, error)` — §5.1.
- `AuthorityRow` tipi: `{Authority, Mint, Symbol, Role string; SafetyScore float64; FirstSeenTs int64}`.
- Fake + Postgres parity (2a/2b/2e-1 deseni).

### 6.3 safety (arayüz + worker)
- `Authorities.MintAuthorities` → pubkey döndürür (§3); `ingest/authorities.go` + fake/testler güncellenir.
- Safety worker: `FetchOnChain` dönen authority pubkey'lerini `UpdateSafety(SafetyUpdate{... MintAuthority, FreezeAuthority, AuthoritiesKnown})` ile persist eder (piggyback; ek çağrı yok). Kısmi-hata izole.

### 6.4 API
- `GET /api/authority-graph` → `store.AuthorityGraphClusters` → `BuildAuthorityGraph` → `WalletGraphResult{nodes,edges}` (nil-guard, err→502, boş→`{nodes:[],edges:[]}`). Her zaman kayıtlı (wallet-graph deseni). `WalletGraphMinCluster/MaxDegree` reuse — **yeni config YOK**. RouterDeps/main wiring.

### 6.5 Config + main
- **Yeni env YOK.** `WALLET_GRAPH_MIN_CLUSTER`(2)/`WALLET_GRAPH_MAX_DEGREE`(50) her iki graph için ortak (aynı semantik). Yeni worker YOK (piggyback).

### 6.6 Frontend (karar: bu turda dahil, 2e-1 simetrisi)
- `getAuthorityGraph` seam gerçek fetch (`notReady` yerine) + `live-endpoints.ts`+1 (2e-1'de getWalletGraph aynı turda eklendi). **UI dokunulmaz** (1c/2d/2e-1 deseni — seam zaten `WalletGraph` tipini taşıyorsa reuse; taşımıyorsa tip eklenir). Frontend testleri yeşil kalır.

---

## 7. Error handling

- Safety worker: `MintAuthorities` hata → mevcut safety guard (authorities-unknown) korunur; `SetAuthorities` hata → izole (o token'ın authority'si boş kalır, safety skoru etkilenmez), WARN.
- Authority iptal edilmiş (null) → `''` persist → küme adayı değil (doğru).
- API: store nil → 502; boş küme → `{nodes:[],edges:[]}` (200).

## 8. Test stratejisi (TDD, SDD ile)

- **authorities.go** (fake RPC): pubkey döndürme (mint+freeze dolu/null); jsonParsed parse. **Safety parity:** bool-türetme (`!=nil`) ile mevcut safety skoru + testleri yeşil kalır.
- **BuildAuthorityGraph** (saf): küme→node/edge sayıları; `authority_wallet`/`token` node tipleri+risk; rol `mint`/`freeze`/`both` birleştirme; program-dışlama; boş→boş graph; MinCluster/MaxDegree sınırları.
- **IsProgramAuthority**: bilinen program dışlanır; bilinmeyen geçer.
- **Store**: `AuthorityGraphClusters` unpivot+HAVING agrega (derece eşiği+tavan); `SetAuthorities` upsert; **fake/postgres parity**.
- **Safety worker**: `SetAuthorities` piggyback çağrısı; kısmi-hata izole.
- **API**: `/api/authority-graph` (nil-guard, boş→`{nodes:[],edges:[]}`).
- **Frontend**: `getAuthorityGraph` gerçek fetch + hibrit (getApi seam).
- `go build`/`vet`/`test -race ./...` + frontend suite.

## 9. Kapsam dışı (bilinçle ertelenen → DEFER/followups)

- **2e-2 trade edge'leri** (`bought`/`sold`, `trader_wallet`/`smart_wallet`): per-trade cüzdan kaynağı (Helius enhanced-tx / Bitquery / swap decode) gelince.
- **2e-2 holder edge'leri** (co-holding, `liquidity_pool`): Helius-safety Parça2 (DAS sağlayıcı) + per-owner persist gelince.
- 2e-3: ileri kümeleme / god-node skorlama / çok-eksenli birleşik graph (muhtemel Python).
- `balanceSol` node alanı (getBalance ek RPC).
- Bilinen-program allowlist genişletme (küçük set; büyüdükçe config/dosya).
- authority-graph için ayrı MinCluster/MaxDegree (şimdilik wallet-graph ile ortak).

## 10. Sonraki dilimler

- **2e-2 trade/holder edge'leri** (veri önkoşulları gelince) → **2e-3 ileri kümeleme** → **Alt-proje 2 TAMAM.**
- Paralel açık: safety holders→DAS (Parça2 — trade/holder edge'lerini de açar); creator/trade verisi için güvenilir-WS/veri sağlayıcı.
- Sonra: Alt-proje 3 (Alerts & Telegram) / 4 (Strategies & backtest) / 5 (Trading — A5 Jito sendBundle referansı).
