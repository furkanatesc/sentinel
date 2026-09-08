# Alarm Kuralları Persist Implementation Plan

> subagent-driven-development / executing-plans. Steps checkbox.

**Goal:** `/alerts` kurallarını backend'de kalıcı kıl — getAlertRules gerçek + create/toggle mutation (uygulamanın ilk mutation seam'i).

**Spec:** `docs/superpowers/specs/2026-09-08-sentinel-alert-rules-persist-design.md`

## Global Constraints
- Clean/SOLID: dar `AlertRuleStore`, SRP handler; mutation = kural CRUD (trade değil).
- Geriye uyumlu: mock modda mutation'lar in-memory dizide simüle (görsel tutarlı).
- `go test ./... -race`+vet+build, frontend tsc/vitest/build yeşil; mevcut testler kırılmaz.
- Commit sonu: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`

## Task 1: migration 0016 + AlertRuleStore (postgres+fake) + seed
- Create `internal/store/migrations/0016_create_alert_rules.sql`; `internal/store/alertrules.go` (AlertRule + AlertRuleStore iface + postgres impl: List/Create/SetEnabled, channels JSON, seed boşsa 4 default); fake in-memory. Bundle.AlertRules + OpenPostgres seed.
- Test `alertrules_test.go` (fake List/Create/SetEnabled + seed).
- Commit.

## Task 2: Go handlers + router + main
- `internal/api/alerts.go`: alertRulesListHandler(GET) + createHandler(POST 201) + setEnabledHandler(PATCH 204, {enabled}); 400/404. RouterDeps.AlertRules + route (d.AlertRules != nil). main.go bundle.AlertRules.
- Test `alerts_test.go` (GET/POST/PATCH/404/400).
- `go build/vet/test ./... -race` yeşil. Commit.

## Task 3: Frontend seam + mock + http + hooks
- `contract.ts`: `createAlertRule(draft): Promise<AlertRule>`, `setAlertRuleEnabled(id, enabled): Promise<void>`.
- `mock.ts`: mutation'lar `alertRules` dizisini günceller (create push + id üret; setEnabled flip).
- `http.ts`: getAlertRules zaten var; createAlertRule POST /api/alert-rules; setAlertRuleEnabled PATCH /api/alert-rules/{id}.
- `live-endpoints.ts`: getAlertRules + createAlertRule + setAlertRuleEnabled.
- `lib/hooks/mutations.ts` (yeni): `useCreateAlertRule`, `useSetAlertRuleEnabled` (invalidate qk.alertRules).
- mock.test şekil. Commit.

## Task 4: /alerts rewire + test
- `AlertRulesPanel`: local override → `useSetAlertRuleEnabled` (toggle gerçek + invalidate).
- `AlertRuleForm`: simüle toast → `useCreateAlertRule` (submit gerçek + invalidate + toast + onDone).
- Testler mutation hook'larını mock'lar (çağrı + invalidate). tsc/vitest/build yeşil. Commit.

## Task 5: Review + docs + merge/push
- pytest yok (Go+frontend); go -race + frontend build yeşil.
- Whole-branch review → receiving-code-review.
- progress.md + MEMORY + followups (NotificationConfig persist, delete/edit, optimistic ertelendi).
- Merge/push (kullanıcı "phase'ler bitene dek soru sorma" dedi → onaysız).

## Self-Review
Spec coverage: migration/store→T1; handlers→T2; seam/mock/http/hooks→T3; rewire→T4; review/docs→T5 ✅.
Type consistency: AlertRule (Go store) ↔ frontend AlertRule; AlertRuleDraft (mevcut); createAlertRule/setAlertRuleEnabled contract↔http↔mock↔hooks; AlertRules bundle↔router↔main ✅.
