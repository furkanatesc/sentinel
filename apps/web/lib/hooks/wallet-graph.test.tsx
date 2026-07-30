import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useWalletGraph } from "./queries";

function Probe() {
  const { data } = useWalletGraph();
  return <div>n:{data?.nodes.length ?? 0}</div>;
}

test("useWalletGraph loads the graph", async () => {
  render(
    <QueryClientProvider client={getQueryClient()}>
      <Probe />
    </QueryClientProvider>
  );
  await waitFor(() => expect(screen.getByText(/n:1[0-9]/)).toBeInTheDocument());
});
