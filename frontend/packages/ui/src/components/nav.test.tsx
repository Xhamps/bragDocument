import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { UserMenu } from "#components/user-menu";
import { TopBar } from "#components/top-bar";
import { PageHeader } from "#components/page-header";
import { Breadcrumb } from "#components/breadcrumb";
import { Tabs } from "#components/tabs";

test("UserMenu opens and reports the chosen item", async () => {
  const onSelect = vi.fn();
  const itemSelect = vi.fn();
  render(
    <UserMenu
      user={{ name: "Ada", role: "Admin" }}
      items={[{ label: "Sign out", tone: "danger", onSelect: itemSelect }]}
      onSelect={onSelect}
    />,
  );
  await userEvent.click(screen.getByRole("button", { name: /Ada/ }));
  await userEvent.click(
    await screen.findByRole("menuitem", { name: "Sign out" }),
  );
  expect(onSelect).toHaveBeenCalledWith(
    "Sign out",
    expect.objectContaining({ tone: "danger" }),
  );
  expect(itemSelect).toHaveBeenCalled();
});

test("TopBar renders brand heading, nav links, actions and the user", () => {
  render(
    <TopBar
      brand={{ name: "Brag Document", href: "/" }}
      items={[{ label: "Settings", href: "/settings" }]}
      actions={<button>theme</button>}
      user={{ variant: "pill", user: { name: "a@acme.com" } }}
    />,
  );
  expect(
    screen.getByRole("heading", { name: "Brag Document" }),
  ).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute(
    "href",
    "/settings",
  );
  expect(screen.getByRole("button", { name: "theme" })).toBeInTheDocument();
  expect(screen.getByText("a@acme.com")).toBeInTheDocument();
});

test("TopBar shows the avatar-only user by default and uses renderLink", () => {
  render(
    <TopBar
      brand={{ name: "Brag Document", href: "/" }}
      items={[{ label: "Settings", href: "/settings" }]}
      user={{ user: { name: "Ada Lovelace" } }}
      renderLink={(item, props) => <a {...props} data-router={item.href} />}
    />,
  );
  expect(
    screen.getByRole("button", { name: "Ada Lovelace, account menu" }),
  ).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute(
    "data-router",
    "/settings",
  );
});

test("PageHeader renders an h1 and actions", () => {
  render(<PageHeader title="Documents" actions={<button>New</button>} />);
  expect(
    screen.getByRole("heading", { level: 1, name: "Documents" }),
  ).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "New" })).toBeInTheDocument();
});

test("Breadcrumb links every crumb but the current one, through renderLink", () => {
  render(
    <Breadcrumb
      items={[{ label: "Documents", href: "/" }, { label: "2026" }]}
      renderLink={(item, props) => <a {...props} data-router={item.href} />}
    />,
  );
  expect(screen.getByRole("link", { name: "Documents" })).toHaveAttribute(
    "data-router",
    "/",
  );
  expect(screen.getByText("2026")).toHaveAttribute("aria-current", "page");
  expect(screen.queryByRole("link", { name: "2026" })).not.toBeInTheDocument();
});

test("Tabs marks the current tab by href", () => {
  render(
    <Tabs
      label="Document views"
      value="/d/dashboard"
      items={[
        { label: "Logs", href: "/d" },
        { label: "Dashboard", href: "/d/dashboard" },
      ]}
    />,
  );
  expect(
    screen.getByRole("navigation", { name: "Document views" }),
  ).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Dashboard" })).toHaveAttribute(
    "aria-current",
    "page",
  );
  expect(screen.getByRole("link", { name: "Logs" })).not.toHaveAttribute(
    "aria-current",
  );
});
