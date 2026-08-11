# SENTINEL Frontend — Follow-ups (takip listesi)

Bu doküman, final whole-branch review (2026-07-30, Increment 1) ve task review'larında
tespit edilen ama bu artımı **bloke etmeyen** maddeleri kaydeder. Sessiz düşürme yok:
her biri ileride ele alınacak. İlgili artıma etiketlendi.

## HTTP adapter artımıyla birlikte (backend AWS bağlanınca)
- **WalletAddress truncation'ı sunum katmanına taşı.** Şu an kısaltma mock verisinde fake
  (`mint: "9xQeWv...4Fk2"`). `httpApi` gerçek 44-karakter base58 mint döndürünce tablo
  kolonu bozulur. Yapılacak: `lib/format.ts`'e `shortenAddress()` ekle, `WalletAddress`
  içinde kısalt, tam adresi copy/`title` için sakla, `mock.ts` tam mint taşısın.
  Bu, mock|http görsel değiştirilebilirliğini korur. (Final review — Important)
- **Seam swappability testi.** `httpApi` yazılınca `NEXT_PUBLIC_DATA_SOURCE` flip edilip
  Overview'un aynı render'ı ürettiğini doğrulayan bir entegrasyon testi ekle. (Final review — Recommendation)

## CI / araç hijyeni (bir sonraki uygun fırsatta)
- `apps/web/package.json`'a `"typecheck": "tsc --noEmit"` script'i **ve** tsconfig'e
  `"types": ["vitest/globals"]` ekle (ikisi birlikte; test global'leri tsc'de tanımsız görünüyor).
  (Task 4 + Final review)
- `apps/web/package.json`'a `"engines": { "node": ">=20" }` ekle. (Task 1)
- `shadcn` paketini `dependencies` → `devDependencies` taşımayı değerlendir. (Task 3)
- Fresh lockfile'da 12 high npm-audit (transitive) — periyodik güncelleme/denetim. (Task 1)

## UI tutarlılık / polish
- ~~**Dil tutarlılığı**~~ — **KAPANDI (2026-07-30):** UI dili Türkçe seçildi; tüm arayüz +
  mock veri etiketleri + kök metadata Türkçeleştirildi (`cfc9aba`, `b43812f`).
- ~~**Opportunity Radar boş render**~~ — **KAPANDI:** Recharts ResponsiveContainer sizing
  düzeltildi (chart wrapper'a explicit height). (`cfc9aba`)
- ~~**Scrollbar teması**~~ — **KAPANDI:** globals.css'e dark temalı scrollbar eklendi. (`cfc9aba`)
- ~~**create-next-app AGENTS.md/CLAUDE.md**~~ — **KAPANDI:** `apps/web/AGENTS.md` + `CLAUDE.md` silindi. (`cfc9aba`)
- **next-themes ölü ağırlık:** `components/ui/sonner.tsx` `useTheme()` çağırıyor ama
  `ThemeProvider` yok; dark-only'de `theme="dark"` zaten geçiliyor. `next-themes`'i kaldırıp
  `theme="dark"` hardcode etmeyi değerlendir. (Final review — Minor)
- **Radar canlı değil:** `subscribeTokens` `qk.tokens`'ı patch'liyor ama `qk.radar` ayrı
  snapshot; feed animasyonluyken radar statik. Muhtemelen kasıtlı — bir yorum ekle ya da
  radar'ı da canlıya bağla. (Final review — Minor)
- **Sparkline tek-nokta koruması:** `Sparkline.tsx` `step = width/(data.length-1)` tek
  elemanlı seride bölme hatası; `(data.length-1)||1` ile guard'la (mock 16 nokta ürettiği
  için bugün ulaşılmaz). (Final review — Minor)
- Composed `(app)`-layout entegrasyon testi (Sidebar+Header+main birlikte) yok; build
  prerender render'ı büyük ölçüde kapsıyor. İstenirse bir shell-integration testi eklenebilir. (Task 8)
- create-next-app'in bıraktığı `apps/web/AGENTS.md` / `CLAUDE.md` ajanlara "önce Next docs oku"
  enjekte edebiliyor; gerekirse sadeleştir/kaldır. (Task 1)

## Wallet Graph (Increment 4)
- **Stale fade (kozmetik, parked):** `WalletGraphCanvas`'ta seçili bir node varken filtre değişip
  graph rebuild olursa (Effect A), fade Effect B'ye bağlı (`focusNodeId` değişmediği için) tekrar
  uygulanmaz → highlight bir sonraki tıklamaya kadar kaybolur. Kendini düzeltir, crash yok.
  Düzeltme: rebuild sonrası fade'i yeniden uygula (Effect B'ye elements-rebuild sinyali ekle ya da
  Effect A sonunda fade'i çağır). (Final review — parked, non-load-bearing, 2026-07-30.)
- **Sonraki Wallet Graph artımı:** path finder, wallet compare, export (PNG/veri), cluster highlight,
  time-range filter (bu artımda kapsam dışıydı). Ayrıca mock node instance label'ları (Funder-1/Creator-A)
  tam Türkçeleştirilebilir; backend gerçek veriyle değiştirecek.

## Creators (Increment 5)
- **Mock derivation dup (parked):** `lib/api/mock.ts`'te `creatorRow` ve `buildCreator` aynı seed'den
  aynı formüllerle (rep/total/active/rugged/successRate/pnl) hesaplıyor → liste satırı ile profil
  metrikleri sessizce sapabilir. Http backend gelince ikisini tek kaynaktan üret (`creatorBase(addr)`
  helper). Fixture kod, düşük etki. (Final review — parked, 2026-07-31.)
- **avgFirstSellMinutes iki yerde:** hem `CreatorMetrics` ("Ort. İlk Satış") hem `CreatorBehavior`
  ("Ort. ilk satış") gösteriyor — kasıtlı (özet vs davranış detayı), gerekirse birinden kaldır.

## globals.css font notları — KAPANDI
Final review doğruladı: `.font-mono` tek kez tanımlı ve `--font-sans` `@theme inline` içinde
mevcut. Task 1'de işaretlenen iki font notu artık geçerli değil.

## Strategies (Increment 6)
- ~~**EquityCurve statik gradient id (Minor, deferred)**~~ — **KAPANDI (2026-08-03, Increment 7 Task 2):**
  `EquityCurve` `components/strategy/` → `components/sentinel/`'e taşındı (git mv, history korundu) ve
  gradient id artık renkten türetiliyor (`equity-grad-${color.replace("#","")}`). Statik `equity-grad`
  çakışma riski kalktı. Kalıntı: aynı sayfada **aynı renkli** iki EquityCurve instance'ı hâlâ id
  paylaşır (bugün ulaşılmaz — tek tüketici); tam güvenlik için ileride `useId()`. (Increment 7 Task 2.)
- **TileGrid DRY (Minor, deferred):** `grid grid-cols-2 gap-3 md:grid-cols-4` + `tiles.map(MetricTile)`
  deseni `StrategyPerformancePanel`, `BacktestSummaryPanel` ve `StrategyDetailContent` risk/sizing
  bloğunda tekrar ediyor. Küçük bir `TileGrid tiles={...}` presentational bileşeni tekrarı kaldırır
  (erken soyutlama değil). Düşük öncelik. (Final review — Minor.)
- **Detay not-found dalı yalnız backend'de ulaşılır (parked):** `StrategyDetailContent`'in
  `isError`/"Strateji bulunamadı" dalı mock seam'de ulaşılamaz — `mockApi.getStrategy` bilinmeyen id
  için `STRATEGY_DEFS[0]`'a düşüyor (creators/tokens ile tutarlı ileri-dönük savunma kodu). Gerçek
  httpApi reject edene kadar test edilmiş davranış değil; mock'un yanına bir not düşülebilir. (Final review — parked.)
- **Boş filtre sonucu mesajı yok (Minor):** `StrategiesListContent`'te bir durum filtresi hiç kart
  bırakmazsa boş grid render olur; "sonuç yok" mesajı eklenebilir. (Final review — Minor.)
- **mock buildStrategy strategyRow'u yeniden hesaplıyor (Minor):** `buildStrategy` precomputed
  `strategyRows` dizisini kullanmak yerine her id için `strategyRow(def)`'i yeniden çağırıyor — saf,
  deterministik, ucuz; httpApi'ye geçişte tek kaynaktan üretilebilir. (Task 1 — Minor.)

## Portfolio / Positions (Increment 7)
- **EquityCurve aynı-renk id çakışması → `useId()` (Minor, deferred):** Gradient id renkten türetiliyor
  ama aynı sayfada aynı renkli iki EquityCurve id paylaşır. Bugün ulaşılmaz (her sayfada tek instance);
  React `useId()` ile tam benzersizlik ileride. (Increment 7 — Minor; yukarıdaki Increment 6 kaydının devamı.)
- **risk-allocation `good` (#17B890) vs `strong` (#2FD98B) yakın hue (Minor, deferred):** Fix wave dilim
  renklerini ayrıklaştırdı (donut + legend artık okunur) ama iki yeşil tonu hâlâ komşu hue'da. Daha güçlü
  ayrım için palet ileride ayarlanabilir. Kritik değil. (Fix wave re-review — Minor, 2026-08-03.)
- **`creatorRisk` mock aralığı `critical`'a ulaşamıyor (parked):** Fix wave `tokenRisk` aralığını genişletti
  (Kritik filtresi artık satır bırakır) ama `creatorRisk` hâlâ `critical` üretmiyor. **Dead filter yok** —
  filtre yalnız `tokenRisk`'e göre (creator-risk filtresi mevcut değil). Backend gerçek veriyle değiştirecek.
  (Fix wave re-review — parked, 2026-08-03.)
- **Filtre çipleri `aria-pressed` yok (Minor):** `PositionsContent` risk filtre çipleri seçili durumu
  görsel (renk) veriyor ama `aria-pressed` taşımıyor; erişilebilirlik için eklenebilir. (Task 12 — Minor.)
- **Artan (ascending) sıralama toggle yok (Minor, kasıtlı):** `PositionsTable` yalnız azalan sıralar
  (brief'e uygun); tıklayınca yön değiştiren toggle ileride eklenebilir. (Task 12 — Minor, brief ile uyumlu.)
- **CI hijyeni tekrar teyit (typecheck script):** Task 13'te `next build` tsc'si, vitest'in kaçırdığı bir
  Recharts `Formatter` tip hatasını yakaladı (RiskAllocationChart tooltip). `"typecheck": "tsc --noEmit"`
  script'i build'den önce tsc-only hataları yakalardı — yukarıdaki CI hijyeni maddesini pekiştirir. (Task 13.)

## Trading Terminal (Increment 8)
- **Order paneli sağ kolonu dar viewport'ta kırpılıyor (Minor, deferred — görsel doğrulama 2026-08-03):**
  `TerminalContent` grid'i `lg:grid-cols-[220px_1fr_300px]`; sidebar + içerik dar kaldığında sağ order
  paneli (300px) yatayda kısmen görünüm dışına taşar. Build/işlev etkilenmez; daha küçük min genişlik,
  order panelini alta yığma (md breakpoint) veya `1fr`'yi daraltma ile iyileştirilebilir. Mobilde tam
  terminal zaten kapsam dışı. (Görsel doğrulama — Minor.)
- **`lightweight-charts` v4 → v5 yükseltme notu (deferred):** `^4.2.3` pinli; v5 seri API'sini değiştirdi
  (`addCandlestickSeries` → `chart.addSeries(CandlestickSeries, ...)`). İleride v5'e geçilirse
  `PriceChartCanvas.tsx` + test mock'u güncellenmeli. (Task 6 — deferred.)
- **`series.setData(candles as never)` geniş cast (Minor, deferred):** lightweight-charts'ın markalı
  `CandlestickData` tipi yerine `as never` kullanılıyor; tek çağrı, runtime riski yok. Dar bir yapısal
  cast (`CandlestickData[]`) tip güvenliğini korurdu. (Task 6 — Minor.)
- **`[candles]` effect chart'ı tümden yeniden kurar (Minor, deferred):** `PriceChartCanvas` her yeni
  `candles` referansında chart'ı `remove()` + `createChart` ile yeniden yaratıyor (WalletGraphCanvas
  konvansiyonuyla tutarlı). `candles` per-mint stabil olduğu için nadir; sık güncellemede `series.setData`
  ile in-place güncelleme perf iyileştirmesi olur. (Task 6 — Minor.)
- **Order-form hata `<span>`'leri `aria-describedby` taşımıyor (Minor, a11y):** Her alanın hata mesajı
  görsel var ama input'una `aria-describedby`/`role="alert"` ile bağlı değil; ekran okuyucu bağlamda
  duyurmaz. Erişilebilirlik follow-up'ı. (Task 8 — Minor.)
- **`sizePct`/`trailingPct` inert (kasıtlı, yorumlandı):** Order formunda "Pozisyon %" ve "Trailing %"
  alanları draft'a yazılıyor ama sizing/exit mantığına bağlı değil (gelecek artım). Kod yorumla işaretli;
  gerçek trade/otomatik-exit backend'i gelince devreye girecek. (Final review — kasıtlı.)
- **`onRowClick={() => {}}` no-op (kasıtlı, yorumlandı):** Terminal alt Pozisyonlar sekmesinde satır
  tıklaması detay drawer açmaz (terminal bağlamında bilinçle atlandı; drawer `/positions` ekranında var).
  Kod yorumla işaretli. (Final review — kasıtlı.)
- **`Txn.kind`/`Txn.status` inline union, `buildMarketData` tokens[0] fallback (Minor, plan-mandated):**
  `Order` named type'lar kullanırken `Txn`'de inline union; `buildMarketData` bilinmeyen mint'te reject
  yerine `tokens[0]`'a düşer (UI'da ulaşılmaz — activeMint hep geçerli). Backend'e geçişte hizalanabilir.
  (Task 1 — Minor, plan-mandated.)

## Backend Alt-proje 1 slice 1a — deferred

Slice 1a (`feat/backend-ingestion-1a`, 2026-08-05) teslim edildi; aşağıdaki maddeler bilinçle bu dilime
dahil edilmedi. Sessiz düşürme yok — deploy doğrulamasına ve Slice 1b'ye bağlı.

- **🔴 SÜREKLİ INGESTION AKIŞI — güvenilir WS sağlayıcısı gerekli (deploy'da bulundu, kod DEĞİL sağlayıcı).**
  Deploy'da pipeline kanıtlandı (gerçek pump.fun token'ları decode edilip DB'ye yazıldı) AMA **Helius free-tier
  standart `logsSubscribe` sürekli teslimat yapmıyor**: kısa bir başlangıç penceresinden sonra susuyor (worker
  heartbeat `alınan_30s=0` doğruladı, bağlantı düşmeden). Denenen ve **işe yaramayan** ücretsiz workaround:
  periyodik proaktif resubscribe (25s) — geri alındı (band-aid, kodu kirletiyordu). **Kalıcı çözüm (kullanıcı
  kararı, ertelendi):** Helius ücretli plan (Developer ~$49/ay, aynı kod) VEYA güvenilir free WS sağlayıcısı
  (Chainstack/QuickNode — WS/RPC URL config değişikliği). Worker kodu doğru; sadece `HELIUS_WS_URL`/`HELIUS_API_KEY`
  yerine güvenilir bir kaynak bağlanacak. Kullanıcı 2026-08-05'te "şimdilik böyle bırak" (D) dedi — demo'da
  gerçek token snapshot'ı var, canlı akış sağlayıcı kararına bağlı. Worker'da kalıcı **heartbeat** logu (`ingest
  heartbeat alınan_30s/işlenen_30s`) sağlayıcı sağlığını görünür kılar.

- **Raydium CPMM `initialize` account pozisyon-index kalibrasyonu:** Şu an "WSOL olmayan ilk base58-uzunluğunda
  account" heuristiği kullanılıyor; gerçek Raydium CPMM `initialize` tx'inde account sırasının kesin index'lerle
  doğrulanması gerekiyor. Deploy'da gerçek bir tx ile kalibre edilecek.
- **PumpSwap decoder** eksik (framework-ready — registry OCP ile eklenebilir). pump.fun graduation (PumpSwap'e
  geçen token) event'lerini yakalamıyor. Ayrıca `apps/web/lib/feed/sources.ts` LAUNCHPADS/DEXES listesine
  "PumpSwap" eklenmedi.
- **Moonshot / Meteora decoder'ları** eksik (framework-ready, aynı registry deseniyle eklenir).
- **Gerçek pump.fun fixture (kısmen kapandı):** Decoder **deploy'da gerçek Helius akışına karşı doğrulandı**
  (2026-08-05: 14 gerçek token/28 event, doğru mint/symbol/name). Regresyon için kayıtlı bir gerçek
  Solscan/Chainstack snapshot fixture'ı hâlâ nice-to-have; elle üretilen test vektörleri korunuyor.
- **pump.fun marker + Program data anchor'lama — ÇÖZÜLDÜ (2026-08-05, deploy hotfix `4e2396f`):** Kök neden
  deploy'da bulundu — `hasMarker` substring'i her buy'daki ATA `Instruction: CreateIdempotent` satırıyla
  eşleşip TradeEvent'i CreateEvent sanıp parse ediyordu (`kısa buffer`). Fix: create marker'ı **trimmed-suffix**
  ile birebir eşleştir (CreateIdempotent hariç) + CreateEvent byte offset'ini **otomatik tespit** et
  (emit! 8B / emit_cpi! 16B; u32 3-string + uri `://` + ardından ≥96 bayt doğrulaması, tüm `Program data:`
  satırlarını dener). Bundled create+buy artık doğru satırı seçer.
- **Worker dedup bounded/TTL değil:** `seen` map şu an sınırsız büyüyor (process ömrü boyunca temizlenmiyor).
  Ayrıca dedup, insert'ten ÖNCE "seen" işaretliyor — geçici bir insert hatası kalıcı skip'e yol açabilir
  (düşük etki; `logsSubscribe` replay yapmıyor).
- **Worker perf:** `RecentTokens` snapshot sorgusu ingest hot path'inde her decode edilen item için çalışıyor;
  ileride batch/debounce edilebilir.
- **Helius client:** `getTransaction` cevap zarfı (envelope) nil-check eksik; `SubscribeLogs`'ın spawn ettiği
  recvLoop goroutine'leri için `WaitGroup` yok; `tx==nil` typed-nil interface guard'ı eksik.
- **WS hub:** origin allowlist yok (şu an `InsecureSkipVerify: true` — 1a'da public read-only olduğu için kabul
  edilebilir); shutdown-sıralaması net değil (register/unregister blocking send'leri shutdown'a karşı
  `select`lenmiyor — process çıkışında reap ediliyor, 1a için yeterli); `json.Marshal` hata guard'ı yok.
- **Frontend `lib/api/ws.ts`:** reconnect-with-backoff + `onerror`/`onclose` **eklendi** (final review fix wave
  `f05b277`). Kalan: `components/terminal/MarketDataHeader.tsx`'te yerel bir `ScoreBadge` var; terminal canlıya
  geçince gözden geçirilmeli.
- **Slice 1b'ye ertelenen:** `getKpis`/`getRadar`/`getToken` gerçeğe dönecek; price/liquidity/vol5m/holders/
  momentum/spark zenginleştirmesi; `first_swap`/`liquidity_added` event tipleri.

## Backend Alt-proje 1 slice 1b — deferred

Slice 1b (`feat/backend-ingestion-1b`, 2026-08-06) teslim edildi: REST tabanlı token keşfi (GeckoTerminal
`new_pools`) + market enrichment (`pools/multi`), Helius WS'ten bağımsız. Aşağıdaki maddeler bilinçle bu dilime
dahil edilmedi. Sessiz düşürme yok — deploy doğrulamasına ve sonraki dilimlere bağlı.

- **Overview (`getKpis`/`getRadar`) → Alt-proje 2'ye bağlı.** İkisi de creator/safety skorlarını (KPI özetleri,
  fırsat radarı sıralaması) tüketiyor; skorlar Alt-proje 2 (Python scoring/ML) tamamlanana kadar nötr placeholder
  kalacağından bu dilimde gerçeğe döndürülmedi — Overview mock kalmaya devam ediyor.
- **Token Detail (`getToken`) + OHLCV serisi + Helius holders → Slice 1c. ✅ TESLİM EDİLDİ (2026-08-06).**
  `getToken` tekil token derinlemesine görünümü, OHLCV zaman serisi ve holder sayısı Slice 1c'de gerçeğe döndü —
  aşağıdaki "Backend Alt-proje 1 slice 1c — deferred" bölümüne bakın (scores/risks/davranış-metrikleri hâlâ
  Alt-proje 2'ye bağlı).
  - **Dürüst alan durumu (1b sonrası):** price/liquidity/vol5m/momentum/spark **GERÇEK** (GeckoTerminal); holders
    **boş**; creatorScore/safetyScore/signal **nötr placeholder** (0/"medium" — Alt-proje 2 dolduracak).
- **DexScreener ikinci provider — gerekince (OCP-hazır).** `internal/market/provider.go`'daki `MarketProvider`
  arayüzü (DIP) `GeckoTerminalClient`'ın tek implementasyonu; ikinci bir sağlayıcı (ör. GeckoTerminal rate-limit'e
  takılırsa ya da veri kalitesi/kapsamı için) mevcut arayüze uyan yeni bir client ile eklenebilir, `Discoverer`/
  `Enricher` değişmeden. Bugün somut ihtiyaç yok, framework hazır.
- **GeckoTerminal JSON alan-adı kalibrasyonu (deploy'da doğrulanacak):** Client `new_pools`/`pools/multi`
  cevaplarını JSON:API şemasına göre parse ediyor ancak gerçek alan adları (fiyat/likidite/hacim path'leri)
  yalnızca canlı GeckoTerminal cevabına karşı deploy'da kalibre edilecek — 1a'nın Raydium account-index
  kalibrasyonuyla aynı desen (placeholder değil, gerekçeli ertelenen madde).
- **Canlı GeckoTerminal + DB round-trip yalnızca deploy'da doğrulanacak** (yerel Postgres/ağ erişimi yok — 1a/1b
  ile aynı desen). Go build/vet/`test -race` yeşil; frontend hiç dokunulmadı (seam zaten alanları taşıyordu).
- **`launchpad` çapraz-yazar etkileşimi (WS dönünce, final review #3):** İki yazar aynı `tokens` satırını upsert
  edebilir — 1a Helius worker `UpsertToken` `launchpad`'i sabit `""` ile `ON CONFLICT SET` yapar; 1b Discoverer
  `UpsertDiscovered` gerçek launchpad + symbol/name yazar. Helius akışı bloke olduğu için şu an latent, ama
  güvenilir WS sağlayıcı gelince (aynı mint iki yoldan görülürse) biri diğerinin `launchpad`/symbol/name'ini
  boşa/eskiye ezebilir. Çözüm: WS dönünce yazarları uzlaştır (ör. non-empty alanı ezme / `COALESCE`). Sessiz
  düşürme yok — WS sağlayıcı işine bağlı, o zaman ele alınacak.

## Backend Alt-proje 2 slice 2a (Token Safety) — deferred

Slice 2a (`feat/backend-scoring-2a`, 2026-08-07) `tokenSafety` skorunu gerçeğe döndürdü: kural-tabanlı
açıklanabilir skor (launchpad-aware authority + top-10 holder yoğunlaşması + holder sayısı + likidite),
arka plan scorer + DB persist (Option A deseni), detail/liste DB'den okur. SDD 8 task + whole-branch review
(opus "With fixes") → 1 Important fix (holder-cap → confidence düşürür) + minor cleanup wave. Aşağıdakiler
bilinçle ertelendi (whole-branch review triage: hiçbiri merge'i bloke etmedi):

- **Skorlama coverage boşlukları (test):** Worker error-isolation (`FetchOnChain`/`UpdateSafety` `continue`
  yolları), `Run` ticker/cancel döngüsü, `SafetyScoreTargets`-fail — happy-path + partial testlerle kapsandı,
  kod inspection-doğru; `Score` holders-known/authorities-unknown ayna dalı ve provider partial-failure ters
  yönü test edilmedi. İleride hardening.
- **`isBondingCurve` case-sensitive** (`"Pump.fun"/"pump.fun"/"PumpSwap"` birebir) — üretici değeri sabit
  (`market/provider.go` → `discoverer.go` `"Pump.fun"` yazar), whole-branch review uçtan uca doğruladı → şu an
  güvenli; ama upstream değeri değişirse `strings.EqualFold` normalize gerekir.
- **Helius alan-şekli kalibrasyonu — ÇÖZÜLDÜ (2026-08-07, `fix/safety-holders-amount-shape` `933dda6`):**
  canlı doğrulama safety skorlarını `conf=0.5`/`top10=0` gösterdi → `getTokenAccounts` `amount` JSON SAYI
  döndürüyordu ama `tokenAccount.Amount` `string` idi → decode fail → HoldersKnown=false. Fix: `flexAmount`
  (sayı VEYA string tolere eder; null/bozuk→0), fixture gerçek şekle güncellendi (1a/1c deseni). `getAccountInfo`
  authorities şekli DOĞRUYDU (breakdown "Freeze authority iptal" gerçek). `tokenAmount.amount` iç içe DEĞİLDİ
  (top-level number). Merge+deploy sonrası conf→1.0/0.75, top10 dolu beklenir.
- **account-not-found → both-revoked:** `MintAuthorities` `value:null` (kapalı/yanmış hesap) → `(false,false,nil)`
  → provider `AuthoritiesKnown=true` + iki authority iptal = güvenli skorlanır. Keşfedilen token'lar on-chain var
  olduğundan nadir; parse "not-found"u "revoked"dan ayırt edemiyor.
- **`ParseFloat` hatası sessiz 0:** `holders.go` bozuk `amount` string'ini 0 sayar (data-quality); status!=200
  yolu ve JSON marshal-err (idiom) test edilmedi.
- **`Metrics.Top10HolderPct` koşulsuz:** skorlanmamış token `top10HolderPct:0` gösterir ("sağlıklı" gibi okunur,
  "bilinmiyor" değil) — metrik seam'inde confidence alanı yok; seam sınırlaması.
- **`safety.Holders` interface param adı `cap`:** somut impl `capN`'e döndü ama interface imzası `cap` kaldı
  (yalnız dokümantasyon, builtin-shadow yok) — kozmetik.
- **Kapsam dışı (sonraki 2a iterasyonu / diğer dilimler):** `creatorHoldingPct` (2b, creator adresi gerektirir),
  LP-burn/LP-lock tespiti (graduated token'lar), sniper%/bot%/buy-sell ratio davranış metrikleri (2c), `signal`
  (opportunity → 2d), diğer 3 skor (creatorReputation 2b / manipulationRisk 2c / opportunity 2d).

## Backend Alt-proje 1 slice 1c — deferred

Slice 1c (`feat/backend-ingestion-1c`, 2026-08-06) teslim edildi: Token Detail (`getToken`) gerçek — header
(fiyat/priceChange24h/marketCap/likidite/hacim24h) + yaşa-uyarlı OHLCV grafiği (GeckoTerminal) + holder sayısı
(Helius `getTokenAccounts`). **Alt-proje 1 (Solana ingestion) bu dilimle TAMAMLANDI (1a+1b+1c).** Aşağıdaki
maddeler bilinçle bu dilime dahil edilmedi. Sessiz düşürme yok — Alt-proje 2'ye ve deploy doğrulamasına bağlı.

- **Scores/risks/davranış-metrikleri → Alt-proje 2'ye bağlı.** `getToken`'in 4 `ScoreKey` skoru, `risks` listesi
  ve diğer davranış-metrikleri (creator/wallet güven skorlaması) nötr placeholder (`neutralScores` — 0/"medium")
  olarak kalmaya devam ediyor; Alt-proje 2 (Python scoring/ML) tamamlanana kadar 1c'de gerçeğe döndürülmedi —
  Overview'un `getKpis`/`getRadar`'ı ile aynı bağımlılık.
- **`series.liquidity` / `series.holders` (geçmiş seri) → ileride örnekleme.** OHLCV'den gelen `series.price`/
  `series.volume` gerçek; likidite ve holder sayısının **zaman içindeki** değişimi için ayrı bir örnekleme/kayıt
  mekanizması gerekiyor (şu an yalnızca anlık `metrics.holders` var, geçmişi yok) — henüz spec'lenmedi.
- **Holders "N+" gerçek-total gösterimi ertelendi (seam `number`, capped→floor).** `HOLDERS_CAP` (varsayılan
  5000) sınırına takıldığında `HoldersCount` sayımı durdurup cap değerini döner; bu sayı gerçek toplamın bir
  **alt sınırıdır** ("5000+" gibi), ama frontend seam'i (`TokenDetail.metrics.holders`) `number` tipinde —
  "N+" belirsizliğini taşıyacak bir alan (ör. `holdersCapped: boolean` ya da string varyantı) yok. Sessiz
  düşürme değil: sayı doğru bir alt sınır, sadece "capped" durumu UI'da ayırt edilemiyor. Seam genişlemesi
  gerektiriyor — ertelendi.
- **Bilinmeyen-mint pool keşfi → YAGNI.** `TokenDetailBase` yalnızca zaten DB'de bilinen (1a/1b ingestion'ının
  gördüğü) mint'leri çözer; DB'de olmayan rastgele bir mint için pool'u canlı keşfetmek (GeckoTerminal arama vb.)
  şu an somut ihtiyaç değil — bilinmeyen mint 404 döner (dürüst, silent-fail değil).
- **GeckoTerminal OHLCV + Helius `getTokenAccounts` alan-adı/sayfalama kalibrasyonu (deploy'da doğrulanacak):**
  `OHLCV` metodu GeckoTerminal'in mum verisini, `HoldersCount` ise Helius'un token-account sayfalarını
  şemaya göre parse ediyor; gerçek alan adları/sayfalama davranışı yalnızca canlı cevaba karşı deploy'da kalibre
  edilecek — 1a'nın Raydium account-index / 1b'nin GeckoTerminal `new_pools`/`pools/multi` kalibrasyonuyla aynı
  desen (placeholder değil, gerekçeli ertelenen madde).
- **Canlı GeckoTerminal OHLCV + Helius holders + DB round-trip yalnızca deploy'da doğrulanacak** (yerel
  Postgres/ağ/key yok — 1a/1b ile aynı desen). Go build/vet/`test -race` yeşil + frontend 190 test yeşil; yeni
  ücretli/harici bağımlılık yok (GeckoTerminal keysiz, holders için mevcut Helius key).
- **Final review minor'ları (opus, merge'e engel değil, ertelendi):** (1) `holders` = token-account sayısı,
  benzersiz-sahip DEĞİL (bir sahip birden çok ATA tutabilir; getTokenAccounts üst-ish yaklaşımı) — deploy
  kalibrasyonuyla birlikte değerlendir. (2) `TokenDetailService` cache: Important "sınırsız büyüme" FIX'LENDİ
  (write'ta TTL-expiry sweep, commit `f7e46f4`). **Header için ÇÖZÜLDÜ (2026-08-07, `feat/detail-header-from-db`):**
  header artık DB'den (enrichment persist etti) sunuluyor, canlı GeckoTerminal çağrısı yok → geçici throttle header'ı
  sıfırlayamaz, dolayısıyla cache de sıfır tutmaz. **Kalan (düşük öncelik):** yalnız OHLCV serisi canlı/best-effort —
  throttle'da boş dönebilir ve ~20s cache'lenir (dürüst; micro-cap'lerde zaten seyrek). Skorlar A2'ye kadar nötr. (3) ✅ **ÇÖZÜLDÜ
  (2026-08-07, `fix/gecko-rate-limit`):** keşif+enrichment+detail artık **tek paylaşılan token-bucket** (`Limiter`
  arayüzü + `WithLimiter` option → main.go'da bir `rate.NewLimiter` iki client'a) + `getJSON` 429'da backoff-retry;
  config `GECKOTERMINAL_RATE_PER_MIN`(25)/`GECKOTERMINAL_BURST`(2). Canlı doğrulamada bulunan aralıklı header-sıfır
  (429) kök nedeni buydu. (4) Nötr skorlar UI'da ekstrem renk gösterebilir (0=düşük/kırmızı, higherIsBetter=false
  için 100=yeşil); `confidence:0` "veri yok" sinyali — A2 gerçek skorları getirene kadar dokümante tasarım kararı.

## Backtesting (Increment 9)
- **Event Replay ertelendi (spec-level, sonraki artım):** Ekran 9'un look-ahead-bias'sız timeline playback
  yarısı bilinçle kapsam dışı bırakıldı (playback state'li oynatıcı + look-ahead engelleme ayrı bir etkileşim).
  Kaçırılan-fırsat/rug-timeline grafikleri, parametre preset kaydetme, sonuç export/karşılaştırma da ertelendi.
  Sessiz düşürme yok — `docs/progress.md` "Sırada" + spec §1'de işaretli. (Spec — deferred.)
- **`isError` dalı mock seam'de ulaşılamaz (ileri-dönük, parked):** `BacktestContent`'in "Backtest çalıştırılamadı"
  dalı `runBacktest` hiç throw etmediği için mock'ta ulaşılmaz (httpApi→`notReady` reject edince gerçek olur);
  strategies/tokens not-found dallarıyla tutarlı savunma kodu. Gerçek httpApi gelene kadar test edilmiş
  davranış değil. (Task 7 — parked.)
- **`BacktestParamsForm` hata `<span>`'leri `aria-describedby` taşımıyor (Minor, a11y):** 6 alanın hata mesajı
  görsel var ama input'una `aria-describedby`/`role="alert"` ile bağlı değil (Inc8 order-form ile aynı a11y
  follow-up'ı). Ekran okuyucu bağlamda duyurmaz. (Task 3 — Minor.)
- **Çalıştır butonu dar viewport'ta katlanma-altında (Minor, görsel doğrulama 2026-08-04):** Sol parametre
  paneli 10 alan + buton; 772px yükseklikte "Çalıştır" ilk ekranda görünüm-altında kaldı (kaydırma gerekti).
  Panel'i `sticky` buton footer'ı veya daha kompakt alan aralığıyla iyileştirilebilir. Kritik değil. (Görsel — Minor.)
- **Sonuç metrik grid'i geniş viewport'ta yatay taşabiliyor (Minor, görsel doğrulama 2026-08-04):** 10-tile
  grid + yan yana grafikler dar-ana-alanda yatay taşma yapabiliyor (sidebar açıkken 5-tile sırası görünüm
  dışına uzadı). Responsive kolon sayısı/`min-w-0` ile sıkılaştırılabilir. (Görsel — Minor.)
- **`BacktestMetrics` (bileşen) vs `BacktestMetrics` (tip) ad çakışması (kasıtlı, çözüldü):** Task 4 tipi
  `as Metrics` alias'ıyla import ederek çakışmayı çözdü — kalıcı; not olarak kayıt. (Task 4 — kasıtlı.)
- **`BacktestMetrics` tip / `BacktestSummary` DRY örtüşmesi (Minor, plan-mandated, parked):** İki özet tipi
  benzer alanlar taşıyor; httpApi'ye geçişte tek kaynaktan türetilebilir. Plan-mandated, düşük etki.
  (Task 1 — Minor, plan-mandated.)

## Backend Alt-proje 2 slice 2b-1 (Creator Capture) — deferred

Slice 2b-1 (`feat/backend-scoring-2b-1`, 2026-08-10) `getCreators`/`getCreator`'ı gerçeğe döndürdü:
pump.fun creator (dev) pubkey yakalama (migration 0006 + COALESCE merge), creator listesi/profili
(kimlik+totalTokens+firstSeen+token geçmişi). Aşağıdakiler bilinçle bu dilime dahil edilmedi. Sessiz
düşürme yok — 2b-2'ye ve deploy doğrulamasına bağlı.

- **İtibar/davranış/outcome → Alt-proje 2 Slice 2b-2'ye bağlı.** Reputation score, riskLevel, outcome
  tespiti (rugged/success/active), davranış paterni, `walletAgeDays`, peak/drawdown, `realizedPnlSol`
  — hepsi nötr placeholder kalıyor; gerçek outcome tespiti (fiyat/likidite geçmişine dayalı) ve itibar
  modeli 2b-2'de teslim edilecek.
- **`activeTokens`/`ruggedTokens`/`successRatePct` nötr 0.** 2b-2'nin outcome tespitine bağlı (bugün
  hiçbir token için outcome sınıflandırması yok).
- **`createdAt`/`firstSeen` ham ISO 8601 — sunum katmanına taşınacak (WalletAddress-truncation ile aynı
  sınıf).** Mock veri göreli format gösteriyordu ("1g önce"); gerçek seam ISO 8601 döndürüyor, frontend
  şu an ham ISO gösteriyor. Yapılacak: `lib/format.ts`'e bir `relativeTime()`/benzeri ekle, ilgili
  bileşenlerde kullan — `httpApi` adaptörüyle birlikte ele alınacak sunum-katmanı maddeleri grubunda
  (bkz yukarıda "HTTP adapter artımıyla birlikte").
- **`label` her zaman boş.** Creator etiketleme (ör. "bilinen rugger", "doğrulanmış") henüz tanımlanmadı;
  seam alanı var ama dolduran bir mekanizma yok.
- **Raydium/diğer launchpad token'ları creator'sız.** Yalnızca pump.fun `CreateEvent`'in `user` alanı
  yakalanıyor; Raydium CPMM (ve framework-ready PumpSwap/Moonshot/Meteora) decoder'ları creator
  taşımıyor — bu token'lar hiçbir creator'ın `history`'sinde görünmez.
- **Fake `Creators` tie-break insertion-order (brief-mandated, low impact):** Fake store eşit
  `totalTokens`'ta ekleme sırasına göre sıralar; postgres `MIN(first_seen_ts)` ile sıralar. İki store
  arasında görünüş farkı yaratabilir (bugün test kapsamında ulaşılmaz, brief'in kendi tasarımı).
  (Task 2 — Minor, deferred.)
- **postgres merge-kuralı + `CreatorDetail` canlı-DB testi yok (DB-gated env):** `UpsertToken`'in
  COALESCE creator-merge kuralı ve `CreatorDetail`'in postgres path'i yalnızca fake store'a karşı test
  edildi; gerçek Postgres'e karşı round-trip testi CI'da yok (yerel Postgres yok — 1a/1b/1c/2a ile aynı
  desen, deploy'da doğrulanacak). (Task 2 + Task 3 — Minor, deferred.)
- **postgres `CreatorDetail` iki ayrı query (non-transactional, düşük etki):** Kimlik/toplam sorgusu ve
  token geçmişi sorgusu ayrı çalışıyor; aralarında bir insert olursa (nadir, iki sorgu arası ms) hafif
  tutarsız bir görünüm mümkün. Kritik değil (read-only endpoint, sonraki poll düzeltir). (Task 3 —
  Minor, deferred.)
- **Router + config'te `CREATORS_LIST_LIMIT` default'u çift tanımlı (drift riski):** Default 100 hem
  config paketinde hem router wiring'inde ayrı yazılı; ileride biri değişip diğeri unutulursa sessiz
  drift olabilir. Tek kaynağa indirgenebilir (config her zaman router'a geçirilecek şekilde). (Task 4 —
  Minor, deferred.)

## Backend Alt-proje 2 slice 2b-2a (Token Outcome Detection) — deferred

Slice 2b-2a (`feat/backend-scoring-2b-2a`, 2026-08-10) 5'li outcome sınıflandırıcı (saf `internal/outcome/`) +
peak takibi (migration 0007) + arka plan `outcome.Worker` (Helius/ağ yok) teslim etti; `/api/creator/{address}`
token geçmişi artık gerçek `outcome`/`peakMarketCap`/`maxDrawdownPct`/`liquidityStatus` gösteriyor. Aşağıdakiler
bilinçle bu dilime dahil edilmedi. Sessiz düşürme yok — 2b-2b'ye, trade-flow dilimlerine (2c/2e) ve deploy
doğrulamasına bağlı.

- **İtibar skoru + metrikler + `walletAgeDays` + `creatorHoldingPct` → Alt-proje 2 Slice 2b-2b'ye bağlı.**
  Reputation score, riskLevel, ilgili metrikler ve creator'ın kendi holding yüzdesi nötr placeholder kalıyor;
  itibar modeli 2b-2b'de (muhtemelen Python) teslim edilecek.
- **`creatorSellPct` (per-token creator satış %) → trade-flow'a bağlı (2c/2e).** Bir creator'ın belirli bir
  token'daki satış yüzdesini hesaplamak trade-akışı (buy/sell event) verisi gerektiriyor; bu dilim yalnızca
  piyasa durumu (market cap/likidite) okuyor, trade-flow yok.
- **`behavior.*` → trade-flow + funding-graph'a bağlı (2c/2e).** Davranış paterni (sniper%/bot%/buy-sell
  ratio vb.) aynı şekilde trade-flow ve wallet-funding-graph verisine dayanıyor, bu dilimde toplanmıyor.
- **`liquidityStatus="locked"` kullanılmıyor.** Sınıflandırıcı şu an yalnız `"unlocked"`/muhtemelen
  `"unknown"` üretiyor (fake insert default'u da `"unlocked"`); on-chain LP-lock doğrulaması (LP token'ın
  kilitli/yakılmış olup olmadığı) henüz kapsam dışı — gelecekte eklenecek.
- **Gerçek pump.fun mezuniyet (Raydium migration) tespiti yerine marketCap-eşiği proxy kullanıldı.**
  `graduated` outcome'u gerçek bir Raydium migration event'i yerine `OUTCOME_GRADUATION_MCAP` marketCap
  eşiğine göre çıkarılıyor — 1a'nın decode edilmemiş PumpSwap graduation event'iyle aynı sınırlama sınıfı.
- **Peak seed conservative — migration'dan itibaren; gerçek tarihsel tepe kayıp.** Migration 0007 mevcut
  token'ların `peak_market_cap`/`peak_liquidity`'sini o anki **güncel** değerden seed'liyor (geçmişte
  gerçekleşmiş daha yüksek bir tepe varsa migration öncesi kayıp); migration sonrası peak'ler doğru şekilde
  yalnızca yükselir (`GREATEST`).
- **Outcome eşikleri deploy'da gerçek dağılıma göre kalibre edilecek (`OUTCOME_*`).** 6 eşik (rug likidite
  oranı, graduation marketCap, dumped drawdown, dead hacim/likidite tabanı, dead yaş) makul varsayılanlarla
  geldi ama canlı token dağılımına karşı yalnızca deploy sonrası kalibre edilebilir — kod değişmeden env ile
  ayarlanır (1a/1b/1c/2a'nın alan-şekli kalibrasyonlarıyla aynı desen, placeholder değil).
- **Fake `OutcomeTargets` insertion-order tie-break (brief-mandated, low impact):** `SafetyScoreTargets`'ı
  yansıtıyor — fake store eşit `outcome_scored_ts`'te ekleme sırasına göre sıralar, postgres gerçek
  `outcome_scored_ts ASC NULLS FIRST` (veya benzeri) ile sıralar. İki store arasında görünüş farkı
  yaratabilir (bugün test kapsamında ulaşılmaz). (Minor, deferred.)
- **Mixed-peak testi yok:** Bir token'ın market cap'i düşüp likiditesinin yükseldiği (ya da tersi) karışık
  senaryoda `UpdateMarket`'in her iki peak kolonunu bağımsız `GREATEST` ile doğru yükselttiğini doğrulayan
  bir test eklenmedi; mevcut testler tek-yönlü artış/azalışı kapsıyor. (Minor, deferred.)
- **`TestLoadOutcomeDefaults` env-hermetic değil:** Diğer config testleri gibi, çalıştırma sırasına/ortam
  değişkeni sızıntısına karşı tam izole değil (paralel test çalıştırmada teorik flake riski, bugün
  gözlemlenmedi). (Minor, deferred.)
- **DB-backed `CreatorDetail` entegrasyon testi yok:** Outcome/peak/drawdown/liquidityStatus alanlarının
  postgres path'inden okunduğu yalnızca fake store'a karşı test edildi; gerçek Postgres'e karşı round-trip
  testi CI'da yok (yerel Postgres yok — 1a/1b/1c/2a/2b-1 ile aynı desen, deploy'da doğrulanacak). (Minor,
  deferred.)

## Backend Alt-proje 2 — REST Creator Backfill — deferred

`feat/backend-creator-backfill` (2026-08-10, HEAD `371b7ce`) 1b'den beri bilinen WS blokörünü
(Helius free-tier `logsSubscribe` kesintiye uğruyor) baypas etti: GeckoTerminal-keşifli
creator'sız pump.fun token'larına creator artık REST ile (`getSignaturesForAddress` en-eski +
`getTransaction` + decode reuse) backfill ediliyor. Aşağıdakiler bilinçle bu dilime dahil
edilmedi. Sessiz düşürme yok — deploy doğrulamasına ve sonraki dilimlere bağlı.

- **Etki (canlı veri kazanımı):** Creator artık WS'e bağımlı değil — GeckoTerminal-keşifli
  pump.fun token'ları için **2b-1 creator alanları** (adres/totalTokens/token geçmişi) ve
  **2b-2a outcome alanları** (`CreatorDetail.history`) canlı veri alır; **2b-2b (itibar skoru,
  henüz teslim edilmedi)** de bu creator'lar üzerinden gerçek veri görecek.
- **Non-pump.fun token'lar hâlâ creator'sız.** Yalnızca pump.fun `CreateEvent`'in `user` alanı
  decode ediliyor; Raydium CPMM (ve framework-ready PumpSwap/Moonshot/Meteora) launchpad'lerinde
  keşfedilen token'lar bu dilimle de creator kazanmadı — 2b-1'deki aynı sınırlama.
- **Cap'e takılan / decode-fail token'lar `creator=''` + damgalı kalır ama HÂLÂ target'tır
  (de-prioritized round-robin re-attempt, kalıcı hariç tutma DEĞİL).** `CREATORFILL_MAX_SIG_PAGES`
  sınırına takılıp en-eski imzaya ulaşılamayan ya da create tx'i bulunup da decode edilemeyen
  (bozuk/beklenmedik log/tx-retention) token'lar boş creator ile damgalanır ama `CreatorFillTargets`'ta
  kalmaya devam eder — `ORDER BY creator_backfill_ts ASC` onu yalnızca sıranın sonuna iter,
  hariç tutmaz. Creator'sız backlog `CREATORFILL_LIMIT`'ten (20) küçükse bu token her ~30s
  cycle'da tekrar çözülmeye çalışılır (cycle başına sınırlı: ≤maxSigPages `getSignatures` +
  1 `getTransaction`) — `SafetyScoreTargets`/`OutcomeTargets` ile aynı round-robin konvansiyonu.
  Bu, geçici RPC hatalarını kurtarır ama immutable-creator not-found'ların düşük frekansta
  tekrar tekrar denenmesi anlamına gelir (dürüst, silent-fail değil; deploy'da gerçek dağılımla
  ne sıklıkta olduğu görülecek).
  **Deploy kaldıracı:** ücretsiz-katman Helius RPC bütçesi dar çıkarsa, sorgu `CreatorFillTargets`'a
  `AND creator_backfill_ts=0` ekleyerek attempt-once'a sıkılaştırılabilir (bu durumda geçici
  hatalar kalıcı hale gelir — kabul edilebilir bir trade-off). Deploy-kalibrasyon opsiyonu olarak
  kayıtlı, şu an uygulanmadı.
- **RPC free-tier limiti gözlenirse `CREATORFILL_*` ile kod değişmeden kısılır.** `_INTERVAL_SEC`/
  `_LIMIT`/`_MAX_SIG_PAGES` — 1a/1b/1c/2a/2b-2a'daki aynı "kalibrasyon env, placeholder değil"
  deseni.
- **DAS `getAsset` creator yolu elendi (spec kararı, kod yok).** Ucuz/tek-çağrılık alternatif
  değerlendirildi ama pump.fun mint'leri için on-chain creator alanını doldurmuyor (yalnız
  CreateEvent log'unda var) — bu yüzden `getSignaturesForAddress`+`getTransaction` yoluna
  gidildi.
- **Task 1-5 ertelenen minor'lar (taşındı):**
  - Boş-creator + başarılı-parse dalı (create tx bulundu, decode başarılı ama creator alanı boş
    çıktı) test edilmedi — teorik, decoder'ın kendisi boş `user` üretmiyor.
  - Çoklu `Program data:` satırlı tx'ler (birden fazla event log'u) test kapsamı dışında —
    tekli-event fixture'larla doğrulandı.
  - `TransactionSignature.Err` alanı okunmuyor/atlanıyor (dokümante edilmiş kasıtlı sınırlama) —
    başarısız (revert) create tx'i de "bulundu" sayılabilir; nadir, kritik değil.
  - Fake `CreatorFillTargets` insertion-order tie-break (brief-mandated) — `SafetyScoreTargets`/
    `OutcomeTargets` ile aynı desen; postgres `creator_backfill_ts ASC NULLS FIRST` (veya benzeri)
    ile sıralar, fake store yalnız ekleme sırasına göre — bugün test kapsamında ulaşılmaz.
  - Yeni store metodları (`CreatorFillTargets`/`SetCreatorBackfill`) için canlı-DB parity testi
    yok (yerel Postgres yok — 1a/1b/1c/2a/2b-1/2b-2a ile aynı desen, deploy'da doğrulanacak).
  - `creatorfill.Worker.Run()` için ayrı bir unit test yok (ticker/cancel döngüsü) — safety/outcome
    worker'larıyla aynı ertelenen kapsam boşluğu.

### Fix: creator-backfill Helius rate-limit / 429-retry (fix/creatorfill-helius-rate-limit)

- **Retry mekanizması duplikasyonu (whole-branch review bulgusu, ERTELENDİ).** `ingest.withRetry`
  + `crBackoff` + `crMaxAttempts` + `sleepCtx` + `Limiter` arayüzü, `market/geckoterminal.go`'daki
  `getJSON` + `backoff` + `gtMaxAttempts` + `sleepCtx` + `Limiter`'ın neredeyse birebir kopyası.
  Tek gerçek fark retryable-yordam (HTTP status==429 vs `is429` ibare-eşleşmesi). Ortak bir
  `internal/retry` helper'ı (isRetryable(err) predicate ile parametrik) ikisini de kapsardı.
  **Neden ertelendi:** çıkarmak canlı GeckoTerminal yolunu (market paketi) da değiştirir →
  bugfix branch kapsamını/riskini büyütür. Ayrı refactor branch'inde ele alınacak.
- **`is429` HTTP-durum tabanlı değil, string-eşleşme.** solana-go `rpc.Client` durum kodunu
  yapılandırılmış vermediğinden hata metnindeki tam `"status code: 429"` ibaresine bakılıyor
  (dar eşleşme, false-positive'i eler). solana-go mesaj formatı değişirse kırılır — düşük risk,
  deploy'da 429 retry davranışı doğrulanır.
- **Kalibrasyon (deploy'da):** `CREATORFILL_RATE_PER_MIN`(120=2/s)/`CREATORFILL_BURST`(2) Helius
  free-tier gerçek RPS'ine göre ayarlanabilir; 2/s hâlâ 429 alırsa düşür (backoff-retry emniyet
  ağı var). Diğer Helius worker'ları (safety/holders) aynı key bütçesini paylaşır ama ayrı/ılımlı
  — ileride tüm Helius çağrılarını tek limiter bütçesine bağlamak potansiyel iyileştirme.
