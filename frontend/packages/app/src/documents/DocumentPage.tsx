import type { ReactNode } from "react";
import { Tabs, Tag } from "@bragdoc/ui";
import { useMe } from "../auth/useMe";
import { Header } from "../layout/Header";
import { routerLink } from "../lib/routerLink";
import type { Document } from "../lib/types";

type Tab = "logs" | "dashboard" | "activity";

/** A document page: crumbs, header (title, sharing facts, actions), then the Logs | Dashboard | Activity tabs holding the view. */
export function DocumentPage({
  doc,
  current,
  actions,
  children,
}: {
  doc: Document;
  current: Tab;
  actions?: ReactNode;
  children: ReactNode;
}) {
  const { data: me } = useMe();
  const base = `/documents/${doc.id}`;
  const tabs = [
    { key: "logs", label: "Logs", href: base },
    { key: "dashboard", label: "Dashboard", href: `${base}/dashboard` },
    // Owners, and tenant admins who can open the page (PRD-0009 FR-11).
    ...(doc.role === "owner" || me?.role === "admin"
      ? [{ key: "activity", label: "Activity", href: `${base}/activity` }]
      : []),
  ];
  const here = tabs.find((t) => t.key === current)!;
  const meta = [
    doc.state === "archived" && <Tag key="a">Archived · read-only</Tag>,
    doc.role === "viewer" && <Tag key="v">Viewer · read-only</Tag>,
    doc.role !== "owner" && (
      <span key="s">{`Shared by ${doc.owner_name} · you are ${doc.role}`}</span>
    ),
  ].filter(Boolean);
  return (
    <>
      <Header
        breadcrumbs={[
          { label: "Documents", href: "/" },
          { label: doc.title, href: base },
          { label: here.label },
        ]}
        title={doc.title}
        subtitle={doc.description || undefined}
        meta={meta.length ? meta : undefined}
        actions={actions}
      />
      <Tabs
        label="Document views"
        items={tabs}
        value={here.href}
        renderLink={routerLink}
      >
        <div className="flex flex-col gap-6">{children}</div>
      </Tabs>
    </>
  );
}
