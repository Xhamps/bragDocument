import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { IconButton } from "#components/icon-button";
import { ButtonGroup } from "#components/button-group";

test("label is the accessible name and pressed sets aria-pressed", () => {
  render(<IconButton icon="moon" label="Dark mode" pressed tooltip={false} />);
  expect(screen.getByRole("button", { name: "Dark mode" })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
});

test("numeric badge renders", () => {
  render(
    <IconButton icon="bell" label="Notifications" badge={3} tooltip={false} />,
  );
  expect(screen.getByText("3")).toBeInTheDocument();
});

test("tooltip defaults to the label", async () => {
  render(<IconButton icon="moon" label="Dark mode" />);
  await userEvent.tab();
  expect(await screen.findByRole("tooltip")).toHaveTextContent("Dark mode");
});

test("gradient does not override the square shape", () => {
  render(
    <IconButton
      icon="plus"
      label="Add"
      variant="gradient"
      shape="square"
      tooltip={false}
    />,
  );
  const cls = screen.getByRole("button").className.split(" ");
  expect(cls).toContain("rounded-md");
  expect(cls).not.toContain("rounded-pill");
});

test("ButtonGroup is a labelled group", () => {
  render(
    <ButtonGroup label="View">
      <button>a</button>
    </ButtonGroup>,
  );
  expect(screen.getByRole("group", { name: "View" })).toBeInTheDocument();
});
