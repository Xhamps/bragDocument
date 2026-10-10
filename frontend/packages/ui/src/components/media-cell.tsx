import type * as React from "react";
import { cn } from "#lib/utils";

/** First letters of the first two words, upper-cased ("Eva Solain" → "ES"). */
function initials(name: string | undefined) {
  return (name || "?")
    .split(/\s+/)
    .map((w) => w[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

// .bd-avatar, shared by NotificationItem and MediaCard authors.
const avatarClass =
  "box-border inline-flex size-10 flex-none items-center justify-center rounded-pill border border-container-border bg-chart-muted object-cover text-[14px] font-semibold text-fg-primary shadow-sm";

type MediaCellProps = Omit<React.ComponentProps<"span">, "title"> & {
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  /** Third, smaller line in fg-tertiary ("14k views · 1 month ago"). */
  meta?: React.ReactNode;
  /** An image URL, or a gradient token name ("gradient-blue-1") for an artwork placeholder. Omit for initials. */
  image?: string;
  imageAlt?: string;
  /** circle = person, rounded = app/brand icon, thumb = 16:9 video or course thumbnail. */
  shape?: "circle" | "rounded" | "thumb";
  size?: "md" | "lg";
  /** Overlay on the image's corner, e.g. a duration "4:30". */
  badge?: React.ReactNode;
};

function MediaCell({
  title,
  subtitle,
  meta,
  image,
  imageAlt,
  shape = "circle",
  size = "md",
  badge,
  className,
  ...props
}: MediaCellProps) {
  const lg = size === "lg";
  const imgClass = cn(
    "box-border block border border-container-border bg-chart-muted object-cover shadow-sm",
    shape === "circle" ? "rounded-pill" : "rounded-md",
    shape === "thumb"
      ? cn(
          lg ? "h-[72px] w-32" : "h-14 w-24",
          // Zooms when its DataTable row is hovered.
          "transition-[scale,box-shadow] duration-slow ease-standard group-hover/row:scale-104 group-hover/row:shadow-md",
        )
      : lg
        ? "size-12"
        : "size-10",
  );
  let media: React.ReactNode;
  if (image && image.startsWith("gradient-"))
    media = (
      <span
        aria-hidden="true"
        className={imgClass}
        style={{ background: `var(--${image})` }}
      />
    );
  else if (image)
    media = <img className={imgClass} src={image} alt={imageAlt || ""} />;
  else
    media = (
      <span
        aria-hidden="true"
        className={cn(
          imgClass,
          "inline-flex items-center justify-center type-callout font-semibold text-fg-primary",
        )}
      >
        {initials(typeof title === "string" ? title : "")}
      </span>
    );
  return (
    <span
      data-slot="media-cell"
      {...props}
      className={cn(
        "inline-flex max-w-full min-w-0 items-center gap-3 align-middle",
        className,
      )}
    >
      <span className="relative inline-flex flex-none">
        {media}
        {badge != null && (
          <span className="absolute right-1 bottom-1 rounded-sm bg-black/75 px-[5px] text-[11px] leading-4 font-semibold text-white tabular-nums">
            {badge}
          </span>
        )}
      </span>
      <span className="flex min-w-0 flex-col">
        <span className="line-clamp-2 type-callout whitespace-normal text-fg-primary">
          {title}
        </span>
        {subtitle && (
          <span className="truncate type-footnote text-fg-secondary">
            {subtitle}
          </span>
        )}
        {meta && (
          <span className="text-xs whitespace-nowrap text-fg-tertiary">
            {meta}
          </span>
        )}
      </span>
    </span>
  );
}

export { MediaCell, initials, avatarClass, type MediaCellProps };
