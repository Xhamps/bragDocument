import { Link, Outlet, useNavigate } from "react-router";
import { Button } from "@bragdoc/ui";
import { useMe } from "../auth/useMe";
import { supabase } from "../lib/supabase";

export function Component() {
  const { data: me } = useMe();
  const navigate = useNavigate();

  async function signOut() {
    await supabase.auth.signOut();
    navigate("/sign-in");
  }

  return (
    <div className="min-h-screen">
      <header className="border-b">
        <nav className="mx-auto flex max-w-5xl items-center gap-6 p-4">
          <h1 className="text-lg font-semibold">
            <Link to="/">Brag Document</Link>
          </h1>
          {me?.role === "admin" && (
            <Link to="/tenant" className="text-sm text-muted-foreground">
              {me.tenant.name}
            </Link>
          )}
          {import.meta.env.DEV && (
            <Link to="/kitchen-sink" className="text-sm text-muted-foreground">
              Kitchen sink
            </Link>
          )}
          <span className="ml-auto text-sm text-muted-foreground">
            {me?.email}
          </span>
          <Button variant="ghost" size="sm" onClick={signOut}>
            Sign out
          </Button>
        </nav>
      </header>
      <main className="mx-auto max-w-5xl p-4">
        <Outlet />
      </main>
    </div>
  );
}
