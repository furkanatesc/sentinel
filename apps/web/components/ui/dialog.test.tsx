import { render, screen } from "@testing-library/react";
import { Dialog, DialogContent, DialogTitle } from "./dialog";

test("renders dialog content when open", () => {
  render(
    <Dialog open>
      <DialogContent><DialogTitle>Onay</DialogTitle></DialogContent>
    </Dialog>
  );
  expect(screen.getByText("Onay")).toBeInTheDocument();
});

test("does not render content when closed", () => {
  render(
    <Dialog open={false}>
      <DialogContent><DialogTitle>Gizli</DialogTitle></DialogContent>
    </Dialog>
  );
  expect(screen.queryByText("Gizli")).not.toBeInTheDocument();
});
