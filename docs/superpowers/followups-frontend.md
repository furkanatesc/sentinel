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
