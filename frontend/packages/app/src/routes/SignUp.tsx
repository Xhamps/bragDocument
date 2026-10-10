import { useState } from "react";
import { Link, Navigate } from "react-router";
import { Button } from "@bragdoc/ui";
import { useAuth } from "../auth/useAuth";
import { AuthCard, Field } from "../auth/AuthCard";
import { callbackUrl, useAuthAction } from "../auth/useAuthAction";
import { supabase } from "../lib/supabase";

export function Component() {
  const { session, loading } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const { busy, run, setMessage, status } = useAuthAction();

  if (!loading && session) return <Navigate to="/" replace />;

  const signUp = () =>
    run(
      () =>
        supabase.auth.signUp({
          email,
          password,
          options: { emailRedirectTo: callbackUrl() },
        }),
      () => setMessage("Check your email to confirm your account."),
    );

  return (
    <AuthCard
      title="Create account"
      description="Sign up with your email and a password."
      onSubmit={signUp}
      footer={
        <Link to="/sign-in" className="underline">
          Already have an account? Sign in
        </Link>
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
        required
        value={password}
        onChange={setPassword}
      />
      {status}
      <Button variant="primary" type="submit" disabled={busy}>
        Create account
      </Button>
    </AuthCard>
  );
}
