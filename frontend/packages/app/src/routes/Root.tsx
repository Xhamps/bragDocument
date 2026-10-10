import { Outlet, useNavigate } from "react-router";
import { IconButton, TopBar, useTheme, type UserMenuItem } from "@bragdoc/ui";
import { routerLink } from "../lib/routerLink";
import { Logo } from "../layout/Logo";
import { useMe } from "../auth/useMe";
import { useDocuments } from "../documents/useDocuments";
import { supabase } from "../lib/supabase";

export function Component() {
  const { data: me } = useMe();
  const { data: docs } = useDocuments();
  const canAudit = me?.role === "admin" || (docs?.owned.length ?? 0) > 0;
  const navigate = useNavigate();
  const { theme, toggle } = useTheme();

  async function signOut() {
    await supabase.auth.signOut();
    navigate("/sign-in");
  }

  const links: (UserMenuItem | false)[] = [
    me?.role === "admin" && {
      label: me.tenant.name,
      href: "/tenant",
      icon: "user",
    },
    canAudit && { label: "Audit log", href: "/audit", icon: "history" },
    import.meta.env.DEV && {
      label: "Kitchen sink",
      href: "/kitchen-sink",
      icon: "grid",
    },
    { label: "Settings", href: "/settings", icon: "settings" },
  ];

  return (
    <div className="min-h-screen">
      <div className="mx-auto max-w-5xl p-4">
        {/* Never pass onSelect: it preventDefaults link clicks and blocks router navigation. */}
        <TopBar
          variant="floating"
          // Fill the page container so the pill lines up with the content.
          className="px-0 pt-0 [&>div]:max-w-none"
          brand={{ name: "Brag Document", href: "/", logo: <Logo /> }}
          renderLink={routerLink}
          actions={
            <IconButton
              icon={theme === "dark" ? "sun" : "moon"}
              label="Toggle dark mode"
              pressed={theme === "dark"}
              onClick={toggle}
            />
          }
          user={{
            user: {
              name: me?.display_name || me?.email || "",
              // Under the name; skipped when the name already is the email.
              email: me?.display_name ? me.email : undefined,
            },
            renderLink: routerLink,
            items: [
              ...links.filter((i) => i !== false),
              "separator",
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
