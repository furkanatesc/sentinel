# SENTINEL Backend — Alt-proje 2, Slice 2e-1: Wallet Graph — Creator-Funding Kümeleme (Tasarım)

**Tarih:** 2026-08-24
**Kapsam:** Alt-proje 2 (Scoring & Graph) → **2e (Wallet Graph)** → **2e-1 (Creator-Funding Kümeleme / Bundler Tespiti)**
**Önceki dilimler:** 2a (Token Safety), 2b-1 (Creator Capture), 2b-2a (Outcome), 2b-2b (Creator Reputation), 2c (Manipulation), 2d (Opportunity+Overview) — hepsi MERGED + canlı.
**Bağımlılık:** `tokens.creator` (2b-1) + creatorfill'in getSignaturesForAddress/getTransaction RPC altyapısı. **Yeni key/ücret YOK** (mevcut SOLANA_RPC_URL/Helius).

---

## 1. Amaç

`getWalletGraph()` frontend kontratını gerçeğe döndürmenin **ilk dilimi**: cüzdan-ilişki grafının **creator-funding kümeleme** kısmı. Değer sorusu: **"tek bir cüzdan kaç token launch'ının arkasında?"** — aynı funding cüzdanından beslenen creator'ları kümeleyerek **bundler / seri-rug operatörü** sinyali üretir.

`getWalletGraph()` **parametresizdir** → tek bir graph döndürür. 2e-1 bu graph'ı **şüpheli-küme odağıyla** üretir: yalnız **≥K creator'ı fonlayan** `funding_wallet` god node'ları (CEX/derece-filtreli) + onların creator'ları + token'ları.

**2e Wallet Graph 3 dilime bölündü (brainstorming'de onaylı):** 2e-1 (bu — creator-funding kümeleme, Go) → 2e-2 (trade/holder graph) → 2e-3 (ileri kümeleme/god-node skorlama, muhtemel Python).

---

## 2. Kapsam kararları (brainstorming'de onaylı)

### 2.1 Option A — saf-DB okuma yolu + arka plan yakalama
2a/2b/2c/2d ile aynı **Option A** deseni: arka plan worker funder'ları yakalar + persist eder; `getWalletGraph` DB'den kümeleri kurar (canlı RPC yok → throttle-dayanıklı). **Stack: Go** — deterministik graph kurma + SQL agrega; ML yok (Python asıl değeri 2e-3 ileri kümeleme için).

### 2.2 Şüpheli-küme odağı (onaylı)
Parametresiz endpoint için kapsam = **yalnız bundler kümeleri**: derece ≥ K funder'lar (aşağıda filtre). "Son-pencere" veya "tüm creator" değil — çünkü asıl değer, çok-launch arkasındaki cüzdanları yüzeye çıkarmak; izole tek-token creator'ları grafı şişirir, sinyal vermez.

### 2.3 CEX/borsa false-positive filtresi (onaylı: derece-eşiği + küçük allowlist)
Birçok creator aynı **CEX hot-wallet'ından** (borsa çekimi) fonlanır → sahte dev "bundler kümesi". İki katmanlı filtre: (a) **derece-eşiği** — çok-yüksek-dereceli funder'lar (≥ MaxDegree) CEX/faucet sayılıp dışlanır; (b) **küçük CEX allowlist** — bilinen büyük CEX hot-wallet'ları `exchange_wallet` etiketlenir + kümelemeden dışlanır.

### 2.4 Funder heuristiği: en eski inbound SOL
Bir creator cüzdanının **funder'ı** = ona **ilk SOL gönderen** cüzdan. Tespit: `getSignaturesForAddress(creator)` en eski sayfa → en eski tx → `jsonParsed` içinde `system` program `transfer` (destination=creator) → source=funder. Heuristik (standart yaklaşım); guard non-corrupting (bulunamazsa funder="", pipeline bozulmaz — 2b-1 creator-offset deseni).

### 2.5 Ertelenen (açık işaretli → 2e-2/2e-3 / followups)
- Node tipleri: `trader_wallet`/`smart_wallet`/`suspicious_wallet`/`liquidity_pool` → 2e-2 (trade/holder graph).
- Edge tipleri: `bought`/`sold`/`transferred`/`provided_liquidity`/`removed_liquidity` → 2e-2; `controls_authority` → 2e-2 (authority adresi 2a'da saklanmıyor, yakalanmalı).
- `balanceSol` (node) → ertelendi (node başına getBalance = ek RPC).
- İleri kümeleme / god-node skorlama / smart-money → 2e-3 (muhtemel Python).

### 2.6 Dürüstlük notu
Graph **gerçek ama seyrek** — creator verisi WS-dormant yüzünden az → az funder → az küme. Küme yoksa `{nodes:[], edges:[]}` (dürüst boş, sahte node DEĞİL; frontend boş graph çizer). Creator verisi yoğunlaştıkça (WS güvenilir sağlayıcı gelince) graph otomatik zenginleşir.

---

## 3. Funder heuristiği (kesin)

`FunderResolver.ResolveFunder(ctx, wallet) (funder string, found bool, err error)`:
1. `getSignaturesForAddress(wallet)` — creatorfill deseni: sayfalayarak EN ESKİ imzayı bul (`maxSigPages` cap, newest-first → son eleman en eski).
2. En eski tx'i `getTransaction` (`jsonParsed` encoding).
3. Tx'in instruction'larında (top-level + inner) `program == "system"` ve `parsed.type == "transfer"` (veya `transferChecked`) olan, `parsed.info.destination == wallet` olan İLK transfer'ı bul → `parsed.info.source` = funder.
4. Bulunamazsa (`transfer` yok / self-fund / decode fail) → `("", false, nil)` (dürüst not-found).
- Cap'e takılırsa found=false (creatorfill deseni).
- **Kalibrasyon riski (deploy'da):** jsonParsed instruction şekli — 1a/2a deseni, guard non-corrupting.

**Not (self-funding / aracı):** funder creator'ın kendisiyse veya bulunamazsa küme oluşmaz (dürüst). Zincir-üstü aracı cüzdanlar (creator'ı fonlayan ara-cüzdan) 2e-1'de tek-hop yakalanır (çok-hop iz sürme 2e-3).

---

## 4. Depolama (migration 0012)

Yeni tablo:
```sql
CREATE TABLE IF NOT EXISTS wallet_funders (
    wallet      TEXT PRIMARY KEY,
    funder      TEXT   NOT NULL DEFAULT '',
    resolved_ts BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_wallet_funders_funder ON wallet_funders (funder) WHERE funder <> '';
```
- `wallet` = creator adresi (2e-1); tablo cüzdan-agnostik (trader'a genişletilebilir 2e-2).
- `funder=''` = çözüldü ama funder bulunamadı (dürüst; yeniden-denemeyi önlemek için `resolved_ts>0` işaretlenir).
- Derece SAKLANMAZ → sorgu-zamanı `GROUP BY funder COUNT(DISTINCT wallet)`.

---

## 5. getWalletGraph sorgusu + graph kurma (kesin)

`store.WalletGraph(ctx, params) (GraphData, error)` — params: `MinCluster`(K), `MaxDegree`, `cexSet`.

1. **Küme adayları:** `tokens` ⋈ `wallet_funders` (ON creator=wallet, funder<>''):
   ```sql
   SELECT wf.funder, COUNT(DISTINCT t.creator) AS degree
   FROM tokens t JOIN wallet_funders wf ON wf.wallet = t.creator
   WHERE wf.funder <> '' AND t.creator <> ''
   GROUP BY wf.funder
   HAVING COUNT(DISTINCT t.creator) >= $MinCluster AND COUNT(DISTINCT t.creator) <= $MaxDegree
   ```
   (CEX allowlist filtresi Go tarafında: `funder NOT IN cexSet` — küçük set, SQL'e gömmek yerine kodda.)
2. **Detay çekme:** hayatta kalan funder'lar için creator+token satırları (funder, creator, token mint/symbol, token safety_score, creator reputation) çekilir.
3. **Graph kurma (Go, saf):**
   - Node: her funder → `funding_wallet` (id=`fund:<addr>`, label kısaltılmış addr, riskLevel dereceden, address dolu); her creator → `creator_wallet` (id=`wal:<addr>`, riskLevel reputation'dan); her token → `token` (id=`tok:<mint>`, label symbol, riskLevel safety'den).
   - Edge: `funded` (funder→creator), `created` (creator→token), `shares_funder` (aynı funder'lı creator çiftleri — hub-yıldız yerine funding_wallet node'u üzerinden `funded` edge'leri zaten kümeyi gösterir; `shares_funder` edge'i creator-çiftleri için opsiyonel, YAGNI: `funded` yıldızı yeterli → **`shares_funder` üretilmez, küme funding_wallet hub'ıyla temsil edilir**; edge tipleri: funded+created).
   - `firstSeen`/`lastSeen`: creator/token için token first_seen; funder için min resolved_ts (RFC3339 veya "—").
   - `riskLevel`: `scoreToLevel` (2d Go helper reuse) — funding_wallet: `scoreToLevel(100 - min(degree*20,100))` (büyük küme→düşük skor→kötü); creator: reputation'dan; token: safety'den.
4. **Boş küme → `{nodes:[], edges:[]}`.**

> **Karar (YAGNI):** `shares_funder` edge'i üretilmez — funding_wallet hub node'u + ona bağlı `funded` edge'leri kümeyi zaten görsel/yapısal olarak temsil eder; her creator-çifti için O(n²) `shares_funder` edge'i eklemek grafı şişirir. Frontend `GraphEdgeType` `shares_funder`'ı destekler ama 2e-1 hub-temsili kullanır (2e-2/2e-3'te gerekirse eklenir).

---

## 6. Mimari

### 6.1 Yeni paket `internal/walletgraph/`
- `resolver.go` — `FunderResolver` (creatorfill `sigTxRPC` deseni + jsonParsed transfer parse). DIP: `sigTxRPC` benzeri dar arayüz (test için fake).
- `worker.go` — `Worker` (creatorfill Worker deseni): `FunderTargets(limit)` çeker → `ResolveFunder` → `SetFunder` persist. ctx-cancel, kısmi-hata izole, tik-özet log.
- `graph.go` — saf `BuildGraph(clusters []Cluster, cexSet) GraphData` (SRP, ağsız/DB'siz) — node/edge kurma + risk/label.
- `cex.go` — bilinen CEX hot-wallet allowlist (Go `map[string]bool` const-benzeri; label sağlar).

### 6.2 Store (migration 0012 + metotlar)
- `FunderTargets(ctx, limit) []FunderTarget` — `tokens.creator<>''` olup **henüz çözülmemiş** creator'lar: `wallet_funders`'ta hiç satırı olmayan VEYA satırı olup `resolved_ts=0` olanlar (çözülmüş = `resolved_ts>0`, funder="" bulunamadı-işareti dahil → yeniden-denemez). SQL: `t.creator NOT IN (SELECT wallet FROM wallet_funders WHERE resolved_ts > 0)` (creatorfill CreatorFillTargets deseni). DISTINCT creator, newest-first.
- `SetFunder(ctx, wallet, funder string, resolvedTs int64)` — `wallet_funders` upsert.
- `WalletGraphClusters(ctx, minCluster, maxDegree int) []ClusterRow` — §5.1-5.2 sorgusu (funder+creator+token+skorlar).
- Fake + Postgres parity (2a/2b deseni).

### 6.3 API
- `GET /api/wallet-graph` → `store.WalletGraph` build → `WalletGraph{nodes,edges}` (nil-guard, err→502, boş→`{nodes:[],edges:[]}`). RouterDeps/main wiring.

### 6.4 Config + main
- `WALLET_GRAPH_ENABLED`(true), `FUNDER_RESOLVE_INTERVAL_SEC`(60), `FUNDER_RESOLVE_LIMIT`(40), `WALLET_GRAPH_MIN_CLUSTER`(2), `WALLET_GRAPH_MAX_DEGREE`(50).
- main.go: funder worker goroutine (creatorfill gibi `preferRPC(SolanaRPCURL, rpcURL)` + paylaşılan limiter; RPC varsa çalışır). getWalletGraph handler `bundle.Tokens != nil` ise.

### 6.5 Frontend
- `http.ts` `getWalletGraph` gerçek fetch (`notReady` yerine) + `live-endpoints.ts`+1. **UI dokunulmaz** (1c/2d deseni).

---

## 7. Error handling

- Worker: `FunderTargets` hata → tik atla + WARN. Per-target `ResolveFunder`/`SetFunder` hata → izole (o cüzdan atla, tik-özette sayılır). 429 → creatorfill retry/backoff.
- Funder bulunamadı → `funder=""` + `resolved_ts=now` persist (yeniden-denemez; dürüst).
- API: store nil → 502; boş küme → `{nodes:[],edges:[]}` (200).

## 8. Test stratejisi (TDD, SDD ile)

- **FunderResolver** (fake sigTxRPC): en-eski-tx seçimi; jsonParsed transfer parse (destination=wallet → source=funder); transfer-yok → not-found; cap → not-found; self-fund guard.
- **BuildGraph** (saf): küme→node/edge sayıları; funding_wallet/creator/token node tipleri+risk; CEX-dışlama; boş→boş graph; MinCluster/MaxDegree sınırları.
- **Worker**: kısmi-hata izole; ctx-cancel; not-found persist.
- **Store**: `WalletGraphClusters` agrega (degree HAVING); `FunderTargets` (resolved_ts=0 filtresi); `SetFunder` upsert; **fake/postgres parity**.
- **API**: `/api/wallet-graph` (nil-guard, boş→`{nodes:[],edges:[]}`).
- **Frontend**: `getWalletGraph` gerçek fetch + hibrit.
- `go build`/`vet`/`test -race ./...` + frontend suite.

## 9. Kapsam dışı (bilinçle ertelenen → followups)

- 2e-2: trade/holder graph (trader/smart_wallet + bought/sold/holder edge'leri + controls_authority + authority adresi yakalama).
- 2e-3: ileri kümeleme / god-node skorlama / smart-money / çok-hop funding iz sürme (muhtemel Python).
- `balanceSol` node alanı (getBalance ek RPC).
- `shares_funder` explicit edge (hub-temsili yeterli).
- CEX allowlist genişletme (2e-1 küçük set; büyüdükçe config/dosya).
- Funder heuristiği rafine: aracı-cüzdan/çok-hop, transferChecked/farklı fund kalıpları.

## 10. Sonraki dilimler

- **2e-2 Trade/holder graph** → **2e-3 ileri kümeleme** → **Alt-proje 2 TAMAM.**
- Sonra: Alt-proje 3 (Alerts & Telegram) / 4 (Strategies & backtest) / 5 (Trading — bkz [[sentinel-backend-program]] + A5 Jito sendBundle referansı).
- Paralel açık: safety holders→DAS (Parça 2); creator verisi için güvenilir-WS sağlayıcı (2e'yi zenginleştirir).
