import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt, supabaseMock } from "../test/mocks";

test("without a session it emails a reset link back to this page", async () => {
  supabaseMock.auth.getSession.mockResolvedValueOnce({
    data: { session: null },
  } as never);
  renderAt("/reset-password");
  await userEvent.type(await screen.findByLabelText(/email/i), "a@acme.com");
  await userEvent.click(screen.getByRole("button", { name: /send reset/i }));
  expect(supabaseMock.auth.resetPasswordForEmail).toHaveBeenCalledWith(
    "a@acme.com",
    { redirectTo: `${window.location.origin}/reset-password` },
  );
  expect(await screen.findByText(/check your email/i)).toBeInTheDocument();
});

test("with a recovery session it sets the new password", async () => {
  mockFetch({ "GET /me": me, "GET /documents": { owned: [], shared: [] } });
  renderAt("/reset-password");
  await userEvent.type(
    await screen.findByLabelText(/new password/i),
    "hunter22",
  );
  await userEvent.click(
    screen.getByRole("button", { name: /update password/i }),
  );
  expect(supabaseMock.auth.updateUser).toHaveBeenCalledWith({
    password: "hunter22",
  });
  expect(
    await screen.findByRole("heading", { name: "Brag Document" }),
  ).toBeInTheDocument();
});
