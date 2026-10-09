import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";

test("admin sees members and invitations, invites, removes", async () => {
  const calls = mockFetch({
    "GET /me": me,
    "GET /tenant/audit": [],
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

test("explains 409s, clears invite error on edit, shows withdraw errors", async () => {
  mockFetch({
    "GET /me": me,
    "GET /tenant/audit": [],
    "GET /tenant/members": [
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
    "POST /tenant/invitations": {
      status: 409,
      body: { message: "conflict with current state" },
    },
    "DELETE /tenant/members/u2": {
      status: 409,
      body: { message: "conflict with current state" },
    },
    "DELETE /tenant/invitations/i1": {
      status: 404,
      body: { message: "invitation not found" },
    },
  });
  renderAt("/tenant");
  expect(await screen.findByText("c@acme.com")).toBeInTheDocument();

  const input = screen.getByLabelText(/invite by email/i);
  await userEvent.type(input, "b@acme.com");
  await userEvent.click(screen.getByRole("button", { name: /^invite$/i }));
  expect((await screen.findAllByRole("alert"))[0]).toHaveTextContent(
    "This email is already a member or already invited.",
  );
  await userEvent.type(input, "x");
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();

  await userEvent.click(
    screen.getByRole("button", { name: /remove b@acme\.com/i }),
  );
  expect((await screen.findAllByRole("alert"))[0]).toHaveTextContent(
    "Cannot remove: this member still owns documents.",
  );

  await userEvent.click(
    screen.getByRole("button", { name: /withdraw c@acme\.com/i }),
  );
  await vi.waitFor(() =>
    expect(screen.getAllByRole("alert").map((a) => a.textContent)).toContain(
      "invitation not found",
    ),
  );
});

test("shows empty invitations message", async () => {
  mockFetch({
    "GET /me": me,
    "GET /tenant/audit": [],
    "GET /tenant/members": [],
    "GET /tenant/invitations": [],
  });
  renderAt("/tenant");
  expect(
    await screen.findByText("No pending invitations."),
  ).toBeInTheDocument();
});

test("member is redirected home", async () => {
  const calls = mockFetch({
    "GET /me": { ...me, role: "member" },
    "GET /tenant/audit": [],
    "GET /documents": { owned: [], shared: [] },
  });
  renderAt("/tenant");
  expect(
    await screen.findByText(/what is a brag document/i),
  ).toBeInTheDocument();
  expect(calls.some((c) => c.path.startsWith("/tenant/"))).toBe(false);
});

test("admin sees the sharing audit and document invitations", async () => {
  mockFetch({
    "GET /me": me,
    "GET /tenant/members": [],
    "GET /tenant/invitations": [
      {
        id: "di1",
        email: "new@acme.com",
        created_at: "2026-01-03T00:00:00Z",
        document_title: "Bob 2026",
      },
    ],
    "GET /tenant/audit": [
      {
        id: 1,
        actor_email: "bob@acme.com",
        action: "grant",
        document_id: "d2",
        document_title: "Bob 2026",
        target: "ada@acme.com",
        role: "viewer",
        at: "2026-01-03T00:00:00Z",
      },
    ],
  });
  renderAt("/tenant");
  expect(
    await screen.findByText(
      /bob@acme\.com shared ada@acme\.com on “Bob 2026” as viewer/,
    ),
  ).toBeInTheDocument();
  expect(screen.getByText(/via “Bob 2026”/)).toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: /withdraw new@acme\.com/i }),
  ).not.toBeInTheDocument();
});
