import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Card } from "#components/card";

test("renders title, description, action and children", () => {
  render(
    <Card title="Q3" description="Wins" action={<button>Edit</button>}>
      body
    </Card>,
  );
  expect(screen.getByRole("heading", { name: "Q3" })).toBeInTheDocument();
  expect(screen.getByText("Wins")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
  expect(screen.getByText("body")).toBeInTheDocument();
});

test("onClick makes it interactive and keyboard-focusable", async () => {
  const onClick = vi.fn();
  render(<Card title="Open" onClick={onClick} />);
  const card = screen.getByText("Open").closest("[data-slot=card]")!;
  expect(card).toHaveAttribute("tabindex", "0");
  expect(card).toHaveAttribute("data-interactive");
  await userEvent.click(card);
  expect(onClick).toHaveBeenCalledTimes(1);
  await userEvent.keyboard("{Enter}");
  await userEvent.keyboard(" ");
  expect(onClick).toHaveBeenCalledTimes(3);
});

test("plain card is not focusable; as and delay are applied", () => {
  render(
    <Card as="section" animate delay={120}>
      x
    </Card>,
  );
  const card = screen.getByText("x");
  expect(card.tagName).toBe("SECTION");
  expect(card).not.toHaveAttribute("tabindex");
  expect(card).not.toHaveAttribute("data-interactive");
  expect(card.style.getPropertyValue("--delay")).toBe("120ms");
});
