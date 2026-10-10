import type * as React from "react";
import { cn, focusRing } from "#lib/utils";
import { Icon } from "#components/icon";

type PageHeaderProps = {
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  /** The last crumb is the current page. */
  breadcrumbs?: Array<{ label: string; href?: string }>;
  /** Tags above the title. */
  eyebrow?: React.ReactNode;
  /** A row of small facts under the subtitle (dates, owners, counts). */
  meta?: React.ReactNode;
  /** Buttons aligned to the right of the title. */
  actions?: React.ReactNode;
  /** Usually a SegmentedControl switching the page's views. */
  tabs?: React.ReactNode;
  gradient?: "primary" | "secondary" | "accent";
  as?: "h1" | "h2";
  className?: string;
};

const GRADIENT = {
  primary: "text-gradient",
  secondary: "text-gradient-secondary",
  accent: "text-gradient-accent",
};

function PageHeader({
  title,
  subtitle,
  breadcrumbs = [],
  eyebrow,
  meta,
  actions,
  tabs,
  gradient,
  as: Title = "h1",
  className,
}: PageHeaderProps) {
  return (
    <div
      data-slot="page-header"
      className={cn("flex flex-col gap-3 px-6 pt-8 pb-4", className)}
    >
      {breadcrumbs.length > 0 && (
        <nav aria-label="Breadcrumb">
          <ol className="m-0 flex list-none flex-wrap items-center gap-1 p-0 type-footnote">
            {breadcrumbs.map((c, i) => {
              const last = i === breadcrumbs.length - 1;
              return (
                <li key={i} className="inline-flex items-center gap-1">
                  {last ? (
                    <span
                      aria-current="page"
                      className="font-medium text-fg-primary"
                    >
                      {c.label}
                    </span>
                  ) : (
                    <>
                      <a
                        href={c.href ?? "#"}
                        className={cn(
                          "rounded-sm text-fg-secondary no-underline transition-colors duration-fast hover:text-button-text",
                          focusRing,
                        )}
                      >
                        {c.label}
                      </a>
                      <Icon
                        name="chevron"
                        size={14}
                        className="text-fg-tertiary"
                      />
                    </>
                  )}
                </li>
              );
            })}
          </ol>
        </nav>
      )}
      <div className="flex flex-wrap items-end justify-between gap-6">
        <div className="flex min-w-0 flex-col gap-2">
          {eyebrow && <div className="flex gap-2">{eyebrow}</div>}
          <Title
            className={cn(
              "m-0 pb-0.5 type-title-1 text-fg-primary",
              gradient && GRADIENT[gradient],
            )}
          >
            {title}
          </Title>
          {subtitle && (
            <p className="m-0 max-w-[640px] type-body text-fg-secondary">
              {subtitle}
            </p>
          )}
          {meta && (
            <div className="flex flex-wrap items-center gap-3 text-[13px] text-fg-secondary">
              {meta}
            </div>
          )}
        </div>
        {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
      </div>
      {tabs && <div className="mt-2">{tabs}</div>}
    </div>
  );
}

export { PageHeader, type PageHeaderProps };
