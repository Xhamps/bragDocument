import type * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn, focusRing } from "#lib/utils";
import { renderIcon, type IconProp } from "#components/icon";
import { buttonColorVariants, glowClass } from "#components/button";
import { Tooltip, type TooltipProps } from "#components/tooltip";

const ICON_PX = { sm: 16, md: 20, lg: 24, xl: 32 } as const;

const iconButtonVariants = cva(
  [
    "relative inline-flex shrink-0 cursor-pointer items-center justify-center border border-transparent p-0",
    "transition-[background-color,border-color,color,box-shadow,translate,scale] duration-base ease-standard",
    "hover:-translate-y-px active:scale-[0.92] active:duration-fast active:ease-spring",
    "disabled:pointer-events-none disabled:cursor-default disabled:shadow-none",
    "[&>svg]:transition-transform [&>svg]:duration-base [&>svg]:ease-spring",
    focusRing,
  ],
  {
    variants: {
      size: { sm: "size-7", md: "size-9", lg: "size-11", xl: "size-16" },
    },
    defaultVariants: { size: "md" },
  },
);

// Applied after the colour classes so the radius beats gradient's rounded-pill.
const pressedSvg =
  "aria-pressed:[&>svg]:-rotate-12 aria-pressed:[&>svg]:scale-106";
const SHAPES = {
  circle: cn("rounded-pill", pressedSvg),
  square: cn("rounded-md", pressedSvg),
  diamond: "rotate-45 rounded-lg [&>svg]:-rotate-45",
};

type IconButtonProps = Omit<React.ComponentProps<"button">, "children"> &
  VariantProps<typeof buttonColorVariants> &
  VariantProps<typeof iconButtonVariants> & {
    icon: IconProp;
    /** circle (default), square (radius-md), or diamond (the glowing + button). */
    shape?: keyof typeof SHAPES;
    /** Required accessible name; also the default tooltip. */
    label: string;
    /** Tooltip text; false hides it. Defaults to `label`. */
    tooltip?: React.ReactNode | false;
    tooltipPlacement?: TooltipProps["placement"];
    shortcut?: string;
    glow?: boolean;
    /** Toggle state: sets aria-pressed and the selected ring. */
    pressed?: boolean;
    /** true = dot; a number or short text = count. */
    badge?: boolean | number | string;
  };

function IconButton({
  icon,
  label,
  variant = "glass",
  size = "md",
  shape = "circle",
  tooltip,
  tooltipPlacement,
  shortcut,
  glow,
  pressed,
  badge,
  className,
  ...props
}: IconButtonProps) {
  const btn = (
    <button
      type="button"
      data-slot="icon-button"
      data-variant={variant}
      aria-label={label}
      aria-pressed={pressed}
      {...props}
      className={cn(
        iconButtonVariants({ size }),
        buttonColorVariants({ variant }),
        SHAPES[shape],
        size === "sm" && shape === "square" && "rounded-sm",
        variant === "tinted" && "text-fg-primary",
        variant === "glass" &&
          "aria-pressed:border-button-text aria-pressed:text-button-text",
        glow && glowClass,
        className,
      )}
    >
      {renderIcon(icon, ICON_PX[size ?? "md"])}
      {badge != null && badge !== false && (
        <span
          aria-hidden="true"
          className={cn(
            "absolute box-border rounded-pill border-2 border-page bg-button px-[3px] text-[10px] leading-[14px] font-semibold text-button-fg",
            badge === true
              ? "-top-0.5 -right-0.5 h-2.5 min-w-2.5"
              : "-top-1.5 -right-1.5 inline-flex h-[18px] min-w-[18px] items-center justify-center",
          )}
        >
          {badge === true ? null : badge}
        </span>
      )}
    </button>
  );
  return tooltip === false ? (
    btn
  ) : (
    <Tooltip
      content={tooltip || label}
      placement={tooltipPlacement}
      shortcut={shortcut}
    >
      {btn}
    </Tooltip>
  );
}

export { IconButton, iconButtonVariants, type IconButtonProps };
