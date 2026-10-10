import type * as React from "react";
import { cn, focusRing } from "#lib/utils";
import { Icon } from "#components/icon";

type BreadcrumbItem = { label: string; href?: string };

/** Props for one rendered crumb link; spread them onto your router's link (map `href` to `to`). */
type BreadcrumbLinkProps = {
  href: string;
  className: string;
  children: React.ReactNode;
};

type BreadcrumbProps = {
  /** The last item is the current page. */
  items: BreadcrumbItem[];
  /** Renders every crumb link. Defaults to a plain `<a>`. */
  renderLink?: (
    item: BreadcrumbItem,
    props: BreadcrumbLinkProps,
  ) => React.ReactNode;
  className?: string;
};

const linkClass = cn(
  "rounded-sm text-fg-secondary no-underline transition-colors duration-fast hover:text-button-text",
  focusRing,
);

function Breadcrumb({
  items,
  renderLink = (_, props) => <a {...props} />,
  className,
}: BreadcrumbProps) {
  if (items.length === 0) return null;
  return (
    <nav aria-label="Breadcrumb" data-slot="breadcrumb" className={className}>
      <ol className="m-0 flex list-none flex-wrap items-center gap-1 p-0 type-footnote">
        {items.map((c, i) => {
          const last = i === items.length - 1;
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
                  {c.href ? (
                    renderLink(c, {
                      href: c.href,
                      className: linkClass,
                      children: c.label,
                    })
                  ) : (
                    <span className="text-fg-secondary">{c.label}</span>
                  )}
                  <Icon name="chevron" size={14} className="text-fg-tertiary" />
                </>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}

export {
  Breadcrumb,
  type BreadcrumbItem,
  type BreadcrumbLinkProps,
  type BreadcrumbProps,
};
