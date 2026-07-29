import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useKpis } from "./queries";

function Probe() {
  const { data } = useKpis();
  return <div>count:{data?.length ?? 0}</div>;
}

test("useKpis loads mock kpis", async () => {
  const client = getQueryClient();
  render(
    <QueryClientProvider client={client}>
      <Probe />
    </QueryClientProvider>
  );
  await waitFor(() => expect(screen.getByText("count:8")).toBeInTheDocument());
});
