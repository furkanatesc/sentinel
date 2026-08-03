"use client";
import { useState } from "react";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { TERMINAL_TAB_DEFS } from "@/lib/terminal/order-defs";
import { usePositions } from "@/lib/hooks/queries";
import { PositionsTable, type SortKey } from "@/components/position/PositionsTable";
import { sortPositions } from "@/lib/position/sort";
import { Skeleton } from "@/components/ui/skeleton";
import { OrdersTable } from "./OrdersTable";
import { TransactionsTable } from "./TransactionsTable";
import { TradeLogsList } from "./TradeLogsList";

function PositionsTab() {
  const { data } = usePositions();
  const [sortKey, setSortKey] = useState<SortKey>("pnlSol");
  if (!data) return <Skeleton className="h-40 w-full" />;
  // Detail drawer is intentionally omitted in the terminal context (kept on the dedicated /positions page).
  return <PositionsTable rows={sortPositions(data, sortKey)} sortKey={sortKey} onSort={setSortKey} onRowClick={() => {}} />;
}

export function BottomTabsPanel() {
  return (
    <div className="rounded-lg border border-border bg-card">
      <Tabs defaultValue="positions" className="w-full">
        <TabsList className="flex flex-wrap">
          {TERMINAL_TAB_DEFS.map((t) => <TabsTrigger key={t.key} value={t.key}>{t.label}</TabsTrigger>)}
        </TabsList>
        <TabsContent value="positions" className="mt-2 overflow-x-auto"><PositionsTab /></TabsContent>
        <TabsContent value="orders" className="mt-2 overflow-x-auto"><OrdersTable /></TabsContent>
        <TabsContent value="transactions" className="mt-2 overflow-x-auto"><TransactionsTable /></TabsContent>
        <TabsContent value="logs" className="mt-2"><TradeLogsList /></TabsContent>
      </Tabs>
    </div>
  );
}
