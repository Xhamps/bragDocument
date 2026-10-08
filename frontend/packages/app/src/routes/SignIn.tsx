import { useState, type FormEvent } from "react";
import {
  Navigate,
  useLocation,
  useNavigate,
  type Location,
} from "react-router";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  Label,
} from "@bragdoc/ui";
import { useAuth } from "../auth/useAuth";
import { supabase } from "../lib/supabase";

const callbackUrl = () => `${window.location.origin}/auth/callback`;

export function Component() {
  const { session, loading } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const fromLoc = (location.state as { from?: Location } | null)?.from;
  const from = fromLoc ? fromLoc.pathname + fromLoc.search + fromLoc.hash : "/";
  if (!loading && session) return <Navigate to={from} replace />;

  async function run(
    action: () => Promise<{ error: { message: string } | null }>,
    onOk?: () => void,
  ) {
    setBusy(true);
    setError(null);
    setMessage(null);
    const { error } = await action();
    setBusy(false);
    if (error) setError(error.message);
    else onOk?.();
  }

  const signIn = (e: FormEvent) => {
    e.preventDefault();
    run(
      () => supabase.auth.signInWithPassword({ email, password }),
      () => navigate(from, { replace: true }),
    );
  };
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
    <main className="mx-auto flex min-h-screen max-w-sm items-center p-4">
      <Card className="w-full">
        <CardHeader>
          <CardTitle>
            <h1>Sign in</h1>
          </CardTitle>
          <CardDescription>
            Use your company account or an email link.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={signIn} className="flex flex-col gap-3">
            <div className="flex flex-col gap-1">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
            {error && (
              <p role="alert" className="text-sm text-destructive">
                {error}
              </p>
            )}
            {message && (
              <p aria-live="polite" className="text-sm text-muted-foreground">
                {message}
              </p>
            )}
            <Button type="submit" disabled={busy}>
              Sign in
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={busy}
              onClick={signUp}
            >
              Create account
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={busy || !email}
              onClick={magicLink}
            >
              Send magic link
            </Button>
            <Button
              type="button"
              variant="secondary"
              disabled={busy}
              onClick={google}
            >
              Continue with Google
            </Button>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
