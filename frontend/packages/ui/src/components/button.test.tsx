import { render, screen } from "@testing-library/react";
import { Button } from "#components/button";

test("renders children and applies the destructive variant", () => {
  render(<Button variant="destructive">Delete</Button>);
  const btn = screen.getByRole("button", { name: "Delete" });
  expect(btn).toBeInTheDocument();
  expect(btn.className).toContain("bg-destructive");
});
