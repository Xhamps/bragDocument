import { render, screen } from "@testing-library/react";
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

test("tooltip defaults to the label", () => {
  render(<IconButton icon="moon" label="Dark mode" />);
  expect(screen.getByRole("button", { name: "Dark mode" })).toBeInTheDocument();
});

test("ButtonGroup is a labelled group", () => {
  render(
    <ButtonGroup label="View">
      <button>a</button>
    </ButtonGroup>,
  );
  expect(screen.getByRole("group", { name: "View" })).toBeInTheDocument();
});
