import { render, screen } from "@testing-library/react";
import { Tag } from "#components/tag";

test("dot variant renders a dot and the text", () => {
  const { container } = render(<Tag tone="success">Active</Tag>);
  expect(screen.getByText("Active")).toHaveAttribute("data-slot", "tag");
  expect(container.querySelector("[data-slot=tag-dot]")).not.toBeNull();
});

test("solid has no dot", () => {
  const { container } = render(<Tag variant="solid">Design</Tag>);
  expect(container.querySelector("[data-slot=tag-dot]")).toBeNull();
});

test("solid accent fills with button and drops the dot-pill classes", () => {
  render(
    <Tag variant="solid" tone="accent">
      UI/UX
    </Tag>,
  );
  const cls = screen.getByText("UI/UX").className.split(" ");
  expect(cls).toContain("[--tag-c:var(--button)]");
  expect(cls).not.toContain("[--tag-c:var(--button-text)]");
  expect(cls).toContain("type-caption");
  expect(cls).not.toContain("type-footnote");
  expect(cls).not.toContain("border");
});
