import { render, screen } from "@testing-library/react";
import { PriceChartCanvas } from "./PriceChartCanvas";

vi.mock("lightweight-charts", () => ({
  ColorType: { Solid: "solid" },
  createChart: vi.fn(() => ({
    addCandlestickSeries: vi.fn(() => ({ setData: vi.fn() })),
    timeScale: vi.fn(() => ({ fitContent: vi.fn() })),
    remove: vi.fn(),
  })),
}));
import { createChart } from "lightweight-charts";

test("creates a candlestick chart and renders a container", () => {
  render(<PriceChartCanvas candles={[{ time: 1, open: 1, high: 2, low: 0.5, close: 1.5 }]} />);
  expect(createChart).toHaveBeenCalled();
  expect(screen.getByTestId("price-chart")).toBeInTheDocument();
});
