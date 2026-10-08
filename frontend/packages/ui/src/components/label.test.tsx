import { render, screen } from "@testing-library/react";
import { Label } from "#components/label";

test("renders with htmlFor", () => {
  render(<Label htmlFor="email">Email</Label>);
  expect(screen.getByText("Email")).toHaveAttribute("for", "email");
});
