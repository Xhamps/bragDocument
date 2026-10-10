import * as React from "react";
import { cn, focusRing } from "#lib/utils";
import { useIndicator } from "#lib/use-indicator";

type TabItem = { label: string; href: string };

/** Props for one rendered tab link; spread them onto your router's link (map `href` to `to`). */
type TabLinkProps = {
  href: string;
  className: string;
  "data-on": boolean;
  "aria-current"?: "page";
  children: React.ReactNode;
};

type TabsProps = {
  items: TabItem[];
  /** The current tab's href. */
  value?: string;
  /** Accessible name of the nav. */
  label?: string;
  /** Renders every tab link. Defaults to a plain `<a>`. */
  renderLink?: (item: TabItem, props: TabLinkProps) => React.ReactNode;
  /** The current view; rendered in a panel joined to the tab strip. */
  children?: React.ReactNode;
  className?: string;
};

const linkClass = cn(
  "relative z-1 inline-flex h-12 items-center rounded-sm px-3 text-[15px] font-medium whitespace-nowrap text-fg-secondary no-underline transition-colors duration-fast ease-standard hover:text-fg-primary data-[on=true]:text-fg-primary",
  focusRing,
);

/** Page-level navigation between views: a strip of links with a sliding underline over a panel holding the current view. */
function Tabs({
  items,
  value,
  label = "Views",
  renderLink = (_, props) => <a {...props} />,
  children,
  className,
}: TabsProps) {
  const { ref, style, animated } = useIndicator<HTMLDivElement>([
    value,
    items.length,
  ]);
  return (
    <section
      data-slot="tabs"
      className={cn("rounded-lg glass shadow-glass", className)}
    >
      <nav aria-label={label} className="border-b px-3">
        <div ref={ref} className="relative -mb-px flex gap-1 overflow-x-auto">
          <span
            aria-hidden="true"
            style={style}
            className={cn(
              "pointer-events-none absolute top-0 left-0 z-0 after:absolute after:inset-x-3 after:bottom-0 after:h-0.5 after:rounded-[2px] after:bg-button-text",
              animated &&
                "[transition:transform_var(--duration-slow)_var(--ease-spring),width_var(--duration-slow)_var(--ease-spring),opacity_var(--duration-fast)_var(--ease-standard)]",
            )}
          />
          {items.map((it) => {
            const on = it.href === value;
            return (
              <React.Fragment key={it.href}>
                {renderLink(it, {
                  href: it.href,
                  className: linkClass,
                  "data-on": on,
                  "aria-current": on ? "page" : undefined,
                  children: it.label,
                })}
              </React.Fragment>
            );
          })}
        </div>
      </nav>
      {children != null && <div className="p-4 md:p-6">{children}</div>}
    </section>
  );
}

export { Tabs, type TabItem, type TabLinkProps, type TabsProps };
