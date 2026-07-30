import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useEvents } from "./queries";

function Probe() {
  const { data } = useEvents();
  return <div>n:{data?.length ?? 0}</div>;
}

test("useEvents loads the seed stream", async () => {
  render(
    <QueryClientProvider client={getQueryClient()}>
      <Probe />
    </QueryClientProvider>
  );
  await waitFor(() => expect(screen.getByText(/n:2[0-9]/)).toBeInTheDocument());
});
