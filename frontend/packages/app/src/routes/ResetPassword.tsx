import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { Button } from "@bragdoc/ui";
import { useAuth } from "../auth/useAuth";
import { AuthCard, Field } from "../auth/AuthCard";
import { callbackUrl, useAuthAction } from "../auth/useAuthAction";
import { supabase } from "../lib/supabase";

// ponytail: one page for both steps. The recovery email links back here; by
// then supabase-js has turned the link into a session, so a session means
// "choose a new password" and no session means "email me a link".
export function Component() {
  const { session, loading } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const { busy, run, setMessage, status } = useAuthAction();

  if (loading) return <p className="p-4 text-muted-foreground">Loading…</p>;

  if (session)
    return (
      <AuthCard
        title="Set a new password"
        description="Choose the password you will sign in with."
        onSubmit={() =>
          run(
            () => supabase.auth.updateUser({ password }),
            () => navigate("/", { replace: true }),
          )
        }
      >
        <Field
          id="password"
          label="New password"
          type="password"
          required
          value={password}
          onChange={setPassword}
        />
        {status}
        <Button type="submit" disabled={busy}>
          Update password
        </Button>
      </AuthCard>
    );

  return (
    <AuthCard
      title="Reset password"
      description="We will email you a link to choose a new password."
      onSubmit={() =>
        run(
          () =>
            supabase.auth.resetPasswordForEmail(email, {
              redirectTo: callbackUrl("/reset-password"),
            }),
          () => setMessage("Check your email for the reset link."),
        )
      }
      footer={
        <Link to="/sign-in" className="underline">
          Back to sign in
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
      {status}
      <Button type="submit" disabled={busy}>
        Send reset link
      </Button>
    </AuthCard>
  );
}
