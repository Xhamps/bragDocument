import { act, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { me, mockFetch, renderAt, supabaseMock } from "../test/mocks";

beforeEach(() => {
  supabaseMock.auth.getSession.mockResolvedValue({
    data: { session: null },
  } as never);
});

test("password sign-in calls supabase with the form values", async () => {
  mockFetch({ "GET /me": me, "GET /documents": { owned: [], shared: [] } });
  renderAt("/sign-in");
  await userEvent.type(await screen.findByLabelText(/email/i), "a@acme.com");
  await userEvent.type(screen.getByLabelText(/password/i), "hunter22");
  await userEvent.click(screen.getByRole("button", { name: /^sign in$/i }));
  expect(supabaseMock.auth.signInWithPassword).toHaveBeenCalledWith({
    email: "a@acme.com",
    password: "hunter22",
  });
  // supabase-js would fire SIGNED_IN here; RequireAuth then lets the shell render
  const onChange = supabaseMock.auth.onAuthStateChange.mock.calls[0][0];
  act(() => onChange("SIGNED_IN", { access_token: "tok" }));
  expect(
    await screen.findByRole("heading", { name: "Brag Document" }),
  ).toBeInTheDocument();
});

test("magic link reports that the email was sent", async () => {
  renderAt("/sign-in");
  await userEvent.type(await screen.findByLabelText(/email/i), "a@acme.com");
  await userEvent.click(screen.getByRole("button", { name: /magic link/i }));
  expect(supabaseMock.auth.signInWithOtp).toHaveBeenCalled();
  expect(await screen.findByText(/check your email/i)).toBeInTheDocument();
});

test("shows the provider error", async () => {
  supabaseMock.auth.signInWithPassword.mockResolvedValueOnce({
    error: { message: "Invalid login credentials" },
  } as never);
  renderAt("/sign-in");
  await userEvent.type(await screen.findByLabelText(/email/i), "a@acme.com");
  await userEvent.type(screen.getByLabelText(/password/i), "x");
  await userEvent.click(screen.getByRole("button", { name: /^sign in$/i }));
  expect(
    await screen.findByText(/invalid login credentials/i),
  ).toBeInTheDocument();
});

test("links to sign-up and reset password", async () => {
  renderAt("/sign-in");
  expect(
    await screen.findByRole("link", { name: /create an account/i }),
  ).toHaveAttribute("href", "/sign-up");
  expect(
    screen.getByRole("link", { name: /forgot your password/i }),
  ).toHaveAttribute("href", "/reset-password");
});
