// Package alerteval, aktif alarm kurallarını mevcut event akışıyla eşleştiren değerlendirme motorudur.
// Saf eşleştirme (match.go) + periyodik worker (worker.go). Yeni harici bağımlılık YOK.
package alerteval

import (
	"strings"

	"github.com/furkanatesc/sentinel/apps/api-go/internal/store"
)

// EvaluableTriggers, v1'de değerlendirilen trigger'lardır (EventRow verisi güvenilir dolu).
// Kapsam-dışı trigger'lar (holder_growth/whale_activity/score_change/creator_sale/strategy_signal)
// veri gelene dek asla eşleşmez — sessiz düşürme yok, bilinçli kapsam.
var EvaluableTriggers = map[string]bool{
	"new_mint":          true,
	"liquidity_added":   true,
	"liquidity_removed": true,
}

// riskRank, risk seviyesini sıralanabilir sayıya çevirir (tavan karşılaştırması için).
func riskRank(level string) int {
	switch strings.ToLower(level) {
	case "low":
		return 0
	case "medium":
		return 1
	case "high":
		return 2
	case "critical":
		return 3
	default:
		return 1 // bilinmeyen → medium gibi davran (güvenli orta)
	}
}

// MatchRule, bir event'in bir kurala uyup uymadığını döner (saf; SRP).
func MatchRule(e store.EventRow, r store.AlertRule) bool {
	if r.Trigger != e.Type || !EvaluableTriggers[r.Trigger] {
		return false
	}
	if e.Liquidity < r.MinLiquidity {
		return false
	}
	if e.CreatorScore < r.MinCreatorScore {
		return false
	}
	if riskRank(e.RiskLevel) > riskRank(r.MaxRisk) {
		return false
	}
	scope := strings.TrimSpace(r.Scope)
	if scope == "" || strings.EqualFold(scope, "Tüm tokenlar") {
		return true
	}
	return strings.EqualFold(scope, strings.TrimSpace(e.Launchpad))
}
