import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt } from "../test/mocks";

test("not linked: generate a code, copy it, open in Telegram", async () => {
  const user = userEvent.setup();
  const writeText = vi
    .spyOn(navigator.clipboard, "writeText")
    .mockResolvedValue();
  mockFetch({
    "GET /me": me,
    "GET /me/telegram": { linked: false },
    "POST /me/telegram/code": {
      code: "ABCD2345",
      expires_at: "2026-10-08T12:10:00Z",
      bot_url: "https://t.me/BragBot?start=ABCD2345",
    },
  });
  renderAt("/settings");
  expect(await screen.findByText("Not linked")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: /generate link code/i }));
  expect(await screen.findByText("ABCD2345")).toBeInTheDocument();
  expect(screen.getByText("/start ABCD2345")).toBeInTheDocument();
  expect(
    screen.getByRole("link", { name: /open in telegram/i }),
  ).toHaveAttribute("href", "https://t.me/BragBot?start=ABCD2345");
  await user.click(screen.getByRole("button", { name: /copy/i }));
  expect(writeText).toHaveBeenCalledWith("ABCD2345");
});

test("linked: unlink after confirming", async () => {
  let linked = true;
  const calls = mockFetch({
    "GET /me": me,
    "GET /me/telegram": () =>
      linked ? { linked, linked_at: "2026-10-01T00:00:00Z" } : { linked },
    "DELETE /me/telegram": () => {
      linked = false;
      return undefined;
    },
  });
  renderAt("/settings");
  expect(await screen.findByText(/linked since/i)).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: /^unlink$/i }));
  expect(calls.some((c) => c.method === "DELETE")).toBe(false);
  await userEvent.click(
    screen.getByRole("button", { name: /confirm unlink/i }),
  );
  expect(await screen.findByText("Not linked")).toBeInTheDocument();
  expect(calls.some((c) => c.method === "DELETE")).toBe(true);
});

test("code generation unavailable shows an error", async () => {
  mockFetch({
    "GET /me": me,
    "GET /me/telegram": { linked: false },
    "POST /me/telegram/code": {
      status: 503,
      body: { message: "service unavailable" },
    },
  });
  renderAt("/settings");
  await userEvent.click(
    await screen.findByRole("button", { name: /generate link code/i }),
  );
  expect(await screen.findByRole("alert")).toHaveTextContent(
    "service unavailable",
  );
});
