// Package market, agregatör (GeckoTerminal) tabanlı token keşif+enrichment sağlar (Slice 1b).
// WS ingestion'dan bağımsız REST döngüleridir.
package market

import "context"

// Pool, bir agregatör havuz kaydının kaynak-bağımsız görünümüdür.
type Pool struct {
	PoolAddr      string
	Mint          string // base token adresi
	Name, Symbol  string
	Dex           string // agregatör dex kimliği (ör. "pumpfun", "raydium")
	Price         float64
	LiquidityUSD  float64
	Vol5m         float64
	PriceChangeH1 float64 // yüzde
	CreatedAtUnix int64

	PriceChangeH24 float64 // h24 yüzde
	MarketCapUSD   float64 // market_cap_usd, yoksa fdv_usd
	Vol24h         float64

	TxnsBuys, TxnsSells, TxnsBuyers, TxnsSellers int // h24 işlem sayıları (2c manipülasyon)
}

// Candle, OHLCV mumunun grafik için gereken kısmıdır (close + volume).
type Candle struct {
	Ts     int64
	Close  float64
	Volume float64
}

// MarketProvider, piyasa verisi kaynağıdır (DIP). GeckoTerminal ilk somut impl; DexScreener sonra (OCP).
type MarketProvider interface {
	NewPools(ctx context.Context) ([]Pool, error)
	PoolsByAddresses(ctx context.Context, poolAddrs []string) ([]Pool, error)
	OHLCV(ctx context.Context, poolAddr, timeframe string, limit int) ([]Candle, error)
}

// Broadcaster, snapshot/olayları client'lara yayar (tüketici-tanımlı arayüz; ws.Hub karşılar).
type Broadcaster interface {
	Broadcast(topic string, payload any)
}

// launchpadByDex, desteklenen dex kimliklerini SENTINEL gösterim adına eşler.
// Buradaki anahtarlar canlı GeckoTerminal örneğiyle doğrulanmalı (deploy kalibrasyonu).
var launchpadByDex = map[string]string{
	"pumpfun":      "Pump.fun",
	"pump-fun":     "Pump.fun",
	"raydium":      "Raydium",
	"raydium-clmm": "Raydium",
	"raydium-cpmm": "Raydium",
}

// DexToLaunchpad, dex kimliğini gösterim adına çevirir; desteklenmiyorsa ok=false (filtrele).
func DexToLaunchpad(dexID string) (string, bool) {
	name, ok := launchpadByDex[dexID]
	return name, ok
}
