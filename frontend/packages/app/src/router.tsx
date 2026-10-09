import { createBrowserRouter, type RouteObject } from "react-router";
import { RequireAuth } from "./auth/RequireAuth";

export const routes: RouteObject[] = [
  { path: "/sign-in", lazy: () => import("./routes/SignIn") },
  { path: "/sign-up", lazy: () => import("./routes/SignUp") },
  { path: "/reset-password", lazy: () => import("./routes/ResetPassword") },
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
          { path: "settings", lazy: () => import("./routes/Settings") },
          { path: "audit", lazy: () => import("./routes/Audit") },
          {
            path: "documents/:id",
            lazy: () => import("./routes/DocumentLogs"),
          },
          {
            path: "documents/:id/dashboard",
            lazy: () => import("./routes/Dashboard"),
          },
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
