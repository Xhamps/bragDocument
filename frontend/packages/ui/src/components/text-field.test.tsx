import { render, screen } from "@testing-library/react";
import { TextField } from "#components/text-field";
import { TextArea } from "#components/text-area";

test("label is associated and hint describes the input", () => {
  render(<TextField label="Email" hint="Work address" />);
  const input = screen.getByLabelText("Email");
  expect(input).toHaveAccessibleDescription("Work address");
  expect(input).not.toHaveAttribute("aria-invalid");
});

test("error replaces the hint and marks the input invalid", () => {
  render(<TextField label="Email" hint="Work address" error="Required" />);
  const input = screen.getByLabelText("Email");
  expect(input).toHaveAttribute("aria-invalid", "true");
  expect(input).toHaveAccessibleDescription("Required");
  expect(screen.getByRole("alert")).toHaveTextContent("Required");
  expect(screen.queryByText("Work address")).toBeNull();
});

test("TextArea shares the frame", () => {
  render(<TextArea label="Notes" error="Too long" />);
  expect(screen.getByLabelText("Notes")).toHaveAttribute(
    "aria-invalid",
    "true",
  );
});
