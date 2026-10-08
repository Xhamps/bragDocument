import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";

test("admin sees members and invitations, invites, removes", async () => {
  const calls = mockFetch({
    "GET /me": me,
    "GET /tenant/members": [
      {
        id: "u1",
        email: "a@acme.com",
        display_name: "Ada",
        role: "admin",
        created_at: "2026-01-01T00:00:00Z",
      },
      {
        id: "u2",
        email: "b@acme.com",
        display_name: "",
        role: "member",
        created_at: "2026-01-02T00:00:00Z",
      },
    ],
    "GET /tenant/invitations": [
      { id: "i1", email: "c@acme.com", created_at: "2026-01-03T00:00:00Z" },
    ],
    "POST /tenant/invitations": (init?: RequestInit) => ({
      id: "i2",
      ...JSON.parse(String(init?.body)),
      created_at: "",
    }),
    "DELETE /tenant/members/u2": undefined,
  });
  renderAt("/tenant");
  expect(await screen.findByText("b@acme.com")).toBeInTheDocument();
  expect(screen.getByText("c@acme.com")).toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: /remove a@acme\.com/i }),
  ).not.toBeInTheDocument();

  const input = screen.getByLabelText(/invite by email/i);
  await userEvent.type(input, "d@acme.com");
  await userEvent.click(screen.getByRole("button", { name: /^invite$/i }));
  await vi.waitFor(() =>
    expect(calls.find((c) => c.method === "POST")?.body).toEqual({
      email: "d@acme.com",
    }),
  );
  await vi.waitFor(() => expect(input).toHaveValue(""));

  await userEvent.click(
    screen.getByRole("button", { name: /remove b@acme\.com/i }),
  );
  await vi.waitFor(() =>
    expect(
      calls.some(
        (c) => c.method === "DELETE" && c.path === "/tenant/members/u2",
      ),
    ).toBe(true),
  );
});

test("shows invite and withdraw errors, empty invitations message", async () => {
  mockFetch({
    "GET /me": me,
    "GET /tenant/members": [],
    "GET /tenant/invitations": [
      { id: "i1", email: "c@acme.com", created_at: "2026-01-03T00:00:00Z" },
    ],
    "POST /tenant/invitations": {
      status: 409,
      body: { message: "already a member" },
    },
    "DELETE /tenant/invitations/i1": {
      status: 404,
      body: { message: "invitation not found" },
    },
  });
  renderAt("/tenant");
  expect(await screen.findByText("c@acme.com")).toBeInTheDocument();
  await userEvent.type(screen.getByLabelText(/invite by email/i), "b@acme.com");
  await userEvent.click(screen.getByRole("button", { name: /^invite$/i }));
  expect(await screen.findByRole("alert")).toHaveTextContent(
    "already a member",
  );
  await userEvent.click(
    screen.getByRole("button", { name: /withdraw c@acme\.com/i }),
  );
  expect(await screen.findByText("invitation not found")).toBeInTheDocument();
});

test("member is redirected home", async () => {
  mockFetch({
    "GET /me": { ...me, role: "member" },
    "GET /documents": { owned: [], shared: [] },
  });
  renderAt("/tenant");
  expect(
    await screen.findByText(/what is a brag document/i),
  ).toBeInTheDocument();
});
