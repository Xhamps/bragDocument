import { screen } from "@testing-library/react";
import {
  accountButton,
  me,
  mockFetch,
  renderAt,
  supabaseMock,
} from "../test/mocks";

function signedOut() {
  supabaseMock.auth.getSession.mockResolvedValueOnce({
    data: { session: null },
  } as never);
}

test("signed-out visitor at / sees the home page", async () => {
  signedOut();
  renderAt("/");
  expect(
    await screen.findByRole("heading", {
      level: 1,
      name: "You did the work. We keep the receipts.",
    }),
  ).toBeInTheDocument();
  // Header and hero both offer Sign in.
  const signIns = screen.getAllByRole("link", { name: "Sign in" });
  expect(signIns).toHaveLength(2);
  for (const a of signIns) expect(a).toHaveAttribute("href", "/sign-in");
  expect(screen.getByRole("link", { name: /brag document/i })).toHaveAttribute(
    "href",
    "/",
  );
  expect(
    screen.getAllByRole("heading", { level: 2 }).map((h) => h.textContent),
  ).toEqual(["From a quick note to a review-ready story"]);
  expect(
    screen.getAllByRole("heading", { level: 3 }).map((h) => h.textContent),
  ).toEqual([
    "Log every win",
    "Telegram bot",
    "Audit trail",
    "See your impact",
    "Share and export",
  ]);
});

test("signed-out visitor on a protected page is still sent to sign-in", async () => {
  signedOut();
  renderAt("/settings");
  expect(
    await screen.findByRole("heading", { name: /sign in/i }),
  ).toBeInTheDocument();
});

test("signed-in user at / still sees their documents", async () => {
  mockFetch({ "GET /me": me, "GET /documents": { owned: [], shared: [] } });
  renderAt("/");
  expect(await accountButton()).toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "Sign in" })).toBeNull();
});
