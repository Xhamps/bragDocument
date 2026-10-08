import { createBrowserRouter, type RouteObject } from "react-router";
import { RequireAuth } from "./auth/RequireAuth";

export const routes: RouteObject[] = [
  { path: "/sign-in", lazy: () => import("./routes/SignIn") },
  { path: "/auth/callback", lazy: () => import("./routes/AuthCallback") },
  {
    element: <RequireAuth />,
    children: [
      {
        path: "/",
        lazy: () => import("./routes/Root"),
        children: [
          { index: true, lazy: () => import("./routes/Documents") },
          { path: "tenant", lazy: () => import("./routes/Tenant") },
          ...(import.meta.env.DEV
            ? [
                {
                  path: "kitchen-sink",
                  lazy: () => import("./routes/KitchenSink"),
                },
              ]
            : []),
          { path: "*", lazy: () => import("./routes/NotFound") },
        ],
      },
    ],
  },
];

export const router = createBrowserRouter(routes);
