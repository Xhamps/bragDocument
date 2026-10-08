import { useEffect } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { useAuth } from "../auth/useAuth";

export function Component() {
  const { session, loading } = useAuth();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const description = params.get("error_description");
  useEffect(() => {
    if (!loading && session) navigate("/", { replace: true });
  }, [loading, session, navigate]);
  if (!loading && !session)
    return (
      <p className="p-4">
        {description ?? "Sign-in link is invalid or expired."}{" "}
        <Link to="/sign-in" className="underline">
          Try again
        </Link>
        .
      </p>
    );
  return <p className="p-4 text-muted-foreground">Signing you in…</p>;
}
