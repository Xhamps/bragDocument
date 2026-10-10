import { Breadcrumb, PageHeader, cn, type PageHeaderProps } from "@bragdoc/ui";
import { routerLink } from "../lib/routerLink";

/**
 * Every signed-in page's top: the breadcrumb as its own pill, then a glass band
 * that sets the title and actions apart from the content.
 */
export function Header({
  breadcrumbs = [],
  className,
  ...props
}: PageHeaderProps) {
  return (
    <>
      <Breadcrumb
        items={breadcrumbs}
        renderLink={routerLink}
        className="self-start rounded-pill glass px-4 py-2 shadow-glass"
      />
      <PageHeader
        as="h2"
        className={cn("rounded-lg glass px-6 py-6 shadow-glass", className)}
        {...props}
      />
    </>
  );
}
