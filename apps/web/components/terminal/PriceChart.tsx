"use client";
import dynamic from "next/dynamic";
import { Skeleton } from "@/components/ui/skeleton";
import { useCandles } from "@/lib/hooks/queries";

const PriceChartCanvas = dynamic(() => import("./PriceChartCanvas").then((m) => m.PriceChartCanvas), {
  ssr: false, loading: () => <Skeleton className="h-[320px] w-full" />,
});

export function PriceChart({ mint }: { mint: string }) {
  const { data } = useCandles(mint);
  if (!data) return <Skeleton className="h-[320px] w-full" />;
  return <PriceChartCanvas candles={data} />;
}
