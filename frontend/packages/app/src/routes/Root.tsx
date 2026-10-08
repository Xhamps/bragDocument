import { Link, Outlet } from "react-router";

export function Component() {
  return (
    <div className="min-h-screen">
      <header className="border-b">
        <nav className="mx-auto flex max-w-5xl items-center gap-6 p-4">
          <h1 className="text-lg font-semibold">
            <Link to="/">Brag Document</Link>
          </h1>
          {import.meta.env.DEV && (
            <Link to="/kitchen-sink" className="text-sm text-muted-foreground">
              Kitchen sink
            </Link>
          )}
        </nav>
      </header>
      <main className="mx-auto max-w-5xl p-4">
        <Outlet />
      </main>
    </div>
  );
}
