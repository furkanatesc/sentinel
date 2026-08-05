# SENTINEL — Alınacak API Key'ler / Hesaplar (checklist)

> ⚠️ **GÜVENLİK — ÖNEMLİ:** Bu dosya **public GitHub repo**'sundadır. **Buraya, repoya veya sohbete GERÇEK KEY YAZMA.**
> Key **değerleri** yalnızca **Railway** (backend servisi → Variables) ve **Vercel** (frontend → Environment Variables)
> panellerine, ilgili servisin secret store'una girilir. Repoya commit edilmez, log'a yazılmaz, chat'e yapıştırılmaz.
> Bu dosya sadece **hangi platformlardan ne alınacağının** listesidir (değer içermez).

Sıra, backend programının alt-projelerine göre. **Şimdi gereken: sadece Helius** (Alt-proje 1). Diğerleri ilgili alt-proje gelince.

---

## 1. Helius — İMPLEMENTASYON TAMAM, deploy'da key gir (Backend Alt-proje 1 slice 1a: Solana ingestion)

- **Ne için:** Gerçek-zaman Solana verisi — WebSocket `logsSubscribe` (yeni token/mint tespiti) + DAS API (token metadata/holder).
- **Hesap:** https://www.helius.dev → Sign up → Dashboard.
- **Alınacak:** bir **API Key** (dashboard'da "API Keys" bölümü). Ayrıca RPC/WS URL'i key'i içerir (ör. `https://mainnet.helius-rpc.com/?api-key=...` ve `wss://mainnet.helius-rpc.com/?api-key=...`).
- **Nereye:** **Railway → `api-go` servisi → Variables** → `HELIUS_API_KEY` = (key değeri).
  **⚠️ Key değeri SADECE Railway Variables paneline girilir — repoya, bu dosyaya veya sohbete (chat'e) ASLA
  yazılmaz/yapıştırılmaz.** Sohbete bir şekilde sızan bir key varsa Helius dashboard'undan **rotate/iptal**
  edilip yenisi Railway'e girilmeli (deploy adımı — bkz `docs/progress.md` "Sırada" bölümü, deploy DUR-noktası).
- **Tier/maliyet:** Ücretsiz tier ile başlanabilir; sürekli WebSocket + hacim artınca **ücretli plan** gerekebilir (rate limit/WS bağlantı sınırları). Free ile başla, gerekince yükselt.
- **Kod durumu:** ✅ **İmplementasyon TAMAM** (branch `feat/backend-ingestion-1a`, worker + WS hub + `/api/events`/`/api/tokens`/`/ws` endpoint'leri hazır; `HELIUS_API_KEY` yoksa ingestion worker başlamaz ama REST endpoint'leri yine çalışır).
- **Durum:** 🔶 **Deploy bekliyor** — kullanıcı Railway → `api-go` servisi → Variables → `HELIUS_API_KEY` girecek (deploy adımı, whole-branch review + master merge sonrası).

---

## 2. Telegram Bot — SONRA (Backend Alt-proje 3: Alerts / Telegram)

- **Ne için:** Gerçek Telegram bildirimleri gönderen bot.
- **Hesap/alınacak:** Telegram'da **@BotFather** ile konuş → `/newbot` → bot adı + kullanıcı adı ver → sana bir **bot token** verir.
- **Nereye:** **Railway → `api-go` servisi → Variables** → `TELEGRAM_BOT_TOKEN` = (token).
- **Maliyet:** Ücretsiz.
- **Durum:** ⬜ Bekliyor (Alt-proje 3'te)

---

## 3. LLM sağlayıcı (opsiyonel) — İLERİDE (Alt-proje 2 scoring/RAG veya Research ekranı)

- **Ne için:** Açıklanabilir skor/özet, RAG, araştırma asistanı (yalnız o özellikler geldiğinde).
- **Hesap/alınacak:** Anthropic (console.anthropic.com) veya OpenAI — bir **API key**.
- **Nereye:** **Railway → ilgili servis → Variables** → (ör. `ANTHROPIC_API_KEY`).
- **Maliyet:** Kullanım başına ücretli. **Şimdilik gerekmez.**
- **Durum:** ⬜ İleride değerlendirilecek

---

## 4. Trading (Jupiter + cüzdan) — EN SON (Backend Alt-proje 5: Trading engine)

- **Jupiter Swap API:** genelde **key gerektirmez** (public API); yüksek hacimde ücretli tier olabilir.
- **İşlem cüzdanı:** gerçek emir için fonlanmış bir Solana cüzdanı + private key gerekir. **⚠️ Private key ASLA repoya/chat'e girmez ve ben private key ile işlem yapmam** (güvenlik kuralı); bu, secret store + backend imzalama ile, ayrı ve dikkatli tasarlanır. **Tasarım paper-trading varsayılanı.**
- **Durum:** ⬜ En son (gerçek para; ayrı güvenlik tasarımı)

---

## Zaten kurulu (aksiyon gerekmez)
- **GitHub:** `furkanatesc/sentinel` (public repo).
- **Vercel:** frontend deploy (`sentinel-brown-alpha.vercel.app`), env'ler ekli.
- **Railway:** backend servisi (`sentinel-production-e14d.up.railway.app`) + Postgres eklentisi.

---

### Özet — şu an senden istenen tek şey
Helius implementasyonu (madde 1) **kod tarafında tamam**. Deploy aşamasında (branch review + master merge
sonrası): eğer key sohbette paylaşıldıysa önce **rotate/iptal** et, sonra Railway → `api-go` servisi →
Variables → `HELIUS_API_KEY` = (taze key değeri). Key değerini bana verme — sadece Railway paneline gir.
