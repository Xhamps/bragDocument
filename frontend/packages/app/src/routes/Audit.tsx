import { AuditView } from "../audit/AuditView";
import { useAudit, useAuditParams } from "../audit/useAudit";
import { Header } from "../layout/Header";

export function Component() {
  const url = useAuditParams();
  const audit = useAudit(url.filters);
  return (
    <div className="flex flex-col gap-6">
      <Header
        breadcrumbs={[
          { label: "Documents", href: "/" },
          { label: "Audit log" },
        ]}
        title="Audit log"
        subtitle="Who changed what across your documents."
      />
      <section className="flex flex-col gap-4 rounded-lg glass p-4 shadow-glass md:p-6">
        <AuditView query={audit} url={url} />
      </section>
    </div>
  );
}
