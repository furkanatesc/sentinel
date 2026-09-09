# SENTINEL — Alarm Kuralları Persist Tasarım Spec'i

**Tarih:** 2026-09-08
**Dilim:** Backend Alt-proje 3 kısmi (Alerts backend — Slack teslimatı hariç). Uygulamanın **İLK
MUTATION seam'i** (kural CRUD; trade-mutasyonu DEĞİL → yapısal-güvenlik ihlali yok).
**Durum:** Spec — implementasyon öncesi.

## Amaç

`/alerts` ekranındaki alarm kurallarını backend'de kalıcı kıl: `getAlertRules` gerçeğe döner (DB) +
**create + toggle** kalıcı olur (şu an simüle-toast/local-state). **Entegrasyon-gerektirmez** (Postgres + Go).
NotificationConfig persist bu dilimde YOK (followup — mock kalır, dilim odaklı).

## Kapsam

**Dahil:**
- Migration `0016_create_alert_rules.sql`.
- Go: `AlertRuleStore` (List/Create/SetEnabled) + seed (mevcut mock default'ları) + handlers (GET/POST/PATCH).
- Frontend: contract mutation'ları (`createAlertRule`, `setAlertRuleEnabled`); httpApi + mock impl;
  `getAlertRules` `LIVE_ENDPOINTS`'e; `/alerts` rewire (toggle→PATCH, form→POST, React Query invalidation).

**Kapsam dışı:** NotificationConfig persist/save (followup, mock kalır); silme/düzenleme (yalnız create+toggle);
gerçek Slack teslimatı (Alt-proje 3 kalanı).

## Mimari

### Migration 0016
```sql
CREATE TABLE IF NOT EXISTS alert_rules (
    id                TEXT PRIMARY KEY,
    name              TEXT NOT NULL,
    trigger           TEXT NOT NULL,
    scope             TEXT NOT NULL,
    min_liquidity     DOUBLE PRECISION NOT NULL DEFAULT 0,
    min_creator_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_risk          TEXT NOT NULL,
    channels          TEXT NOT NULL DEFAULT '[]',  -- JSON []string
    enabled           BOOLEAN NOT NULL DEFAULT true,
    created_ts        BIGINT NOT NULL DEFAULT 0
);
```

### Store (`internal/store/alertrules.go` + postgres + fake)
```go
type AlertRule struct {
    ID, Name, Trigger, Scope, MaxRisk string
    MinLiquidity, MinCreatorScore     float64
    Channels                          []string
    Enabled                           bool
    CreatedTs                         int64
}
type AlertRuleStore interface {
    ListAlertRules(ctx) ([]AlertRule, error)
    CreateAlertRule(ctx, AlertRule) (AlertRule, error)   // id yoksa üret (ör. "r"+ts/rand); channels JSON
    SetAlertRuleEnabled(ctx, id string, enabled bool) error
}
```
postgres: channels JSON marshal/unmarshal; seed (List boşsa mevcut 4 mock kuralı ekle — `seedStrategies` deseni).
fake: in-memory slice (parity).

### Handlers (`internal/api/alerts.go`)
- `GET /api/alert-rules` → List.
- `POST /api/alert-rules` (body AlertRule draft) → Create → 201 + oluşturulan kural.
- `PATCH /api/alert-rules/{id}` (body `{"enabled": bool}`) → SetEnabled → 204.
Router: `d.AlertRules != nil` ise kayıtlı; `RouterDeps.AlertRules`. main.go: `bundle.AlertRules` (postgres) /
fake. Bundle'a `AlertRules` eklenir; OpenPostgres seed çağrısı.

### Frontend
- `contract.ts`: `createAlertRule(draft: AlertRuleDraft): Promise<AlertRule>`; `setAlertRuleEnabled(id, enabled): Promise<void>`.
- `http.ts`: `getAlertRules` zaten var (LIVE_ENDPOINTS'e ekle → gerçek); `createAlertRule` POST; `setAlertRuleEnabled` PATCH.
- `mock.ts`: mutation'lar in-memory `alertRules` dizisini günceller (simüle ama tutarlı — mock modda da çalışsın).
- `live-endpoints.ts`: `getAlertRules` + `createAlertRule` + `setAlertRuleEnabled` (mock modda mock, http modda gerçek).
- Rewire: `AlertRulesPanel` toggle → `useSetAlertRuleEnabled` mutation + `qk.alertRules` invalidate (local override kalkar);
  `AlertRuleForm` submit → `useCreateAlertRule` mutation + invalidate + toast; `useMutation` hook'ları `lib/hooks/`.
- `AlertRuleDraft` zaten `lib/alerts/alert-defs.ts`'te.

## Veri akışı

`/alerts` → getAlertRules (gerçek DB) → toggle PATCH / create POST → React Query invalidate → yeniden fetch.
Mock modda mock dizisi güncellenir (görsel tutarlılık). Frontend seam kontratı genişler (2 mutation) ama
mevcut read akışı korunur.

## Hata yönetimi

- Store: create id çakışması → hata; DB hatası → 5xx.
- Handler: geçersiz body → 400; PATCH bilinmeyen id → 404.
- Frontend: mutation hatası → toast.error; optimistic değil (invalidate-refetch, basit + doğru).

## Test

- Go: fake store List/Create/SetEnabled; handler GET/POST/PATCH (201/204/404/400).
- Frontend: mock mutation'lar (create ekler, toggle çevirir); AlertRulesPanel toggle mutation çağırır;
  AlertRuleForm submit createAlertRule çağırır (mock hook). Mevcut testler kırılmaz.

## Deploy sonrası

Railway'de migration 0016 otomatik; `/api/alert-rules` gerçek. Frontend push → Vercel; /alerts kuralları
kalıcı. (Mutation ilk kez canlı — güvenli: trade değil, config CRUD.)

## Followup

- NotificationConfig persist + save mutation. Kural silme/düzenleme. Gerçek Slack teslimatı (Alt-proje 3).
- Optimistic update (şu an invalidate-refetch).
