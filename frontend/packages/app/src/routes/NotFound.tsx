import { Header } from "../layout/Header";

export function Component() {
  return (
    <div className="flex flex-col gap-6">
      <Header
        breadcrumbs={[
          { label: "Documents", href: "/" },
          { label: "Not found" },
        ]}
        title="Page not found"
        subtitle="This page does not exist or was moved."
      />
    </div>
  );
}
