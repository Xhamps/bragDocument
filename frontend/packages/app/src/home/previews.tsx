import { Button, Card, MediaCell, Tag } from "@bragdoc/ui";

export function LogPreview() {
  return (
    <div className="flex flex-col gap-4">
      <div className="ml-auto max-w-xs rounded-lg rounded-br-sm bg-button px-4 py-3 type-callout text-button-fg shadow-md">
        /log Shipped the new landing page, signups up 18%
      </div>
      <Card>
        <div className="flex items-start justify-between gap-4">
          <p className="type-title-3">New landing page</p>
          <div className="flex gap-2">
            <Tag tone="accent">high</Tag>
            <Tag variant="outline">Done</Tag>
          </div>
        </div>
        <p className="mt-2 type-body text-fg-secondary">
          Rebuilt the page on the design system. Signups went up 18% in the
          first month.
        </p>
        <div className="mt-3 flex gap-2">
          <Tag variant="outline">#growth</Tag>
          <Tag variant="outline">#frontend</Tag>
        </div>
      </Card>
    </div>
  );
}

const BARS = [3, 5, 2, 6, 4, 7, 5, 8, 6, 9, 7, 11];

export function DashboardPreview() {
  return (
    <div className="flex h-full flex-col gap-4">
      {[
        ["Total logs", "68"],
        ["High or critical", "21"],
        ["In progress", "4"],
      ].map(([label, value]) => (
        <Card
          key={label}
          compact
          className="flex items-baseline justify-between"
        >
          <p className="type-footnote text-fg-secondary">{label}</p>
          <p className="type-title-2">{value}</p>
        </Card>
      ))}
      <Card compact className="flex min-h-40 flex-1 flex-col">
        <p className="type-callout">Logs per month</p>
        <div className="mt-4 flex flex-1 items-end gap-1.5">
          {BARS.map((n, i) => (
            <div
              key={i}
              className={
                i === BARS.length - 1
                  ? "flex-1 rounded-sm bg-chart-1"
                  : "flex-1 rounded-sm bg-chart-muted"
              }
              style={{ height: `${(n / Math.max(...BARS)) * 100}%` }}
            />
          ))}
        </div>
      </Card>
    </div>
  );
}

export function SharePreview() {
  return (
    <Card>
      <div className="flex items-center justify-between gap-4">
        <MediaCell title="Sam Rivera" subtitle="sam@acme.com" />
        <Tag tone="accent">Editor</Tag>
      </div>
      <div className="mt-4 flex items-center justify-between gap-4 border-t border-divider pt-4">
        <MediaCell title="Ana Costa" subtitle="ana@acme.com" />
        <Tag>Viewer</Tag>
      </div>
      <div className="mt-6 flex items-center gap-4">
        <div
          className="aspect-[3/4] w-20 rounded-md shadow-md"
          style={{ background: "var(--gradient-blue-1)" }}
        />
        <div className="flex flex-col gap-2">
          <p className="type-callout">Brag document, 2026</p>
          <Button variant="primary" icon="download">
            Export PDF
          </Button>
        </div>
      </div>
    </Card>
  );
}

export function TelegramPreview() {
  return (
    <p className="type-code font-mono">
      <span className="text-button-text">/log</span> Fixed the flaky deploy
    </p>
  );
}

export function AuditPreview() {
  return (
    <div className="flex flex-col gap-1 type-code font-mono text-fg-secondary">
      <p>10:42 ana edited “New landing page”</p>
      <p>09:15 sam shared the document</p>
    </div>
  );
}
