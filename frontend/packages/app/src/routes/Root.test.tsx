import { screen } from "@testing-library/react";
import { me, mockFetch, renderAt, supabaseMock } from "../test/mocks";

test("shell renders the title, the caller, and the documents page", async () => {
  mockFetch({ "GET /me": me, "GET /documents": { owned: [], shared: [] } });
  renderAt("/");
  expect(
    await screen.findByRole("heading", { name: "Brag Document" }),
  ).toBeInTheDocument();
  expect(await screen.findByText("a@acme.com")).toBeInTheDocument();
});

test("unauthenticated visitor is sent to sign-in", async () => {
  supabaseMock.auth.getSession.mockResolvedValueOnce({
    data: { session: null },
  } as never);
  renderAt("/");
  expect(
    await screen.findByRole("heading", { name: /sign in/i }),
  ).toBeInTheDocument();
});

test("unknown path renders not found", async () => {
  mockFetch({ "GET /me": me });
  renderAt("/nope");
  expect(await screen.findByText(/not found/i)).toBeInTheDocument();
});
