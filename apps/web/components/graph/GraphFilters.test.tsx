import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphFilters } from "./GraphFilters";
import { EMPTY_GRAPH_FILTERS } from "@/lib/api/types";
test("toggling a relationship chip emits it", async () => {
  const onChange = vi.fn();
  render(<GraphFilters value={EMPTY_GRAPH_FILTERS} onChange={onChange} />);
  await userEvent.click(screen.getByText("Oluşturdu"));
  expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ relationships: ["created"] }));
});
test("Temizle resets", async () => {
  const onChange = vi.fn();
  render(<GraphFilters value={{ relationships: ["created"], risks: [] }} onChange={onChange} />);
  await userEvent.click(screen.getByText("Temizle"));
  expect(onChange).toHaveBeenCalledWith(EMPTY_GRAPH_FILTERS);
});
