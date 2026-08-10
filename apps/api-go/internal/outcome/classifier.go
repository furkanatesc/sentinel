// Package outcome, tokenin piyasa trajektörisinden akıbetini (outcome) sınıflandırır.
package outcome

// Outcome ve liquidityStatus için geçerli frontend enum key'leri (apps/web/lib/creator/outcome-defs.ts).
const (
	OutcomeActive    = "active"
	OutcomeGraduated = "graduated"
	OutcomeDumped    = "dumped"
	OutcomeRug       = "rug"
	OutcomeDead      = "dead"

	LiquidityUnlocked = "unlocked"
	LiquidityRemoved  = "removed"
)

// Input, tekil tokenin anlık + tepe piyasa durumudur.
type Input struct {
	CurMarketCap, CurLiquidity, PeakMarketCap, PeakLiquidity, Vol24h float64
	AgeSeconds                                                       int64
}

// Result, sınıflandırma çıktısıdır.
type Result struct {
	Outcome         string
	MaxDrawdownPct  float64
	LiquidityStatus string
}

// Thresholds, sınıflandırma eşikleridir (config'ten; deploy'da kalibre edilir).
type Thresholds struct {
	RugLiqRatio, GraduationMcap, DumpedDrawdown, DeadVol, MinLiqFloor float64
	DeadAgeSec                                                        int64
}

// Classify, öncelikli-eşleşme (ilk eşleşen kazanır) ile outcome üretir. Sıra: rug → graduated
// → dumped → dead → active (terminal/kötü sinyaller önce).
func Classify(in Input, t Thresholds) Result {
	drawdown := 0.0
	if in.PeakMarketCap > 0 {
		drawdown = (in.PeakMarketCap - in.CurMarketCap) / in.PeakMarketCap * 100
		if drawdown < 0 {
			drawdown = 0
		}
		if drawdown > 100 {
			drawdown = 100
		}
	}
	switch {
	case in.PeakLiquidity >= t.MinLiqFloor && in.CurLiquidity <= in.PeakLiquidity*t.RugLiqRatio:
		return Result{OutcomeRug, drawdown, LiquidityRemoved} // LP çekildi
	case in.PeakMarketCap >= t.GraduationMcap && in.CurLiquidity >= t.MinLiqFloor:
		return Result{OutcomeGraduated, drawdown, LiquidityUnlocked} // yüksek-cap + likit
	case drawdown >= t.DumpedDrawdown && in.CurLiquidity >= t.MinLiqFloor:
		return Result{OutcomeDumped, drawdown, LiquidityUnlocked} // fiyat çöktü, likidite duruyor
	case in.AgeSeconds >= t.DeadAgeSec && in.Vol24h <= t.DeadVol:
		return Result{OutcomeDead, drawdown, LiquidityUnlocked} // yaşlı + işlemsiz
	default:
		return Result{OutcomeActive, drawdown, LiquidityUnlocked}
	}
}
