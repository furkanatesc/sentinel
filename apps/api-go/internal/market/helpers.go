package market

const (
	momentumK = 0.5 // +100% → 100, -100% → 0
	sparkMax  = 16  // spark dizisinde tutulan son örnek sayısı
)

// momentumFromChange, kısa-vade fiyat değişimini 0-100 momentum'a çevirir (50=yatay). Skor DEĞİL, saf fiyat aksiyonu.
func momentumFromChange(pctH1 float64) float64 {
	m := 50 + pctH1*momentumK
	if m < 0 {
		return 0
	}
	if m > 100 {
		return 100
	}
	return m
}

// appendSpark, güncel fiyatı spark'a ekler ve son sparkMax örnekle sınırlar.
func appendSpark(cur []float64, price float64) []float64 {
	out := append(append([]float64{}, cur...), price)
	if len(out) > sparkMax {
		out = out[len(out)-sparkMax:]
	}
	return out
}
