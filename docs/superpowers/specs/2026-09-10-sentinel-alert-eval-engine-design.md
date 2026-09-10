# Alarm Değerlendirme Motoru — Tasarım (Spec)

> Backend Alt-proje 3'ün üçüncü dilimi. Entegrasyon-gerektirmez (mevcut Postgres event akışı).
> Kararlar: periyodik worker + watermark; v1 trigger kapsamı = veri-mevcut olanlar (kullanıcı onayı 2026-09-10).

## Amaç

`/alerts` geçmişini gerçek yapmak: aktif alarm kurallarını mevcut event akışıyla eşleştiren bir
değerlendirme motoru + üretilen alarmların kalıcılığı + `getAlerts`'i mock'tan canlıya çevirme.
**Gerçek Slack/email teslimatı HARİÇ** (harici entegrasyon → sona ertelendi). "web" kanalı =
`/alerts` geçmişinde görünme.

## Bağlam (mevcut kod)

- `getAlerts` şu an `notReady` (http) → **mock-only**; backend'de alarm-geçmişi/değerlendirme kodu YOK.
- `EventRow` (append-only `events` tablosu, `RecentEvents(limit)`) değerlendirme için gereken tüm alanları
  taşır: `Type` (= AlertTriggerType), `Symbol`, `Mint`, `Launchpad`, `Liquidity`, `CreatorScore`,
  `RiskLevel`, `Severity`, `Detail`, `Ts`.
- `AlertRule` (0016): `Trigger`, `Scope`, `MinLiquidity`, `MinCreatorScore`, `MaxRisk`, `Channels`, `Enabled`.
- `AlertEvent` (frontend `types.ts`): `{ id, type, token, detail, severity, time }`.
- Worker deseni (`internal/trend`): immediate cycle + ticker, dar `Sampler` DIP arayüzü, `health.Reporter`
  nil-güvenli, config-gated. Yeni worker bunu birebir izler.

## Mimari & veri akışı

Yeni `internal/alerteval` paketi + worker. Config-gated: `ALERTEVAL_ENABLED` (+ `ALERTEVAL_INTERVAL_SEC`,
varsayılan ör. 30). Her cycle:

1. `GetWatermark()` → son işlenen event ts'i (yoksa 0).
2. `RecentEvents(limit)` → newest-first; watermark'tan **yeni** olanları (ts > watermark) al, eskiden-yeniye sırala.
3. Aktif kuralları (`ListAlertRules` → `Enabled` olanlar) her yeni event'le eşleştir (`matchRule`).
4. Eşleşen her (event, rule) için bir `AlertEvent` üret + `InsertAlertEvent`.
5. `SetWatermark(maxTs)` — işlenen en yeni event ts'ine ilerlet.

**Watermark = dedup mekanizması:** her event tam bir kez değerlendirilir; tekrar tetiklenme yok.
Restart-dayanıklı (watermark DB'de tek-satır meta). Bir event birden çok kurala eşleşebilir → birden
çok alarm (beklenen).

Health: worker `health.Reporter` ile `ok/cyclesRun/itemsProcessed` raporlar (System Health paneli görür).

## Eşleştirme — saf fonksiyon

`func matchRule(e store.EventRow, r store.AlertRule) bool` (saf, tablo-testli):

1. **Trigger:** `r.Trigger == e.Type` **ve** trigger v1-kapsamında (`new_mint`, `liquidity_added`,
   `liquidity_removed`). Kapsam-dışı trigger → false (asla eşleşmez).
2. **Likidite:** `e.Liquidity >= r.MinLiquidity`.
3. **Üretici skoru:** `e.CreatorScore >= r.MinCreatorScore`.
4. **Risk tavanı:** `riskRank(e.RiskLevel) <= riskRank(r.MaxRisk)` (ör. low<medium<high<critical).
5. **Kapsam:** `r.Scope` boş ya da "Tüm tokenlar" → hepsi; aksi halde `e.Launchpad` ile case-insensitive eşleşme.

**v1-kapsam registry'si** (kapsam-dışı trigger'lar açıkça işaretli — sessiz düşürme yok):
`evaluableTriggers = {new_mint, liquidity_added, liquidity_removed}`. holder_growth/whale_activity/
score_change/creator_sale/strategy_signal → veri gelince eklenecek (followup).

Üretilen `AlertEvent`: `type = e.Type`, `token = e.Symbol` (boşsa `e.Mint`), `severity = e.Severity`,
`detail = r.Name + " — " + e.Detail`, `time = e.Time` (event'in mevcut formatlı string'i yeniden kullanılır —
locale/format karmaşası yok), `id` = üretilen (ör. `a` + UnixNano ya da rule_id+event_id kompoziti).

## Kalıcılık

`migration 0018_create_alert_events.sql`:
- `alert_events` (id TEXT PK, rule_id TEXT, type TEXT, token TEXT, detail TEXT, severity TEXT, time TEXT, ts BIGINT).
- `alert_eval_meta` (id TEXT PK 'default', watermark_ts BIGINT) — tek-satır watermark.

`AlertEventStore` (DIP; postgres + fake):
- `InsertAlertEvent(ctx, AlertEventRow) error`
- `RecentAlertEvents(ctx, limit) ([]AlertEventRow, error)` (newest-first)
- `GetAlertWatermark(ctx) (int64, error)` / `SetAlertWatermark(ctx, ts) error`

`AlertEventRow` (Go) frontend `AlertEvent` ile birebir JSON (`time` hariç türetme handler'da ya da row'da).
Append-only; retention/prune v1'de YOK → followup. Bundle.AlertEvents + OpenPostgres migration (seed yok).

## API / frontend

- `GET /api/alerts` → `RecentAlertEvents(limit)` → `AlertEvent[]`. RouterDeps.AlertEvents + route (nil→atla).
- http.ts: `getAlerts` notReady → `getJson<AlertEvent[]>("/api/alerts")`. LIVE_ENDPOINTS'e `getAlerts`.
- **Frontend değişmez:** `AlertHistoryPanel` zaten `getAlerts` (useAlerts) tüketiyor; yalnız veri kaynağı
  mock→canlı. Mock `getAlerts` korunur (hibrit modda çalışır).
- Boş geçmiş: canlı modda ilk açılışta `[]` (henüz eşleşme yok) — panel "kayıt yok" durumu zaten var.

## Test

- Saf `matchRule` tablo-testi: her filtre (trigger/likidite/skor/risk/kapsam) + kapsam-dışı trigger + scope varyantları.
- `riskRank` testi.
- Fake `AlertEventStore`: insert/recent (newest-first) + watermark get/set.
- Worker bir-cycle testi (enjekte fake event+rule): alarm üretir + watermark ilerler; **ikinci cycle tekrar üretmez** (dedup).
- Handler testi (`GET /api/alerts` 200 + şekil).
- `go build/vet/test ./... -race` + frontend tsc/vitest/build yeşil; mevcut testler kırılmaz.

## Global kısıtlar

- Clean/SOLID: dar DIP arayüzleri (Sampler-benzeri), saf `matchRule` (SRP, test-edilebilir), worker orkestrasyon.
- Config-gated + nil-güvenli health; diğer worker'larla aynı iskelet.
- Commit sonu: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.

## Kapsam dışı (followups)

Gerçek Slack/email teslimatı (sona); minSeverity delivery gating; kapsam-dışı 5 trigger (veri gelince);
alarm-geçmişi retention/prune; kural-bazlı dedup penceresi (şu an event-bazlı, doğru ve yeterli);
alarm→WS canlı push (`subscribeAlerts` hâlâ mock).
