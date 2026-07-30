import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { TokenActions } from "./TokenActions";
import { useSessionStore } from "@/lib/store/session";
vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { warning: vi.fn() }) }));

test("Al in live mode warns about real funds", async () => {
  useSessionStore.setState({ tradingMode: "live" });
  render(<TokenActions symbol="PULSE" />);
  await userEvent.click(screen.getByText("Al"));
  expect((toast as any).warning).toHaveBeenCalled();
});
