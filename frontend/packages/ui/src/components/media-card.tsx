import type * as React from "react";
import { cn } from "#lib/utils";
import { Card, type CardProps } from "#components/card";
import { avatarClass, initials } from "#components/media-cell";

// White-on-scrim restyles for Tags and glass Buttons inside the overlay layout.
const overlayBody = [
  "relative z-1 flex-1 justify-end p-6",
  "[&_[data-slot=tag]]:border-white/40 [&_[data-slot=tag]]:bg-white/12 [&_[data-slot=tag]]:text-white [&_[data-slot=tag]]:shadow-none",
  "[&_[data-slot=button][data-variant=glass]]:border-white/32 [&_[data-slot=button][data-variant=glass]]:bg-white/16 [&_[data-slot=button][data-variant=glass]]:text-white [&_[data-slot=button][data-variant=glass]]:shadow-none [&_[data-slot=button][data-variant=glass]:hover:not(:disabled)]:bg-white/26",
];

type MediaCardProps = Omit<
  CardProps,
  | "title"
  | "description"
  | "action"
  | "compact"
  | "elevation"
  | "as"
  | "onClick"
> & {
  title: React.ReactNode;
  /** Image URL, or a gradient token name ("gradient-blue-1") as an artwork placeholder. */
  image?: string;
  imageAlt?: string;
  /** vertical = image on top (default); horizontal = image left; overlay = text over the image behind a dark scrim. */
  layout?: "vertical" | "horizontal" | "overlay";
  /** CSS aspect-ratio of the image (vertical) or the whole card (overlay); default 16 / 9, overlay 4 / 5. */
  aspect?: string;
  /** A Tag (or several) above the title. */
  tag?: React.ReactNode;
  eyebrow?: React.ReactNode;
  description?: React.ReactNode;
  /** 0–100: a thin progress bar under the description. */
  progress?: number;
  progressLabel?: string;
  author?: { name: string; role?: string; avatar?: string };
  /** Overlay on the image's bottom-right corner, e.g. a duration "4:30". */
  badge?: React.ReactNode;
  /** A small IconButton on the image's top-right corner (save, play). */
  mediaAction?: React.ReactNode;
  actions?: React.ReactNode;
  /** Makes the whole card a link (the title gets the hit area). Use href or onClick, not both. */
  href?: string;
  /** Makes the card clickable and focusable. Use href or onClick, not both. */
  onClick?: () => void;
  titleAs?: "h2" | "h3" | "h4";
};

function MediaCard({
  title,
  image,
  imageAlt,
  layout = "vertical",
  aspect,
  tag,
  eyebrow,
  description,
  progress,
  progressLabel,
  author,
  badge,
  mediaAction,
  actions,
  href,
  titleAs: Title = "h3",
  interactive,
  onClick,
  className,
  style,
  ...props
}: MediaCardProps) {
  const overlay = layout === "overlay";
  const horizontal = layout === "horizontal";
  const isInteractive = !!(interactive || onClick || href);
  const ratio = aspect || (horizontal ? "1 / 1" : overlay ? "4 / 5" : "16 / 9");
  const imgClass = cn(
    "block size-full object-cover transition-[scale] duration-(--duration-enter) ease-standard",
    isInteractive && "group-hover/mcard:scale-105",
  );
  let media: React.ReactNode;
  if (image && image.startsWith("gradient-"))
    media = (
      <span
        className={imgClass}
        style={{ background: `var(--${image})` }}
        role={imageAlt ? "img" : undefined}
        aria-label={imageAlt}
      />
    );
  else if (image)
    media = (
      <img
        className={imgClass}
        src={image}
        alt={imageAlt || ""}
        loading="lazy"
      />
    );
  else
    media = (
      <span
        aria-hidden="true"
        className={cn(imgClass, "[background:var(--gradient-blue-3)]")}
      />
    );
  const onScrim = (base: string, light: string) => (overlay ? light : base);

  return (
    <Card
      as="article"
      interactive={isInteractive}
      onClick={onClick}
      style={{ "--ratio": ratio, ...style } as React.CSSProperties}
      {...props}
      data-slot="media-card"
      data-layout={layout}
      className={cn(
        "group/mcard relative flex flex-col overflow-hidden p-0",
        "has-[a[data-slot=media-card-link]:focus-visible]:outline-2 has-[a[data-slot=media-card-link]:focus-visible]:outline-offset-2 has-[a[data-slot=media-card-link]:focus-visible]:outline-focus-ring has-[a[data-slot=media-card-link]:focus-visible]:outline-solid",
        horizontal && "flex-row items-stretch",
        overlay && "aspect-(--ratio) min-h-[280px]",
        className,
      )}
    >
      <div
        className={cn(
          "relative flex-none overflow-hidden rounded-md bg-chart-muted",
          horizontal
            ? "my-2 ml-2 min-h-[120px] w-[38%] max-w-[200px]"
            : overlay
              ? "absolute inset-0 rounded-[inherit] after:absolute after:inset-0 after:bg-[linear-gradient(180deg,transparent_25%,rgba(4,6,23,0.55)_55%,rgba(4,6,23,0.9))]"
              : "mx-2 mt-2 aspect-(--ratio)",
        )}
      >
        {media}
        {badge != null && (
          <span className="absolute right-2 bottom-2 rounded-sm bg-black/72 px-1.5 py-px text-xs leading-[18px] font-semibold text-white tabular-nums">
            {badge}
          </span>
        )}
        {/* z-2 lifts it above the href hit area so it stays clickable (DS omits this). */}
        {mediaAction && (
          <span className="absolute top-2 right-2 z-2">{mediaAction}</span>
        )}
      </div>
      <div
        className={cn(
          "flex min-w-0 flex-1 flex-col gap-2 px-5 pt-4 pb-5",
          horizontal && "justify-center py-4",
          overlay && overlayBody,
        )}
      >
        {tag && <div className="flex flex-wrap gap-1">{tag}</div>}
        {eyebrow && (
          <div
            className={cn(
              "type-footnote",
              onScrim("text-fg-secondary", "text-white/80"),
            )}
          >
            {eyebrow}
          </div>
        )}
        <Title
          className={cn(
            "m-0",
            onScrim("type-title-3 text-fg-primary", "type-title-2 text-white"),
          )}
        >
          {href ? (
            <a
              href={href}
              data-slot="media-card-link"
              className="text-inherit no-underline outline-hidden after:absolute after:inset-0 after:z-1"
            >
              {title}
            </a>
          ) : (
            title
          )}
        </Title>
        {description && (
          <p
            className={cn(
              "m-0 line-clamp-3 type-body",
              onScrim("text-fg-secondary", "text-white/80"),
            )}
          >
            {description}
          </p>
        )}
        {progress != null && (
          <div
            role="progressbar"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={Math.max(0, Math.min(100, progress))}
            aria-label={progressLabel || "Progress"}
            className={cn(
              "mt-1 h-1 overflow-hidden rounded-pill",
              onScrim("bg-chart-muted", "bg-white/25"),
            )}
          >
            <span
              className="block h-full rounded-[inherit] bg-button transition-[width] duration-slow ease-standard"
              style={{ width: `${Math.max(0, Math.min(100, progress))}%` }}
            />
          </div>
        )}
        {author && (
          <div className="mt-2 flex items-center gap-3">
            {author.avatar ? (
              <img
                className={cn(avatarClass, "size-9 text-[13px]")}
                src={author.avatar}
                alt=""
              />
            ) : (
              <span
                className={cn(avatarClass, "size-9 text-[13px]")}
                aria-hidden="true"
              >
                {initials(author.name)}
              </span>
            )}
            <span className="flex flex-col">
              <span
                className={cn(
                  "type-callout",
                  onScrim("text-fg-primary", "text-white"),
                )}
              >
                {author.name}
              </span>
              {author.role && (
                <span
                  className={cn(
                    "type-footnote",
                    onScrim("text-fg-secondary", "text-white/80"),
                  )}
                >
                  {author.role}
                </span>
              )}
            </span>
          </div>
        )}
        {actions && (
          <div
            className={cn(
              "relative z-2 flex flex-wrap gap-2 pt-2",
              overlay ? "mt-2" : "mt-auto",
            )}
          >
            {actions}
          </div>
        )}
      </div>
    </Card>
  );
}

export { MediaCard, type MediaCardProps };
