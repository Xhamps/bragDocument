import type * as React from "react";
import { cn } from "#lib/utils";
import { Button } from "#components/button";

const GRADIENT = {
  primary: "text-gradient",
  secondary: "text-gradient-secondary",
  accent: "text-gradient-accent",
  none: "",
} as const;

const SIZE = {
  display: "type-display max-[600px]:text-[40px] max-[600px]:leading-[44px]",
  "title-1": "type-title-1",
  "title-2": "type-title-2",
} as const;

type HeroHeaderProps = Omit<React.ComponentProps<"header">, "title"> & {
  /** The headline; it carries the gradient. */
  title: React.ReactNode;
  /** Optional words before the title, set in plain fg-primary. */
  lead?: React.ReactNode;
  subtitle?: React.ReactNode;
  /** A short announcement pill above the title (string = glass sm button), or any node such as a Tag. */
  eyebrow?: React.ReactNode;
  onEyebrowClick?: () => void;
  /** Buttons under the subtitle. */
  actions?: React.ReactNode;
  gradient?: keyof typeof GRADIENT;
  size?: keyof typeof SIZE;
  align?: "center" | "left";
  /** Heading element, default h1. */
  as?: "h1" | "h2" | "h3";
};

function HeroHeader({
  title,
  lead,
  subtitle,
  eyebrow,
  onEyebrowClick,
  actions,
  gradient = "primary",
  size = "display",
  align = "center",
  as: Heading = "h1",
  className,
  ...props
}: HeroHeaderProps) {
  const center = align === "center";
  return (
    <header
      data-slot="hero-header"
      {...props}
      className={cn(
        "box-border flex max-w-[880px] flex-col gap-4 px-6 py-12",
        center ? "mx-auto items-center text-center" : "items-start text-left",
        className,
      )}
    >
      {eyebrow && (
        <div>
          {typeof eyebrow === "string" ? (
            <Button
              size="sm"
              chevron={!!onEyebrowClick}
              onClick={onEyebrowClick}
            >
              {eyebrow}
            </Button>
          ) : (
            eyebrow
          )}
        </div>
      )}
      <Heading
        className={cn(
          "m-0 pb-[0.08em] text-fg-primary [&_span]:box-decoration-clone",
          SIZE[size],
        )}
      >
        {lead && <span>{lead} </span>}
        <span className={GRADIENT[gradient] || undefined}>{title}</span>
      </Heading>
      {subtitle && (
        <p
          className={cn(
            "m-0 max-w-[560px] text-fg-secondary",
            size === "display" ? "text-[17px] leading-[26px]" : "type-body",
          )}
        >
          {subtitle}
        </p>
      )}
      {actions && (
        <div
          className={cn(
            "mt-2 flex flex-wrap gap-3",
            center && "justify-center",
          )}
        >
          {actions}
        </div>
      )}
    </header>
  );
}

export { HeroHeader, type HeroHeaderProps };
