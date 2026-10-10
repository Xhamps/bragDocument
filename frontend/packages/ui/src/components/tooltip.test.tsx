import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Tooltip } from "#components/tooltip";

test("shows content and shortcut on focus", async () => {
  render(
    <Tooltip content="Search" shortcut="⌘K">
      <button>s</button>
    </Tooltip>,
  );
  await userEvent.tab();
  expect(await screen.findByRole("tooltip")).toHaveTextContent("Search⌘K");
});

test("open forces it visible", () => {
  render(
    <Tooltip content="Hi" open>
      <button>s</button>
    </Tooltip>,
  );
  expect(screen.getByRole("tooltip")).toHaveTextContent("Hi");
});

test("open={false} does not lock it closed", async () => {
  render(
    <Tooltip content="Hi" open={false}>
      <button>s</button>
    </Tooltip>,
  );
  await userEvent.tab();
  expect(await screen.findByRole("tooltip")).toHaveTextContent("Hi");
});
