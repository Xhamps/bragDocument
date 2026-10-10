import { lazy, Suspense } from "react";
import { Navigate, Outlet, useLocation } from "react-router";
import { useAuth } from "./useAuth";

// Visitors at "/" get the public home page; everything else stays protected.
const Home = lazy(() =>
  import("../routes/Home").then((m) => ({ default: m.Component })),
);

export function RequireAuth() {
  const { session, loading } = useAuth();
  const location = useLocation();
  if (loading) return <p className="p-4 text-fg-secondary">Loading…</p>;
  if (!session) {
    if (location.pathname === "/")
      return (
        <Suspense fallback={null}>
          <Home />
        </Suspense>
      );
    return <Navigate to="/sign-in" state={{ from: location }} replace />;
  }
  return <Outlet />;
}
