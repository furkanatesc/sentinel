import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FeedFilters } from "./FeedFilters";
import { EMPTY_FILTERS } from "@/lib/api/types";

test("toggling an event chip emits updated types", async () => {
  const onChange = vi.fn();
  render(<FeedFilters value={EMPTY_FILTERS} onChange={onChange} />);
  await userEvent.click(screen.getByText("Balina Alımı"));
  expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ types: ["whale_buy"] }));
});

test("Temizle resets to EMPTY_FILTERS", async () => {
  const onChange = vi.fn();
  render(<FeedFilters value={{ ...EMPTY_FILTERS, watchlistOnly: true }} onChange={onChange} />);
  await userEvent.click(screen.getByText("Temizle"));
  expect(onChange).toHaveBeenCalledWith(EMPTY_FILTERS);
});
