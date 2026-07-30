import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { CreatorProfileContent } from "./CreatorProfileContent";

test("renders reputation, metrics, history and behavior", async () => {
  render(
    <QueryClientProvider client={getQueryClient()}>
      <CreatorProfileContent address="CreAxz" />
    </QueryClientProvider>
  );
  await waitFor(() => expect(screen.getByText("Token Geçmişi")).toBeInTheDocument());
  expect(screen.getByText("Davranış Paterni")).toBeInTheDocument();
  expect(screen.getByText("Toplam Token")).toBeInTheDocument();
});
