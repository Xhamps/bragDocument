import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderAt, supabaseMock } from "../test/mocks";

beforeEach(() => {
  supabaseMock.auth.getSession.mockResolvedValue({
    data: { session: null },
  } as never);
});

test("sign-up calls supabase and asks to confirm the email", async () => {
  renderAt("/sign-up");
  await userEvent.type(await screen.findByLabelText(/email/i), "a@acme.com");
  await userEvent.type(screen.getByLabelText(/password/i), "hunter22");
  await userEvent.click(
    screen.getByRole("button", { name: /create account/i }),
  );
  expect(supabaseMock.auth.signUp).toHaveBeenCalledWith(
    expect.objectContaining({ email: "a@acme.com", password: "hunter22" }),
  );
  expect(await screen.findByText(/check your email/i)).toBeInTheDocument();
});
