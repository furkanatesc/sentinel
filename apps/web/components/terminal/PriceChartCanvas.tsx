"use client";
import { useEffect, useRef } from "react";
import { createChart, ColorType, type IChartApi } from "lightweight-charts";
import type { Candle } from "@/lib/api/types";

export function PriceChartCanvas({ candles }: { candles: Candle[] }) {
  const ref = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  useEffect(() => {
    if (!ref.current) return;
    const chart = createChart(ref.current, {
      autoSize: true,
      layout: { background: { type: ColorType.Solid, color: "transparent" }, textColor: "#8A94A6" },
      grid: { vertLines: { color: "rgba(255,255,255,0.05)" }, horzLines: { color: "rgba(255,255,255,0.05)" } },
      rightPriceScale: { borderColor: "rgba(255,255,255,0.07)" },
      timeScale: { borderColor: "rgba(255,255,255,0.07)" },
    });
    const series = chart.addCandlestickSeries({
      upColor: "#2FD98B", downColor: "#F0476B",
      wickUpColor: "#2FD98B", wickDownColor: "#F0476B", borderVisible: false,
    });
    series.setData(candles as never);
    chart.timeScale().fitContent();
    chartRef.current = chart;
    return () => { chartRef.current = null; chart.remove(); };
  }, [candles]);
  return <div ref={ref} data-testid="price-chart" className="h-[320px] w-full rounded-lg border border-border bg-card" />;
}
