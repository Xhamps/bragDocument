import { useState } from "react";
import {
  Link,
  Navigate,
  useLocation,
  useNavigate,
  type Location,
} from "react-router";
import { Button } from "@bragdoc/ui";
import { useAuth } from "../auth/useAuth";
import { AuthCard, Field } from "../auth/AuthCard";
import { callbackUrl, useAuthAction } from "../auth/useAuthAction";
import { supabase } from "../lib/supabase";

export function Component() {
  const { session, loading } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const { busy, run, setMessage, status } = useAuthAction();

  const fromLoc = (location.state as { from?: Location } | null)?.from;
  const from = fromLoc ? fromLoc.pathname + fromLoc.search + fromLoc.hash : "/";
  if (!loading && session) return <Navigate to={from} replace />;

  const signIn = () =>
    run(
      () => supabase.auth.signInWithPassword({ email, password }),
      () => navigate(from, { replace: true }),
    );
  const magicLink = () =>
    run(
      () =>
        supabase.auth.signInWithOtp({
          email,
          options: { emailRedirectTo: callbackUrl() },
        }),
      () => setMessage("Check your email for the sign-in link."),
    );
  const google = () =>
    run(() =>
      supabase.auth.signInWithOAuth({
        provider: "google",
        options: { redirectTo: callbackUrl() },
      }),
    );

  return (
    <AuthCard
      title="Sign in"
      description="Use your company account or an email link."
      onSubmit={signIn}
      footer={
        <>
          <Link to="/sign-up" className="underline">
            Create an account
          </Link>
          <Link to="/reset-password" className="underline">
            Forgot your password?
          </Link>
        </>
      }
    >
      <Field
        id="email"
        label="Email"
        type="email"
        required
        value={email}
        onChange={setEmail}
      />
      <Field
        id="password"
        label="Password"
        type="password"
        value={password}
        onChange={setPassword}
      />
      {status}
      <Button variant="primary" type="submit" disabled={busy}>
        Sign in
      </Button>
      <Button
        type="button"
        variant="glass"
        disabled={busy || !email}
        onClick={magicLink}
      >
        Send magic link
      </Button>
      <Button type="button" variant="glass" disabled={busy} onClick={google}>
        Continue with Google
      </Button>
    </AuthCard>
  );
}
