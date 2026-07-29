Temel ürün döngüsü şu:

Discover → Enrich → Score → Alert → Decide → Execute → Observe → Learn

1. Yeni token keşif motoru

Buradaki kritik ayrım şu: Bir tokenın mint edilmesi, onun gerçekten işlem yapılabilir olduğu anlamına gelmez. Sistem farklı olayları ayrı ayrı izlemeli:

Yeni SPL Token / Token-2022 mint oluşturulması
Metadata oluşturulması veya değiştirilmesi
İlk likidite havuzunun açılması
İlk swap işlemi
Bonding curve başlangıcı veya tamamlanması
DEX migrasyonu
Likidite ekleme ve çekme olayları
Mint authority ve freeze authority değişiklikleri
Token arzındaki anormal artışlar

Gerçek zamanlı keşif için Solscan’i tek başına ana veri kaynağı yapmam. Solscan Pro API, geçmiş veri ve zenginleştirme için değerlidir; ancak canlı olay yakalamada Solana WebSocket abonelikleri veya özel stream sağlayıcıları daha uygun olur. Solana’nın RPC katmanı logsSubscribe, programSubscribe ve benzeri PubSub yöntemlerini destekliyor. Solscan ise token, hesap, program ve piyasa analitiği gibi işlenmiş veriler için kullanılabilir.

İzlenecek programlar başlangıçta configurable olmalı:

Token Program
Token-2022 Program
Metaplex metadata
Pump.fun ve benzeri launchpad programları
Raydium
Meteora
Orca
Jupiter ile ilişkili swap akışları
Daha sonra ortaya çıkan yeni launchpad ve AMM programları

Yeni programları kod deploy etmeden sisteme eklemek için Program Registry özelliği tasarlanabilir.

2. Token Intelligence Profile

Her token için tek bir normalize edilmiş profil üretmeliyiz:

TokenProfile
├── mint_address
├── token_program
├── name / symbol / metadata_uri
├── creator_wallet
├── funding_wallet
├── creation_time
├── first_pool_time
├── first_trade_time
├── total_supply
├── holder_count
├── liquidity
├── volume
├── price
├── authorities
├── pool_information
├── holder_distribution
├── wallet_cluster_information
├── social_information
├── security_flags
└── computed_scores

Bu profil yalnızca güncel snapshot olmamalı. Zaman serileri de tutulmalı:

İlk 10 saniye / 1 dakika / 5 dakika hacmi
Holder büyüme hızı
Benzersiz alıcı sayısı
Likidite değişimi
Top 10 holder oranı
Creator satış oranı
Buy/sell dağılımı
Ortalama işlem büyüklüğü
Bot ve insan cüzdanı oranı
Token yaşam döngüsü

Bu sayede yalnızca “şu anda iyi görünüyor” değil, “hangi yönde değişiyor?” sorusunu da cevaplarız.

3. Creator ve wallet güven skoru

En değerli feature’lardan biri bu olur. Ancak yalnızca mint authority adresini “creator” saymak yetersizdir. Gerçek aktörü anlamak için şu ilişkileri çözmek gerekir:

Mint işlemini imzalayan adres
İşlem ücretini ödeyen adres
Creator’ı fonlayan adres
İlk likiditeyi sağlayan adres
Metadata update authority
İlk token dağıtımını yapan adresler
Aynı fon kaynağını kullanan diğer creator’lar
Token açıldıktan sonra satış yapan bağlantılı cüzdanlar
Örnek Creator Trust Score
Creator Trust Score = 0–100

Geçmiş token performansı           %20
Rug / liquidity removal geçmişi    %25
Bağlantılı cüzdanların davranışı   %15
Creator'ın satış davranışı         %10
Holder dağıtım kalitesi            %10
Cüzdan yaşı ve işlem geçmişi        %5
Fonlama kaynağı güvenilirliği       %5
Tekrarlanan deploy paterni         %10

Skorun yanında mutlaka açıklanabilir nedenler gösterilmeli:

Creator Score: 27/100 — Yüksek Risk

- Son 12 tokenın 8'i 24 saat içinde %90'dan fazla değer kaybetti
- 6 projede likidite aynı wallet cluster tarafından çekildi
- Creator, arzın %18'ini bağlantılı cüzdanlara dağıttı
- Fonlayan adres daha önce 14 düşük ömürlü token finanse etti

Bu, kullanıcının kör şekilde tek bir sayıya güvenmesini engeller.

Ayrı skorlar kullanmak daha doğru

Tek bir “güven skoru” yerine:

Creator Reputation Score
Token Safety Score
Market Quality Score
Momentum Score
Manipulation Risk Score
Execution Risk Score

üretilmeli.

Sonrasında ihtiyaca göre birleşik skor hesaplanabilir:

Opportunity Score =
0.30 × Market Quality
+ 0.25 × Momentum
+ 0.25 × Creator Reputation
+ 0.20 × Token Safety

Buradaki ağırlıklar stratejiye göre değişebilir.

4. Wallet clustering ve Knowledge Graph

Knowledge graph bu projede gerçekten anlamlı; fakat chatbot knowledge graph’ından ziyade önce on-chain entity graph olarak ele alınmalı.

Node tipleri
Wallet
Token
Transaction
Liquidity pool
Program
Launchpad
Social account
Domain
Funding source
CEX deposit/withdrawal address
Edge tipleri
CREATED
FUNDED
TRANSFERRED_TO
BOUGHT
SOLD
PROVIDED_LIQUIDITY
REMOVED_LIQUIDITY
SHARES_FUNDER_WITH
INTERACTED_WITH
CONTROLS_AUTHORITY
PROMOTED_BY

Örnek:

Wallet A
  ├── FUNDED → Wallet B
  ├── FUNDED → Wallet C
  └── FUNDED → Wallet D

Wallet B ── CREATED → Token X
Wallet C ── CREATED → Token Y
Wallet D ── CREATED → Token Z

Böyle bir durumda B, C ve D farklı creator adresleri görünse bile aynı aktöre ait olma ihtimali vardır.

İlk aşamada Neo4j zorunlu değil. PostgreSQL üzerinde adjacency tabloları ve recursive query’lerle başlanabilir. Grafik sorguları karmaşıklaştığında Neo4j, Memgraph veya Amazon Neptune değerlendirilebilir.

5. Rug ve manipülasyon tespit özellikleri

Sistem yalnızca klasik kontrolleri değil, davranış tabanlı riskleri de yakalamalı.

Statik riskler
Mint authority aktif
Freeze authority aktif
Değiştirilebilir metadata
Token-2022 transfer fee veya transfer hook
Anormal decimals/supply yapısı
Çok yüksek top-holder yoğunluğu
Creator’ın yüksek token payı
Düşük veya kilitsiz likidite
Sahte veya tekrar kullanılan metadata
Metadata ile on-chain özelliklerin uyuşmaması
Davranışsal riskler
Creator’ın küçük parçalara bölerek satış yapması
Bağlantılı cüzdanlar arasında token döndürülmesi
Wash trading
Aynı bloklarda koordineli alım
Bundle veya sniper yoğunluğu
Holder sayısının yapay büyütülmesi
Çok sayıda cüzdanın aynı fonlayıcıdan SOL alması
Likidite çekme hazırlığı
Birbirine bağlı adreslerin arzı dağıtılmış gibi göstermesi
Tekrarlanan pump-and-dump şablonu

2026’da yayımlanan SolRugDetector çalışması da Solana’daki rug davranışlarının çoğu zaman yalnızca kötü niyetli kontrat kodundan değil, on-chain piyasa manipülasyonu ve cüzdan davranışlarından anlaşılması gerektiğini vurguluyor. Bu nedenle graph ve zaman serisi analizi projenin merkezinde olmalı.

6. Telegram bot özellikleri

Telegram yalnızca bildirim kanalı değil, operasyon arayüzü olabilir.

Bildirimler
🚨 Yeni Token Tespit Edildi

Token: EXAMPLE
Mint: Abc...xyz
Yaş: 14 saniye
Likidite: $18.400
Creator Score: 82/100
Token Safety: 74/100
Top 10 Holder: %21
Creator Holdings: %1,8
İlk 30 sn Hacim: $9.200

Riskler:
• Metadata değiştirilebilir
• İlk 20 alıcının 6'sı aynı funder ile bağlantılı

[Solscan] [Web Panel] [Detay] [Simüle Et]
Bot komutları
/token <mint>
/wallet <address>
/creator <address>
/watch <address>
/unwatch <address>
/positions
/pnl
/risk
/strategies
/pause
/emergency_close
/explain <signal-id>
Telegram üzerinden trade

Telegram üzerinden doğrudan trade yapılabilir; ancak:

İlk sürümde tek tıklamayla doğrudan market buy yerine onay ekranı
Maksimum pozisyon limiti
Maksimum slippage
Günlük zarar limiti
Token başına risk limiti
İşlem öncesi simulation
Yetki seviyeleri
Withdraw izninden ayrı trading wallet

kullanılmalı.

7. Trading engine

Trading engine’i keşif ve analiz servislerinden kesin biçimde ayırmak gerekir.

Emir özellikleri
Market buy/sell
Limit order
Take-profit
Stop-loss
Trailing stop
Time-based exit
Partial take-profit
DCA entry/exit
Liquidity-based exit
Creator-sale-triggered exit
Risk-score-triggered exit
Volatility halt
Emergency close

Jupiter, swap ve diğer DeFi işlevleri için resmi API’ler sunuyor. Swap işlemlerinde rota oluşturma ve işlem instruction’larını alma katmanında kullanılabilir. Ancak gönderim, confirmation ve retry mekanizması ayrıca tasarlanmalıdır.

İşlem güvenlik katmanı

Her trade’den önce:

Güncel token durumu tekrar okunur.
Quote alınır.
Price impact ölçülür.
Transaction simulate edilir.
Token ve creator skoru tekrar kontrol edilir.
Pozisyon limitleri kontrol edilir.
Transaction gönderilir.
Confirmation izlenir.
Gerçekleşen fiyat ve fee kaydedilir.
Devre kesiciler
Günlük maksimum zarar
Ardışık başarısız işlem sayısı
RPC gecikmesi
Fiyat kaynağı uyuşmazlığı
Anormal slippage
Ağ congestion
Stratejinin beklenen ve gerçekleşen performansı arasındaki sapma
Private key/KMS erişim anomalisi
8. Strateji platformu

Stratejileri doğrudan backend koduna gömmek yerine standart bir contract tanımlayalım:

class Strategy:
    def evaluate(self, context) -> Decision:
        ...

class Decision:
    action: BUY | SELL | HOLD | REJECT
    confidence: float
    position_size: float
    stop_loss: float | None
    take_profit: list[TakeProfitLevel]
    reasons: list[str]

context içerisinde:

Token snapshot
Creator score
Wallet graph
Son işlemler
Holder değişimi
Likidite
Momentum metrikleri
Mevcut pozisyon
Portföy riski
Piyasa durumu

bulunur.

Strateji türleri
İlk likidite momentum stratejisi
Güvenilir creator stratejisi
Bonding curve graduation stratejisi
Smart-wallet takip stratejisi
Volume breakout
Liquidity acceleration
Holder growth
Creator exit detection
Mean reversion
Copy-trade
Ensemble strategy

Başlangıçta LLM’in doğrudan al/sat kararı vermesini önermem. LLM:

Sinyali açıklayabilir
Araştırma özeti çıkarabilir
Risk nedenlerini insan diline çevirebilir
Strateji loglarını analiz edebilir
Yeni kural önerileri üretebilir

Ancak execution kararı deterministic kurallara, doğrulanmış modellere ve risk engine’e bağlı kalmalı.

9. Backtesting, replay ve paper trading

Bu bölüm olmadan strateji geliştirmek büyük ölçüde tahmine dönüşür.

Gerekli modlar:

Historical backtest
Event replay
Paper trading
Shadow trading
Live trading
A/B strategy comparison

Özellikle event replay önemli:

2026-07-20 13:41:10 → Token oluşturuldu
2026-07-20 13:41:14 → İlk pool açıldı
2026-07-20 13:41:16 → Sinyal üretildi
2026-07-20 13:41:17 → Quote alındı
2026-07-20 13:41:18 → İşlem simüle edildi
2026-07-20 13:41:19 → Buy gerçekleştirildi

Sistem geçmiş olayları aynı sırayla yeniden oynatarak stratejiyi o anda sahip olduğu veriyle test etmeli. Gelecekteki verinin yanlışlıkla modele sızması, yani look-ahead bias, engellenmeli.

Performans metrikleri:

Win rate
Expectancy
Profit factor
Maximum drawdown
Sharpe/Sortino
Slippage-adjusted return
Latency-adjusted return
Rug exposure rate
Missed opportunity rate
False-positive rate
Strategy başına PnL
Creator-score segmentlerine göre PnL
10. RAG, CAG, knowledge graph ve “loop engineering”

Bu teknolojilerin her biri farklı yerde değer üretir.

RAG

Şunları sorgulamak için:

Strateji dokümantasyonu
Geçmiş trade açıklamaları
Post-mortem kayıtları
Güvenlik kuralları
Token analiz raporları
Protokol dokümantasyonu

Örnek soru:

“Bu token neden reddedildi ve geçmişte aynı risk kombinasyonuna sahip tokenlar nasıl performans gösterdi?”

CAG

Sabit ve sık kullanılan bilgileri model context’inde/cache’te tutmak için:

Risk politikaları
Skor açıklamaları
Strateji tanımları
Sistem sözlüğü
Telegram cevap şablonları

Fakat on-chain gerçek zamanlı veri CAG içine sabitlenmemeli; tool/data query ile alınmalı.

Knowledge graph

Creator, funder, token, pool ve wallet cluster bağlantılarının analizi için RAG’den daha merkezi bir rol oynar.

Loop engineering

Şu kontrollü döngü kurulabilir:

Observe
→ Generate signal
→ Validate
→ Execute or reject
→ Measure outcome
→ Attribute success/failure
→ Update feature statistics
→ Propose strategy revision
→ Backtest revision
→ Human approval
→ Deploy

“Model kendi stratejisini otomatik değiştirip canlıya alsın” yaklaşımı ilk aşamada tehlikeli olur. Model önerir; değişiklik backtest, shadow mode ve insan onayından geçer.

11. Önerdiğim MVP kapsamı

İlk sürümü fazla genişletmeden şu sekiz özelliğe indirgerdim:

Yeni token ve ilk likidite tespiti
Token güvenlik kontrolleri
Creator geçmişi ve temel reputation score
Basit wallet funding graph
Telegram gerçek zamanlı bildirim
Web dashboard ve token detay sayfası
Paper trading
Event storage ve replay altyapısı

İkinci faz:

Otomatik trade
Gelişmiş wallet clustering
Smart-money analizi
Backtesting framework
Strateji marketplace/registry
Portfolio risk engine
RAG destekli araştırma asistanı

Üçüncü faz:

ML tabanlı rug ve opportunity scoring
Graph embeddings
Online learning
Multi-agent analiz
Kullanıcı bazlı strateji optimizasyonu
SaaS abonelikleri ve çoklu tenant
API erişimi
12. Go + Python ayrımı

Hybrid yaklaşım burada mantıklı.

Go
Solana event ingestion
WebSocket/RPC bağlantıları
Transaction dispatcher
Telegram event handling
API gateway
Position/risk kontrolleri
Düşük gecikmeli worker’lar
Rate limiting
Retry ve idempotency
Python
Feature engineering
Creator scoring
Wallet clustering
ML modelleri
Backtest
RAG ve LLM servisleri
Graph analytics
Research notebooks
Strategy experimentation

Aralarında başlangıçta Kafka kurmak zorunlu değil. AWS üzerinde:

Solana Streams
      ↓
Go Ingestion
      ↓
Kinesis / MSK / SQS
      ↓
Normalization Workers
      ↓
PostgreSQL + Redis + S3
      ↓
Python Scoring
      ↓
Signal Service
      ↓
Telegram / Web / Trading Engine

MVP’de maliyeti ve operasyon yükünü azaltmak için SQS + PostgreSQL + Redis yeterli olabilir. Yoğun event hacmine ulaşıldığında Kinesis veya MSK’ya geçilir.

En kritik ürün kararı

İlk ürünün ana vaadini şu şekilde belirlerdim:

Yeni Solana tokenlarını saniyeler içinde tespit eden, tokenı çıkaran aktörün geçmişini ve bağlantılı cüzdanlarını analiz eden, açıklanabilir risk skoru üreten gerçek zamanlı istihbarat ve işlem platformu.

İlk başarı metriği “kaç token buldu?” değil:

Tokenı ne kadar erken buldu?
Rug’ların yüzde kaçını filtreledi?
İyi fırsatların yüzde kaçını kaçırdı?
Sinyal üretim gecikmesi neydi?
Skorlar gerçekleşen sonuçlarla ne kadar korelasyon gösterdi?
Slippage ve işlem masrafı sonrası strateji pozitif expectancy üretti mi?

olmalı.