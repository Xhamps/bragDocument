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

test("glow wins over the variant shadow, also on hover", () => {
  render(<Button glow>Go</Button>);
  const cls = screen.getByRole("button").className.split(" ");
  expect(cls).toContain("hover:shadow-glow-strong");
  expect(cls).not.toContain("hover:shadow-md");
  expect(cls).not.toContain("shadow-button");
});

test("lg size and sm pill resolve their conflicts", () => {
  render(
    <>
      <Button size="lg">L</Button>
      <Button size="sm" pill>
        S
      </Button>
    </>,
  );
  const lg = screen.getByRole("button", { name: "L" }).className.split(" ");
  expect(lg).toContain("type-body");
  expect(lg).not.toContain("type-callout");
  const sm = screen.getByRole("button", { name: "S" }).className.split(" ");
  expect(sm).toContain("rounded-pill");
  expect(sm).not.toContain("rounded-sm");
});
