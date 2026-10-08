import { createBrowserRouter, type RouteObject } from "react-router";

export const routes: RouteObject[] = [
  {
    path: "/",
    lazy: () => import("./routes/Root"),
    children: [
      { index: true, lazy: () => import("./routes/Home") },
      ...(import.meta.env.DEV
        ? [{ path: "kitchen-sink", lazy: () => import("./routes/KitchenSink") }]
        : []),
      { path: "*", lazy: () => import("./routes/NotFound") },
    ],
  },
];

export const router = createBrowserRouter(routes);
