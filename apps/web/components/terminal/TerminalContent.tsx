"use client";
import { useState } from "react";
import { DEFAULT_TERMINAL_MINT } from "@/lib/terminal/order-defs";
import { TokenWatchlistPanel } from "./TokenWatchlistPanel";
import { MarketDataHeader } from "./MarketDataHeader";
import { PriceChart } from "./PriceChart";
import { OrderPanel } from "./OrderPanel";
import { BottomTabsPanel } from "./BottomTabsPanel";

export function TerminalContent() {
  const [activeMint, setActiveMint] = useState(DEFAULT_TERMINAL_MINT);
  return (
    <div className="flex flex-col gap-3">
      <h1>Terminal</h1>
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-[220px_1fr_300px]">
        <TokenWatchlistPanel activeMint={activeMint} onSelect={setActiveMint} />
        <div className="flex flex-col gap-3">
          <MarketDataHeader mint={activeMint} />
          <PriceChart mint={activeMint} />
        </div>
        <OrderPanel mint={activeMint} />
      </div>
      <BottomTabsPanel />
    </div>
  );
}
