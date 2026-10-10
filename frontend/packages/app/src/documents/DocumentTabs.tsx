import { Link } from "react-router";

const base = "border-b-2 pb-1 text-sm";

/** Logs | Dashboard switch for one document. */
export function DocumentTabs({
  id,
  current,
}: {
  id: string;
  current: "logs" | "dashboard";
}) {
  const tab = (to: string, label: string, key: typeof current) => (
    <Link
      to={to}
      aria-current={current === key ? "page" : undefined}
      className={
        current === key
          ? `${base} border-fg-primary font-medium`
          : `${base} border-transparent text-fg-secondary hover:text-fg-primary`
      }
    >
      {label}
    </Link>
  );
  return (
    <nav aria-label="Document views" className="flex gap-4">
      {tab(`/documents/${id}`, "Logs", "logs")}
      {tab(`/documents/${id}/dashboard`, "Dashboard", "dashboard")}
    </nav>
  );
}
