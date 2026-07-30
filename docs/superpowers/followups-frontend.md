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
