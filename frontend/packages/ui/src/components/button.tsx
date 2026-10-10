import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { Slot } from "radix-ui";
import { cn, focusRing } from "#lib/utils";
import { Icon, renderIcon, type IconProp } from "#components/icon";

// Colour per variant only; shared with IconButton. Applied after the base so
// its border colours win over `border-transparent` in cn.
const buttonColorVariants = cva("", {
  variants: {
    variant: {
      glass:
        "glass border-container-border text-fg-primary shadow-button hover:border-fg-tertiary hover:shadow-md active:inset-shadow-ds disabled:border-button-inactive disabled:bg-transparent disabled:text-button-inactive",
      primary:
        "bg-button font-semibold text-button-fg hover:bg-button-hover hover:shadow-md active:bg-button-hover disabled:bg-button-inactive",
      ghost:
        "border-button-text bg-transparent text-button-text hover:border-button-hover hover:text-button-hover hover:shadow-md disabled:border-button-inactive disabled:text-button-inactive dark:hover:border-fg-primary dark:hover:text-fg-primary",
      tinted:
        "bg-transparent text-button-text hover:bg-container disabled:text-button-inactive",
      gradient:
        "bg-(image:--gradient-red-3) font-semibold text-button-fg shadow-cta hover:brightness-105 hover:saturate-115 disabled:bg-button-inactive disabled:bg-none",
      // Extension: the DS has a danger token but no destructive button.
      danger:
        "border-danger bg-transparent text-danger hover:bg-glass-tint-rose disabled:border-button-inactive disabled:text-button-inactive",
    },
  },
  defaultVariants: { variant: "glass" },
});

const buttonVariants = cva(
  [
    "group/button inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 rounded-md border border-transparent whitespace-nowrap select-none type-callout",
    "transition-[background-color,border-color,color,box-shadow,translate,scale,filter] duration-base ease-standard",
    "hover:-translate-y-px active:translate-y-0 active:scale-[0.98]",
    "disabled:pointer-events-none disabled:cursor-default disabled:shadow-none",
    "[&_svg]:size-4 [&_svg]:shrink-0",
    focusRing,
  ],
  {
    variants: {
      size: {
        sm: "h-7 rounded-sm px-2 type-footnote font-medium",
        md: "h-9 px-3",
        lg: "h-11 px-4 type-body font-semibold",
      },
      pill: { true: "rounded-pill" },
      fullWidth: { true: "flex w-full" },
    },
    defaultVariants: { size: "md" },
  },
);

// After the colour classes so it beats their own shadow.
const glowClass = "shadow-glow hover:shadow-glow-strong";

type ButtonProps = Omit<React.ComponentProps<"button">, "children"> &
  VariantProps<typeof buttonVariants> &
  VariantProps<typeof buttonColorVariants> & {
    asChild?: boolean;
    /** Adds `shadow-glow`. At most one per view. */
    glow?: boolean;
    chevron?: boolean;
    icon?: IconProp;
    trailingIcon?: IconProp;
    children?: React.ReactNode;
  };

function Button({
  className,
  variant = "glass",
  size = "md",
  pill,
  fullWidth,
  glow,
  asChild = false,
  chevron,
  icon,
  trailingIcon,
  children,
  ...props
}: ButtonProps) {
  const Comp = asChild ? Slot.Root : "button";
  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      {...(asChild ? {} : { type: "button" as const })}
      className={cn(
        buttonVariants({ size, pill, fullWidth }),
        buttonColorVariants({ variant }),
        // DS: the gradient CTA is always pill-shaped (Button only, not IconButton).
        variant === "gradient" && "rounded-pill",
        glow && glowClass,
        className,
      )}
      {...props}
    >
      {icon != null && renderIcon(icon, 16)}
      <Slot.Slottable>{children}</Slot.Slottable>
      {trailingIcon != null && renderIcon(trailingIcon, 16)}
      {chevron && (
        <Icon
          name="chevron"
          size={16}
          className="transition-transform duration-base group-hover/button:translate-x-0.5"
        />
      )}
    </Comp>
  );
}

export {
  Button,
  buttonVariants,
  buttonColorVariants,
  glowClass,
  type ButtonProps,
};
