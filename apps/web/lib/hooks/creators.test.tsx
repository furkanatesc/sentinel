import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useCreator } from "./queries";

function Probe() {
  const { data } = useCreator("CreAxz");
  return <div>a:{data?.address ?? "-"}</div>;
}

test("useCreator loads a profile", async () => {
  render(
    <QueryClientProvider client={getQueryClient()}>
      <Probe />
    </QueryClientProvider>
  );
  await waitFor(() => expect(screen.getByText("a:CreAxz")).toBeInTheDocument());
});
