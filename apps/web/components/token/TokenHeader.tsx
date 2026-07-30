import { TokenAvatar } from "@/components/sentinel/TokenAvatar";
import { WalletAddress } from "@/components/sentinel/WalletAddress";
import { TokenActions } from "./TokenActions";
import { formatAge, formatPrice, formatUsd } from "@/lib/format";
import type { TokenDetail } from "@/lib/api/types";

function Stat({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div className="flex flex-col">
      <span className="text-muted-foreground" style={{ fontSize: 11 }}>{label}</span>
      <span className="font-mono tabular-nums" style={{ fontSize: 14, fontWeight: 600, color }}>{value}</span>
    </div>
  );
}

export function TokenHeader({ token }: { token: TokenDetail }) {
  const up = token.priceChange24h >= 0;
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <TokenAvatar symbol={token.symbol} size={44} />
          <div className="flex flex-col">
            <div className="flex items-center gap-2">
              <span style={{ fontSize: 18, fontWeight: 600 }}>{token.name}</span>
              <span className="text-muted-foreground">{token.symbol}</span>
            </div>
            <WalletAddress address={token.mint} />
          </div>
        </div>
        <TokenActions symbol={token.symbol} />
      </div>
      <div className="mt-4 grid grid-cols-3 gap-4 md:grid-cols-6">
        <Stat label="Fiyat" value={formatPrice(token.price)} />
        <Stat label="24s Değişim" value={`${up ? "+" : ""}${token.priceChange24h.toFixed(1)}%`} color={up ? "#2FD98B" : "#F0476B"} />
        <Stat label="Market Cap" value={formatUsd(token.marketCap)} />
        <Stat label="Likidite" value={formatUsd(token.liquidity)} />
        <Stat label="24s Hacim" value={formatUsd(token.volume24h)} />
        <Stat label="Yaş" value={formatAge(token.ageSeconds)} />
      </div>
    </div>
  );
}
