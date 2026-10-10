import { Link, Outlet, useLocation, useNavigate } from "react-router";
import { IconButton, TopBar, useTheme, type TopBarItem } from "@bragdoc/ui";
import { useMe } from "../auth/useMe";
import { useDocuments } from "../documents/useDocuments";
import { supabase } from "../lib/supabase";

export function Component() {
  const { data: me } = useMe();
  const { data: docs } = useDocuments();
  const canAudit = me?.role === "admin" || (docs?.owned.length ?? 0) > 0;
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const { theme, toggle } = useTheme();

  async function signOut() {
    await supabase.auth.signOut();
    navigate("/sign-in");
  }

  const items = [
    me?.role === "admin" && { label: me.tenant.name, href: "/tenant" },
    canAudit && { label: "Audit log", href: "/audit" },
    import.meta.env.DEV && { label: "Kitchen sink", href: "/kitchen-sink" },
    { label: "Settings", href: "/settings" },
  ].filter((i): i is TopBarItem & { href: string } => Boolean(i));
  // TopBar highlights the first item without a value; "" = nothing current (e.g. "/").
  const current =
    items.find((i) => pathname === i.href || pathname.startsWith(`${i.href}/`))
      ?.label ?? "";

  return (
    <div className="min-h-screen bd-backdrop">
      <div className="mx-auto max-w-5xl p-4">
        {/* Never pass onSelect: it preventDefaults link clicks and blocks router navigation. */}
        <TopBar
          brand={{ name: "Brag Document", href: "/" }}
          items={items}
          value={current}
          renderLink={(_item, { href, ...props }) => (
            <Link to={href} {...props} />
          )}
          actions={
            <IconButton
              icon={theme === "dark" ? "sun" : "moon"}
              label="Toggle dark mode"
              pressed={theme === "dark"}
              onClick={toggle}
            />
          }
          user={{
            variant: "pill",
            user: { name: me?.email ?? "" },
            items: [
              {
                label: "Sign out",
                icon: "signOut",
                tone: "danger",
                onSelect: signOut,
              },
            ],
          }}
        />
      </div>
      <main className="mx-auto max-w-5xl p-4">
        <Outlet />
      </main>
    </div>
  );
}
