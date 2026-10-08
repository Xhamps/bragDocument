import { render, screen } from "@testing-library/react";
import { Badge } from "#components/badge";

test("renders text with badge slot", () => {
  render(<Badge variant="secondary">Archived</Badge>);
  expect(screen.getByText("Archived")).toHaveAttribute("data-slot", "badge");
});
