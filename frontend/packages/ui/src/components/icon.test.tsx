import { render } from "@testing-library/react";
import { Icon, renderIcon } from "#components/icon";

test("renders a named icon at 20px with a 1.5 stroke", () => {
  const { container } = render(<Icon name="search" />);
  const svg = container.querySelector("svg")!;
  expect(svg).toHaveAttribute("width", "20");
  expect(svg).toHaveAttribute("stroke-width", "1.5");
  expect(svg).toHaveAttribute("aria-hidden", "true");
});

test("renderIcon passes nodes through and ignores unknown names", () => {
  const { container } = render(
    <>
      {renderIcon(<b>x</b>)}
      {renderIcon("nope")}
    </>,
  );
  expect(container.innerHTML).toBe("<b>x</b>");
});
