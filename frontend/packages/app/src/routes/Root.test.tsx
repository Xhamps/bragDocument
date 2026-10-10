import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  accountButton,
  me,
  mockFetch,
  openAccountMenu,
  renderAt,
  supabaseMock,
} from "../test/mocks";

test("shell renders the title, the caller's avatar, and the documents page", async () => {
  mockFetch({ "GET /me": me, "GET /documents": { owned: [], shared: [] } });
  renderAt("/");
  expect(
    await screen.findByRole("heading", { name: "Brag Document" }),
  ).toBeInTheDocument();
  await accountButton();
  // The email only shows inside the menu, not in the bar.
  expect(screen.queryByText("a@acme.com")).not.toBeInTheDocument();
  expect(
    await screen.findByRole("heading", { name: "Your documents" }),
  ).toBeInTheDocument();
});

test("unauthenticated visitor is sent to sign-in", async () => {
  supabaseMock.auth.getSession.mockResolvedValueOnce({
    data: { session: null },
  } as never);
  renderAt("/settings");
  expect(
    await screen.findByRole("heading", { name: /sign in/i }),
  ).toBeInTheDocument();
});

test("unknown path renders not found", async () => {
  mockFetch({ "GET /me": me });
  renderAt("/nope");
  expect(
    await screen.findByRole("heading", { name: "Page not found" }),
  ).toBeInTheDocument();
});

test("sign-out clears the cached caller", async () => {
  mockFetch({ "GET /me": me, "GET /documents": { owned: [], shared: [] } });
  renderAt("/");
  await accountButton();
  const onChange = supabaseMock.auth.onAuthStateChange.mock.calls[0][0];
  act(() => onChange("SIGNED_OUT", null));
  await waitFor(() =>
    expect(
      screen.queryByRole("button", { name: "Ada, account menu" }),
    ).not.toBeInTheDocument(),
  );
});

test("the account menu holds the nav and sign-out; the theme toggle stays in the bar", async () => {
  mockFetch({
    "GET /me": me,
    "GET /documents": { owned: [], shared: [] },
  });
  const { router } = renderAt("/");
  expect(
    await screen.findByRole("button", { name: "Toggle dark mode" }),
  ).toBeInTheDocument();
  await openAccountMenu();
  expect(screen.getByText("a@acme.com")).toBeInTheDocument();
  expect(screen.getByRole("menuitem", { name: "Acme" })).toHaveAttribute(
    "href",
    "/tenant",
  );
  expect(screen.getByRole("menuitem", { name: "Audit log" })).toHaveAttribute(
    "href",
    "/audit",
  );
  expect(
    screen.getByRole("menuitem", { name: "Sign out" }),
  ).toBeInTheDocument();
  // Client-side navigation, not a full page load.
  await userEvent.click(screen.getByRole("menuitem", { name: "Settings" }));
  expect(router.state.location.pathname).toBe("/settings");
});
