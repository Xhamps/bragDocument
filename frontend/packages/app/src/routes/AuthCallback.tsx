import { useEffect } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../auth/useAuth";

export function Component() {
  const { session, loading } = useAuth();
  const navigate = useNavigate();
  useEffect(() => {
    if (!loading && session) navigate("/", { replace: true });
  }, [loading, session, navigate]);
  if (!loading && !session)
    return (
      <p className="p-4">
        Sign-in link is invalid or expired.{" "}
        <a href="/sign-in" className="underline">
          Try again
        </a>
        .
      </p>
    );
  return <p className="p-4 text-muted-foreground">Signing you in…</p>;
}
