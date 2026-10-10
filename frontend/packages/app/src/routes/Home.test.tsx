import { screen } from "@testing-library/react";
import { me, mockFetch, renderAt, supabaseMock } from "../test/mocks";

function signedOut() {
  supabaseMock.auth.getSession.mockResolvedValueOnce({
    data: { session: null },
  } as never);
}

test("signed-out visitor at / sees the home page", async () => {
  signedOut();
  renderAt("/");
  expect(
    await screen.findByRole("heading", { level: 1, name: "Brag Document" }),
  ).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Sign in" })).toHaveAttribute(
    "href",
    "/sign-in",
  );
  expect(
    screen.getAllByRole("heading", { level: 2 }).map((h) => h.textContent),
  ).toEqual(["Log every win", "See your impact", "Share and export"]);
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
  expect(await screen.findByText("a@acme.com")).toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "Sign in" })).toBeNull();
});
