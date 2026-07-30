import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useToken } from "./queries";

function Probe() { const { data } = useToken("PULSE"); return <div>sym:{data?.symbol ?? "-"}</div>; }

test("useToken loads a token detail", async () => {
  render(<QueryClientProvider client={getQueryClient()}><Probe /></QueryClientProvider>);
  await waitFor(() => expect(screen.getByText("sym:PULSE")).toBeInTheDocument());
});
