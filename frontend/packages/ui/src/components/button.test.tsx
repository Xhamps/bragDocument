import { render, screen } from "@testing-library/react";
import { Button } from "#components/button";

test("defaults to a glass md button of type button", () => {
  render(<Button>Save</Button>);
  const b = screen.getByRole("button", { name: "Save" });
  expect(b).toHaveAttribute("type", "button");
  expect(b).toHaveAttribute("data-variant", "glass");
  expect(b).toHaveAttribute("data-size", "md");
});

test("type can be overridden", () => {
  render(<Button type="submit">Send</Button>);
  expect(screen.getByRole("button")).toHaveAttribute("type", "submit");
});

test("chevron and named icons render svgs", () => {
  render(
    <Button icon="download" chevron>
      Export
    </Button>,
  );
  expect(screen.getByRole("button").querySelectorAll("svg")).toHaveLength(2);
});

test("asChild renders the child element with icons around its text", () => {
  render(
    <Button asChild variant="primary" trailingIcon="arrowRight">
      <a href="/x">Go</a>
    </Button>,
  );
  const link = screen.getByRole("link", { name: "Go" });
  expect(link).toHaveAttribute("data-variant", "primary");
  expect(link).not.toHaveAttribute("type");
  expect(link.querySelector("svg")).not.toBeNull();
});

test("danger variant is supported", () => {
  render(<Button variant="danger">Delete</Button>);
  expect(screen.getByRole("button")).toHaveAttribute("data-variant", "danger");
});
