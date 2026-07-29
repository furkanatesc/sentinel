import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { WalletAddress } from "./WalletAddress";

test("copy writes to clipboard", async () => {
  const writeText = vi.fn();
  Object.assign(navigator, { clipboard: { writeText } });
  render(<WalletAddress address="9xQeWv...4Fk2" />);
  await userEvent.click(screen.getByTitle("Copy address"));
  expect(writeText).toHaveBeenCalledWith("9xQeWv...4Fk2");
});
