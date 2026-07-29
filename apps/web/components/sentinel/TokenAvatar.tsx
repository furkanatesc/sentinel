interface TokenAvatarProps { symbol: string; size?: number; }
const palette = ["#7C5CFF", "#3E9BFF", "#2FD98B", "#FFB020", "#F0476B", "#9B6BFF"];

export function TokenAvatar({ symbol, size = 28 }: TokenAvatarProps) {
  const hash = symbol.split("").reduce((a, c) => a + c.charCodeAt(0), 0);
  const bg = palette[hash % palette.length];
  return (
    <div
      className="flex shrink-0 items-center justify-center rounded-full font-mono"
      style={{ width: size, height: size, fontSize: size * 0.38, fontWeight: 600, color: "#0B1019", background: `linear-gradient(135deg, ${bg}, ${bg}bb)` }}
    >
      {symbol.slice(0, 2)}
    </div>
  );
}
