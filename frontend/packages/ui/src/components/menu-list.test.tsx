import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MenuList } from "#components/menu-list";

const items = [
  "Workspace",
  { label: "Dashboard", value: "dash", icon: "grid" },
  { label: "Reports", children: [{ label: "Weekly", value: "weekly" }] },
  "separator",
  { label: "Docs", href: "https://x.dev", external: true },
] as const;

test("selecting a row updates aria-current and calls onSelect", async () => {
  const onSelect = vi.fn();
  render(
    <MenuList
      label="Main"
      items={[...items]}
      defaultValue="dash"
      onSelect={onSelect}
      submenu="inline"
    />,
  );
  // DS: buttons get aria-current="true"; only links get "page".
  expect(screen.getByRole("button", { name: "Dashboard" })).toHaveAttribute(
    "aria-current",
    "true",
  );
  await userEvent.click(screen.getByRole("button", { name: "Reports" }));
  await userEvent.click(screen.getByRole("button", { name: "Weekly" }));
  expect(onSelect).toHaveBeenCalledWith(
    "weekly",
    expect.objectContaining({ label: "Weekly" }),
  );
  expect(screen.getByRole("button", { name: "Weekly" })).toHaveAttribute(
    "aria-current",
    "true",
  );
  expect(screen.getByRole("button", { name: "Dashboard" })).not.toHaveAttribute(
    "aria-current",
  );
});

test("inline groups expand with aria-expanded; the group holding the value starts open", () => {
  render(<MenuList items={[...items]} value="weekly" submenu="inline" />);
  expect(screen.getByRole("button", { name: "Reports" })).toHaveAttribute(
    "aria-expanded",
    "true",
  );
});

test("external links open in a new tab and headings render", () => {
  render(<MenuList items={[...items]} />);
  expect(screen.getByRole("link", { name: /Docs/ })).toHaveAttribute(
    "target",
    "_blank",
  );
  expect(screen.getByText("Workspace")).toBeInTheDocument();
});

test("flyout groups open on hover and close on select", async () => {
  const onSelect = vi.fn();
  render(<MenuList items={items} onSelect={onSelect} />);
  const group = screen.getByRole("button", { name: "Reports" });
  expect(group).toHaveAttribute("aria-expanded", "false");
  await userEvent.hover(group);
  expect(group).toHaveAttribute("aria-expanded", "true");
  await userEvent.click(await screen.findByRole("button", { name: "Weekly" }));
  expect(onSelect).toHaveBeenCalledWith(
    "weekly",
    expect.objectContaining({ label: "Weekly" }),
  );
  expect(group).toHaveAttribute("aria-expanded", "false");
});

test("ArrowRight opens the flyout and focuses its first row; Escape returns focus", async () => {
  render(<MenuList items={items} />);
  const group = screen.getByRole("button", { name: "Reports" });
  group.focus();
  await userEvent.keyboard("{ArrowRight}");
  expect(await screen.findByRole("button", { name: "Weekly" })).toHaveFocus();
  await userEvent.keyboard("{Escape}");
  expect(group).toHaveAttribute("aria-expanded", "false");
  expect(group).toHaveFocus();
});

test("Tab from the flyout moves on to the next row in the list", async () => {
  render(<MenuList items={items} />);
  screen.getByRole("button", { name: "Reports" }).focus();
  await userEvent.keyboard("{ArrowRight}");
  expect(await screen.findByRole("button", { name: "Weekly" })).toHaveFocus();
  await userEvent.tab();
  expect(screen.getByRole("link", { name: /Docs/ })).toHaveFocus();
  const group = screen.getByRole("button", { name: "Reports" });
  expect(group).toHaveAttribute("aria-expanded", "false");
  group.focus();
  await userEvent.keyboard("{ArrowRight}");
  expect(await screen.findByRole("button", { name: "Weekly" })).toHaveFocus();
  await userEvent.tab({ shift: true });
  expect(group).toHaveFocus();
});
