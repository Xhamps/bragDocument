import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Input } from "#components/input";

test("accepts typed text", async () => {
  render(<Input aria-label="name" />);
  await userEvent.type(screen.getByLabelText("name"), "hello");
  expect(screen.getByLabelText("name")).toHaveValue("hello");
});
