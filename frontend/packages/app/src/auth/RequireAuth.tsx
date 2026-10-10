import { Navigate, Outlet, useLocation } from "react-router";
import { useAuth } from "./useAuth";

export function RequireAuth() {
  const { session, loading } = useAuth();
  const location = useLocation();
  if (loading) return <p className="p-4 text-fg-secondary">Loading…</p>;
  if (!session)
    return <Navigate to="/sign-in" state={{ from: location }} replace />;
  return <Outlet />;
}
