# SENTINEL — UI/UX Tasarım Spec'i

> Kaynak: Figma Make dosyası "Film-Festival-Schedule" (fileKey `zAWGuUwmbKkSK0n7YKlANt`) içindeki
> `src/imports/pasted_text/sentinel-ui-ux-design.md`. Bu dosya, ROADMAP.md ile birlikte frontend'in
> tek tasarım gerçek kaynağıdır. Figma Make ayrıca çalışan bir referans implementasyonu içerir
> (dark tema token'ları, app shell, Overview dashboard, Sentinel primitive'leri, shadcn/ui seti).

Solana ekosisteminde yeni çıkan coinleri gerçek zamanlı tespit eden, token geliştiricilerini geçmiş projelerine göre güven skorlamasına tabi tutan, risk analizi yapan, Telegram bildirimleri gönderen ve otomatik alım-satım işlemleri gerçekleştirebilen profesyonel bir web uygulaması için kapsamlı bir desktop-first UI/UX tasarımı.

Ürün; crypto trader'lar, quant kullanıcılar, araştırmacılar ve yüksek riskli yeni token piyasalarını takip eden ileri seviye kullanıcılar için tasarlanmalıdır.

## Ürün konumlandırması

Ürün adı: "Sentinel".

Sentinel, Solana üzerinde yayınlanan yeni tokenları saniyeler içinde tespit eden; tokenı çıkaran creator wallet'ı, bağlantılı cüzdanları, geçmiş token performansını, likiditeyi, holder dağılımını, akıllı kontrat yetkilerini ve piyasa hareketlerini analiz eden gerçek zamanlı bir token intelligence ve automated trading platformudur.

Ana ürün akışı: **Discover → Analyze → Score → Alert → Trade → Monitor**

## Genel tasarım yaklaşımı

Modern, yüksek güven veren, veri yoğun fakat anlaşılır bir fintech ve crypto trading arayüzü.

Görsel dil:
- Dark mode ana tema
- Profesyonel trading terminal hissi
- Temiz, keskin ve yüksek kontrastlı tasarım
- Gereksiz neon, aşırı glow ve oyunlaştırılmış crypto görsellerinden kaçın
- Bloomberg Terminal, Linear, Vercel, Stripe Dashboard, TradingView ve modern institutional trading platformlarından esinlen
- Kurumsal, teknik ve premium görünüm
- Bilgiler kolay taranabilir
- Kritik riskler renk, ikon ve hiyerarşiyle hızlı anlaşılır
- Monospace font yalnızca cüzdan adresleri, token mint adresleri, transaction hash ve teknik metriklerde

## Renk sistemi (Sentinel dark theme token'ları)

Referans implementasyonundaki `.dark` token'ları:
- `--background: #080B12`
- `--foreground: #E6EAF2`
- `--card / surface-1: #111722`
- `--popover / surface-2: #151C28`
- `--secondary / accent / surface-3: #1A2331`
- `--primary (accent mor): #7C5CFF`
- `--muted-foreground / neutral: #8A94A6`
- Durum renkleri: positive `#2FD98B`, warning `#FFB020`, critical/destructive `#F0476B`, info/accent-blue `#3E9BFF`
- `--border: rgba(255,255,255,0.07)`
- Chart paleti: `#7C5CFF, #2FD98B, #3E9BFF, #FFB020, #F0476B`

Renkler erişilebilir kontrasta sahip olmalı. Yalnızca renkle değil ikon, label ve metinle de bilgi ver.

## Tipografi

- Başlıklar: Inter / Geist gibi modern sans-serif
- Teknik değerler: JetBrains Mono / IBM Plex Mono
- Hiyerarşi: büyük dashboard başlıkları, orta section başlıkları, küçük fakat okunabilir tablo/metric metinleri; yoğun veride kontrollü compact tipografi

## Ana layout

Desktop-first. Ana ekran genişliği 1440px, 12 kolon grid, 24px dış boşluk, 16–24px component spacing.

Layout:
1. Sol sabit (daraltılabilir) sidebar
2. Üst global header
3. Ana içerik alanı
4. Sağda isteğe bağlı canlı activity / alert paneli

## Sidebar navigasyonu

Bölümler: Overview, Live Feed, Discover, Tokens, Creators, Wallet Graph, Smart Wallets, Strategies, Positions, Orders, Portfolio, Backtesting, Alerts, Telegram, Research, System Health, Settings.

Sidebar altı: RPC status, Solana network durumu, Telegram bağlantı durumu, aktif trading mode, kullanıcı profili.

Trading mode görünür: **Paper / Shadow / Live**. Live mode kırmızı veya amber güvenlik etiketiyle belirtilmeli.

## Global header

Global token/wallet/transaction arama alanı, Solana network seçici, son veri güncelleme zamanı, RPC latency, gas/priority fee göstergesi, notification ikonu, Emergency Pause butonu, kullanıcı profili.

Arama placeholder: "Token, wallet, creator veya transaction ara".

## Ekran 1: Overview Dashboard

KPI kartları: son 24 saatte tespit edilen tokenlar, yüksek güven skorlu tokenlar, kritik risk tespitleri, aktif sinyaller, açık pozisyonlar, günlük realized PnL, günlük unrealized PnL, sistem latency.

Her KPI kartında: ana değer, önceki döneme göre değişim, mini sparkline, tooltip, son güncelleme zamanı.

### Live Token Feed
Gerçek zamanlı yeni token tablosu. Kolonlar: Token, Yaş, Fiyat, Likidite, İlk 5dk hacmi, Holder sayısı, Creator Score, Token Safety, Momentum, Risk seviyesi, Sinyal, Hızlı aksiyonlar. Her satırda token logosu, ad/symbol, kısaltılmış mint adresi, yaş, skor badge'leri, mini fiyat grafiği, watchlist ikonu, Analyze, Trade. Riskli satırlarda görsel uyarı (tabloyu tamamen kırmızıya boyamadan).

### Opportunity Radar
Scatter/bubble chart: X = Creator Trust Score, Y = Momentum Score, bubble büyüklüğü = likidite, bubble rengi = risk seviyesi. İnteraktif ve filtrelenebilir.

### Alerts Timeline
Sağ panelde canlı alert akışı: yeni token, ilk likidite, creator satışı, likidite çekildi, whale alımı, risk skoru değişti, trade emri gerçekleşti. Her event zaman damgası + severity.

## Ekran 2: Live Feed

Gerçek zamanlı event terminali. Üst filtreler: event type, launchpad, DEX, min liquidity, min creator score, risk level, token age, volume, holder growth, watchlist only.

Event tipleri: New Mint, Metadata Created, Pool Created, First Swap, Liquidity Added, Liquidity Removed, Creator Sell, Whale Buy, Suspicious Cluster, Score Change, Strategy Signal. Event açılınca sağdan detay drawer.

## Ekran 3: Token Detail

Header: token logosu, ad, symbol, mint adresi, yaş, fiyat, market cap, likidite, 24s hacim, fiyat değişimi, Watchlist, Telegram alert, Simulate Trade, Buy, Sell.

Dört ana skor (0–100): Opportunity Score, Creator Reputation, Token Safety, Manipulation Risk. Yanında risk seviyesi, confidence, son hesaplanma zamanı, "Why this score?".

Tablar: Overview, Market, Holders, Creator, Wallet Graph, Transactions, Risk Analysis, Social, Strategy Signals, Audit Log.

Overview tabı: fiyat/likidite/hacim grafikleri, holder büyümesi, buy/sell oranı, unique buyer, creator holding oranı, top10 holder oranı, sniper oranı, bot activity oranı.

Risk Analysis: Contract Risk (mint/freeze authority, mutable metadata, token-2022 hooks, transfer fee), Market Risk (düşük likidite, price impact, konsantre holder, wash trading), Creator Risk (rug bağlantıları, başarısız token, bağlantılı satışlar, şüpheli fon). Her risk: severity, açıklama, kanıt, ilgili transaction, ilk/son görülme.

Explainable Score paneli: skorun kanıta dayalı breakdown'u.

## Ekran 4: Creator Profile

Header: kısaltılmış wallet address, wallet yaşı, ilk görülme, Creator Reputation Score, risk etiketi, Watch Creator, Telegram alarmı.

Metrikler: toplam token, aktif token, rug işaretli token, ortalama token ömrü, ortalama peak market cap, creator realized PnL, başarı oranı, ortalama satış zamanı.

Token geçmişi tablosu + creator davranış paterni (çıkarma sıklığı, ilk satış süresi, tekrarlanan funder, benzer metadata, aynı sosyal hesap, aynı likidite davranışı).

## Ekran 5: Wallet Graph

İnteraktif graph visualization. Node türleri: Creator wallet, Funding wallet, Token, Liquidity pool, Trader wallet, Smart wallet, Suspicious wallet, Exchange wallet. Edge türleri: FUNDED, CREATED, BOUGHT, SOLD, TRANSFERRED, PROVIDED LIQUIDITY, REMOVED LIQUIDITY, SHARES FUNDER, CONTROLS AUTHORITY.

Özellikler: zoom, pan, node expand, time range filter, relationship filter, risk filter, cluster highlight, path finder, wallet compare, export image, export data. Sağ panelde seçilen node detayı. Varsayılan olarak yalnızca en önemli bağlantılar gösterilir.

## Ekran 6: Strategies

Strateji kartları: name, status, mode, timeframe, win rate, profit factor, max drawdown, total trades, net PnL, son sinyal zamanı. Durumlar: Draft, Backtesting, Paper, Shadow, Live, Paused, Archived.

Detay: strategy logic, entry/exit conditions, risk rules, position sizing, supported launchpads, minimum scores, backtest results, live performance, version history, audit log.

Create Strategy stepper: 1) details 2) universe 3) entry rules 4) exit rules 5) risk 6) position sizing 7) backtest 8) deploy. No-code condition builder.

## Ekran 7: Trading Terminal

Layout: sol token listesi/watchlist, orta fiyat grafiği/market data, sağ order paneli, alt orders/positions/transactions/logs.

Order paneli: Buy/Sell, Market/Limit, SOL/USDC miktarı, position size %, slippage, priority fee, stop-loss, take-profit, trailing stop, estimated price impact, minimum received, route provider, simulation status.

Emir öncesi confirmation modal (token, işlem tipi, tutar, tahmini fiyat, slippage, price impact, risk score, creator score, açık riskler, wallet balance, expected fees). Live trading'de güçlü güvenlik uyarısı.

## Ekran 8: Portfolio ve Positions

Portfolio: total value, available balance, invested, realized/unrealized/daily PnL, max drawdown, risk exposure, rug exposure. Grafikler: equity curve, PnL by strategy/creator score/token age, risk allocation, win/loss distribution.

Positions tablosu: token, strategy, entry, current price, position size, PnL, stop-loss, take-profit, token risk, creator risk, age, actions.

## Ekran 9: Backtesting

Parametreler: strategy, date range, initial capital, max positions, slippage model, priority fee, latency model, liquidity constraints, token/creator filters.

Sonuç metrikleri: net PnL, win rate, profit factor, Sharpe, Sortino, max drawdown, average trade, rug exposure, trade sayısı, ortalama holding süresi. Grafikler: equity curve, drawdown, monthly return, trade distribution, PnL by score, entry/exit noktaları, missed opportunity.

Event Replay: token oluşturulmasından işlem kapanışına tüm olaylar timeline üzerinden oynatılabilir (look-ahead bias engellenir).

## Ekran 10: Alerts ve Telegram

Alert formu: name, token/creator scope, event trigger, min liquidity, min creator score, max risk, holder growth, creator sale, liquidity removal, whale activity, strategy signal, delivery channel. Kanallar: Web, Telegram, Email, Webhook.

Telegram ekranı: bot connection status, chat ID, test notification, notification severity, quiet hours, alert templates, trade approval settings. Telegram notification preview bileşeni.

## Ekran 11: Research Assistant

AI destekli araştırma paneli. Örnek sorular: bu token neden riskli, creator geçmişi özeti, wallet cluster şüpheli bağlantıları, benzer skorlu tokenların performansı, sinyal neden üretildi, trade neden kapatıldı. Cevaplarda kaynak: transaction hash, wallet, token, risk rule, strategy version, timestamp. AI sonuçları "informational analysis" olarak etiketlenir; trade kararından ayrıştırılır.

## Ekran 12: System Health

Servisler: Solana RPC, event ingestion, scoring engine, wallet graph, database, Redis, trading engine, Telegram, Jupiter, WebSocket connections. Metrikler: status, latency, error rate, events/sec, queue lag, last successful message, RPC rate limit, transaction success rate, failed transaction count. Kritik system alert görünür.

## Component sistemi

Buttons, inputs, selects, multi-select, search, tabs, tables, data cards, metric cards, score badges, risk badges, token avatars, wallet address, copy-to-clipboard, tooltips, toasts, drawers, modals, confirmation dialogs, command palette, charts, graph nodes, timeline, activity feed, empty states, loading skeletons, error states.

## Score componentleri

Seviyeler: 0–24 Critical, 25–49 High Risk, 50–69 Medium, 70–84 Good, 85–100 Strong. Componentte: sayısal skor, risk seviyesi, progress ring/horizontal meter, trend, confidence, tooltip, explain action. Yalnızca renk değil metin etiketi de.

## Table UX

Sorting, filtering, column customization/resizing, sticky header, row selection, bulk actions, saved views, pagination/virtual scroll, export CSV, empty/loading state. Adreslerde: kısaltılmış adres, copy, external explorer, hover preview.

## Responsive

Desktop 1440, laptop 1280, tablet 768, mobile 390. Mobilde tam terminal yerine portfolio, alerts, token detail, position management, emergency pause, telegram settings önceliklenir.

## Güvenlik UX'i

Live trading açma onayı, max günlük zarar limiti, max position size, max slippage, emergency pause, withdraw/trading yetki ayrımı, Paper/Live görsel ayrımı, kritik işlemlerde ikinci onay, işlem öncesi simulation. Live trading aktifken uygulama genelinde ince ama görünür durum göstergesi.

## Onboarding

1) Ürün tanıtımı 2) Solana wallet bağlantısı 3) Telegram bot 4) risk profili 5) paper trading seçimi 6) ilk alert 7) ilk watchlist 8) dashboard turu. Varsayılan mod **Paper Trading**; kullanıcı doğrudan live'a yönlendirilmez.

## Öncelik sırası

1. Hızlı karar verme
2. Risklerin görünürlüğü
3. Güvenilir ve açıklanabilir skorlar
4. Gerçek zamanlı veri takibi
5. Güvenli trading deneyimi
6. Yoğun veriye rağmen yüksek okunabilirlik
7. Profesyonel institutional görünüm
