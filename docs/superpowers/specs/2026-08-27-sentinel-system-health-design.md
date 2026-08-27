# SENTINEL — System Health (Dev/Ops Teşhis Paneli) — Tasarım

**Tarih:** 2026-08-27
**Kapsam:** Frontend `system-health` ekranı (şu an `PlaceholderScreen`) + backend `internal/health` telemetri katmanı + `GET /api/system-health`. Greenfield (hem frontend hem backend sıfırdan).
**Bağlam:** Graph işlerinin çoğu veri-sağlayıcı kararına bloklu; DAS=Helius-paid seçildi ama admin adımı ertelendi → **veri-blokusuz D pivotu** olarak System Health seçildi (tek harici-bağımlılıksız ekran).
**Bağımlılık:** **YOK** — yeni key/ücret/harici hesap gerektirmez. Dahili durum okur. Public endpoint (secret sızdırmaz).

---

## 1. Amaç

Pipeline'ın canlı sağlığını tek ekranda görünür kılmak — özellikle **WS-dormant / seyrek-veri** durumunu teşhis etmek. Şu an worker'larda hiç canlılık/durum izleme yok; "veri neden gelmiyor?" sorusu ancak Railway loglarından el yordamıyla cevaplanıyor. Bu ekran şu ayrımı net yapar: bir worker **kapalı mı** (env gate off), **hiç çalışmadı mı**, **stalled mı** (takıldı), yoksa **degraded mı** (ör. 429 alıyor).

İkincil değer: A kararını (Helius paid'e geçmeli mi?) **veriyle** destekler — safety worker'ın `getTokenAccounts` 429 alıp almadığı panelde `degraded` + `lastErr` olarak görünür.

**Kapsam dışı (v1, YAGNI):** WS push (REST poll yeterli), tarihsel trend/grafik, alert eşiği/bildirim (Alerts işi), auth/token guard (public, secret sızmaz).

---

## 2. Kapsam kararları (brainstorming'de onaylı, 2026-08-27)

| Karar | Seçim | Gerekçe |
|-------|-------|---------|
| Amaç/derinlik | **Dev/ops teşhis paneli** | Detaylı iç durum; cilalı kullanıcı-durum-sayfası değil |
| Erişim | **Public, secret sızdırmaz** | Mevcut `/api/*` deseni; gösterilenler secret değil; ekstra secret yönetimi yok (ertelenen admin adımları gibi yük eklemez) |
| Tazeleme | **REST poll** | Mevcut tokens/kpis/radar deseni; snapshot yeterli |
| Toplama modeli | **Merkezi registry (push) + istek-anı probe (hibrit)** | Worker liveness'i (asıl ihtiyaç) doğru yakalayan tek yaklaşım; SOLID sınırları net |

---

## 3. Mimari — `internal/health` paketi

### 3.1 Registry (push, thread-safe)
`Registry`, worker adı → `Status` eşlemesini `sync.RWMutex` altında tutar. İki yazma yolu:

- **`Register(name string, enabled bool, interval time.Duration)`** — startup'ta her worker (açık VEYA kapalı) kendini kaydeder. Kapalı worker `enabled=false` ile kaydolur → panelde "off" ayrımı mümkün. `registeredAt=now` saklanır (starting-grace türetimi için, §3.3).
- **`Report(name string, ok bool, err error, processed int)`** — worker `Run` döngüsünün her cycle sonunda çağırır. `lastRunAt=now`, `cyclesRun++`, `itemsProcessed += processed`, `ok=false` ise `lastErr` sanitize edilip saklanır.

### 3.2 Reporter — dar arayüz (ISP/DIP)
Worker'lar registry somut tipini değil, dar bir arayüzü alır:

```go
type Reporter interface {
    Register(name string, enabled bool, interval time.Duration)
    Report(name string, ok bool, err error, processed int)
}
```

`Registry` bu arayüzü karşılar. Worker'lar birbirini ve endpoint'i bilmez (gevşek bağlılık). Test'te `Reporter` kolayca stub'lanır.

### 3.3 Snapshot (okuma)
`Snapshot() Report` — endpoint'in okuduğu, kilit **dışında** serialize edilen kopya. State türetimi snapshot anında yapılır (saf fonksiyon, `now` parametreli → test edilebilir). **5 durum**, bu sırayla değerlendirilir:

- `off` — `enabled=false`.
- `starting` — `enabled=true` && `cyclesRun==0` (henüz ilk cycle bitmemiş) && grace içinde: `interval==0` **veya** `now - registeredAt ≤ 3 × interval`. (Yeni başlamış worker'ın yanlışlıkla `stalled` görünmesini engeller.)
- `stalled` — `enabled=true` && `interval>0` && grace/tazelik aşıldı:
  - hiç çalışmadı: `cyclesRun==0` && `now - registeredAt > 3 × interval`, **veya**
  - çalıştı ama takıldı: `cyclesRun>0` && `now - lastRunAt > 3 × interval`.
- `degraded` — `enabled=true` && `cyclesRun>0` && son cycle `ok=false`.
- `ok` — diğer tüm açık+taze durumlar.

**`interval==0` (event-driven) worker'lar** (ör. `ingest-ws`): zaman-tabanlı stall UYGULANMAZ (periyodik değiller). İlk Report'a kadar `starting`, sonra `ok`/`degraded`. Bu worker'ların "dormant"lığı panelde `lastRunAt` yaşından (ör. "son olay 2 sa önce") + `itemsProcessed=0`'dan **insan tarafından** okunur — v1'de otomatik dormant-alarm yok (YAGNI).

### 3.4 Best-effort garanti
`Report`/`Register` **asla** worker'ı bloklamaz veya panik etmez. Registry nil-guard'lıdır: worker'a `Reporter` enjekte edilmezse (veya nil) çağrılar no-op (worker health-izlemeden bağımsız çalışmaya devam eder — mevcut davranış korunur, OCP).

---

## 4. Toplanan sinyaller (v1)

### 4.1 Worker başına (`WorkerStatus`)
`name`, `state` (`off|starting|ok|degraded|stalled`, §3.3), `lastRunAt` (RFC3339 veya boş), `lastErr` (kısa, sanitize), `cyclesRun`, `itemsProcessed`, `intervalSec`.

**Kapsanan worker'lar (main.go envanteri):** `ingest-ws` (WS-dormant kaynağı), `market-disc`, `market-enrich`, `safety`, `outcome`, `creatorfill`, `funder` (walletgraph), `reputation`, `manipulation`, `opportunity`. Hub ayrı raporlanmaz (bkz §4.3 dbOk yanında `wsClients`).

### 4.2 İstek-anı probe
`dbOk bool` + `dbLatencyMs int` — endpoint DB'ye hafif bir ping atar (`SELECT 1` / store'da `Ping(ctx)` seam). Fake store → `dbOk=true, latency=0` (dürüst; in-memory). Ping başarısız → `dbOk=false`, endpoint yine **200** döner.

### 4.3 Global
`uptimeSec` (process başlangıcından), `version` (git commit — env `RAILWAY_GIT_COMMIT_SHA` okunur, yoksa `"dev"`; build-time ldflags gerektirmez, basit), `wsClients` (`hub.ClientCount()`), `gates` (env-gate özeti: `MARKET_ENABLED`, `SAFETY_ENABLED`, `OUTCOME_ENABLED`, `CREATORFILL_ENABLED`, `WALLET_GRAPH_ENABLED`, `REPUTATION_ENABLED`, `MANIPULATION_ENABLED`, `OPPORTUNITY_ENABLED`).

---

## 5. Endpoint — `GET /api/system-health`

`router.go`'da mevcut `/api/*` grubuna eklenir. Handler: registry `Snapshot()` + DB ping + hub client sayısı + config gate'lerini birleştirip JSON döner.

```json
{
  "uptimeSec": 12345,
  "version": "2004972",
  "dbOk": true,
  "dbLatencyMs": 8,
  "wsClients": 0,
  "workers": [
    {"name":"ingest-ws","state":"starting","lastRunAt":"","lastErr":"","cyclesRun":0,"itemsProcessed":0,"intervalSec":0},
    {"name":"safety","state":"degraded","lastRunAt":"2026-08-27T10:00:00Z","lastErr":"getTokenAccounts: status 429","cyclesRun":42,"itemsProcessed":120,"intervalSec":30}
  ],
  "gates": {"MARKET_ENABLED":true,"SAFETY_ENABLED":true,"WALLET_GRAPH_ENABLED":false}
}
```

Handler deps: `health.Registry` (Snapshot), `store` (Ping), `ws.Hub` (ClientCount), `config` (gates), başlangıç zamanı + version. `router.Deps`'e eklenir.

---

## 6. Frontend

- `app/(app)/system-health/page.tsx`: `PlaceholderScreen` → gerçek panel.
- **Hibrit getApi seam** (mevcut desen): `contract.ts`'e `SystemHealth` tipi; `mock.ts`'e temsili mock; `http.ts`/getApi `/api/system-health`'e bağlanır; `queries.ts`'e `useSystemHealth` (React Query, ~10 sn poll).
- **UI (mevcut tasarım-dili, yeni bileşen icat etmeden):**
  - Üst şerit: DB durumu (ok/latency), uptime, version, wsClients.
  - Worker tablosu: ad · state rozeti (yeşil `ok` / mavi `starting` / sarı `degraded` / kırmızı `stalled` / gri `off`) · "son çalışma: x sn önce" · lastErr (varsa) · cyclesRun/itemsProcessed.
  - Gate özeti şeridi (hangi worker'lar açık).
- Mevcut UI bileşen kütüphanesi/tokenları kullanılır; UI dokunulmazlığı korunur (yeni ekran içeriği, mevcut shell/nav zaten `system-health` route'unu taşıyor).

---

## 7. Hata yönetimi & güvenlik

- **Best-effort registry (§3.4):** worker sağlığı health-izlemeden bağımsız; Report asla bloklamaz/panik etmez.
- **Graceful degrade:** DB ping fail → `dbOk:false`, HTTP **200** (500 değil — dürüst degraded, panel yine yüklenir).
- **Secret sızmaz (public endpoint garantisi):**
  - RPC URL / API key **asla** snapshot'a girmez.
  - `lastErr` **sanitize** edilir: `?api-key=...` ve benzeri query-string kırpılır (registry `Report` içinde, tek yerde). Test bunu doğrular.
  - Gösterilen her alan sağlık durumu — kimlik/secret değil.

---

## 8. Test (TDD)

**`internal/health`:**
- Registry eşzamanlılık (`-race`): paralel Report/Register + Snapshot.
- Snapshot izolasyonu (kopya, kilit dışı).
- State türetimi: off/starting/ok/degraded/stalled sınır durumları (interval eşiği, grace, zero-time, interval==0 event-driven).
- `lastErr` sanitize: `?api-key=SECRET` içeren hata → kırpılmış.
- Nil/no-op reporter davranışı.

**`internal/api`:**
- Handler JSON şekli (workers, gates, dbOk).
- `dbOk=false` path'i (200 döner, degraded gösterir).

**Frontend:**
- `mock.ts` seam tipi + `mock.test.ts` şekil doğrulaması.

---

## 9. Uygulama sırası (plan'a girdi)

1. `internal/health` paketi: `Registry` + `Reporter` + `Snapshot` + state türetimi + sanitize (TDD).
2. Worker'lara `Reporter` enjeksiyonu (dar arayüz) + `Register`/`Report` çağrıları — birer birer, mevcut testler yeşil kalarak (nil-guard sayesinde enjeksiyon opsiyonel).
3. Store `Ping(ctx)` seam (postgres + fake).
4. `GET /api/system-health` handler + `router.Deps` + route.
5. `cmd/server/main.go` wiring: registry oluştur, worker'lara geçir, handler deps'e bağla, version/başlangıç-zamanı.
6. Frontend: contract tip + mock + getApi + query + panel ekranı.
7. Whole-branch review + merge.

**Kapsam işareti:** WS push, trend grafiği, alert bildirimi v1 dışı (§1). Bunlar ertelendi, sessiz düşürülmedi.
