import { render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router";
import { routes } from "../router";

test("shell renders the app title and the home page", async () => {
  const router = createMemoryRouter(routes, { initialEntries: ["/"] });
  render(<RouterProvider router={router} />);
  expect(
    await screen.findByRole("heading", { name: "Brag Document" }),
  ).toBeInTheDocument();
  expect(await screen.findByText(/nothing here yet/i)).toBeInTheDocument();
});

test("unknown path renders not found", async () => {
  const router = createMemoryRouter(routes, { initialEntries: ["/nope"] });
  render(<RouterProvider router={router} />);
  expect(await screen.findByText(/not found/i)).toBeInTheDocument();
});
