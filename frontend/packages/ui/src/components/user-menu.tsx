import type * as React from "react";
import { cn, focusRing } from "#lib/utils";
import { Icon, renderIcon, type IconProp } from "#components/icon";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "#components/dropdown-menu";

type UserMenuItem = {
  label: string;
  icon?: IconProp;
  href?: string;
  onSelect?: () => void;
  tone?: "danger";
  shortcut?: string;
};

/** Props for one rendered menu link; spread them onto your router's link (map `href` to `to`). */
type UserMenuLinkProps = {
  href: string;
  className: string;
  children: React.ReactNode;
};

type UserMenuProps = {
  user: {
    name: string;
    email?: string;
    role?: string;
    avatar?: string;
    status?: "online" | "busy" | "away";
  };
  /** Defaults to Account, Billing, separator, Sign out. */
  items?: Array<UserMenuItem | "separator">;
  onSelect?: (label: string, item: UserMenuItem) => void;
  /** Renders items with an `href`. Defaults to a plain `<a>`. */
  renderLink?: (
    item: UserMenuItem,
    props: UserMenuLinkProps,
  ) => React.ReactNode;
  /** pill = glass button with avatar, name, role and caret (default); avatar = the avatar alone. */
  variant?: "pill" | "avatar";
  placement?: "bottom-end" | "bottom-start" | "top-start" | "top-end";
  defaultOpen?: boolean;
  className?: string;
};

const DEFAULT_ITEMS: Array<UserMenuItem | "separator"> = [
  { label: "Account", icon: "user" },
  { label: "Billing", icon: "card" },
  "separator",
  { label: "Sign out", icon: "signOut", tone: "danger" },
];

const initials = (n: string) =>
  (n || "?")
    .split(/\s+/)
    .map((w) => w[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();

function Avatar({ user }: Pick<UserMenuProps, "user">) {
  const cls =
    "box-border inline-flex size-10 flex-none items-center justify-center rounded-pill border border-container-border bg-chart-muted object-cover text-sm font-semibold text-fg-primary shadow-sm";
  return user.avatar ? (
    <img className={cls} src={user.avatar} alt="" />
  ) : (
    <span className={cls} aria-hidden="true">
      {initials(user.name)}
    </span>
  );
}

function NameBlock({ name, sub }: { name: string; sub?: string }) {
  return (
    <span className="flex min-w-0 flex-col">
      <span className="text-sm leading-5 font-medium whitespace-nowrap text-fg-primary">
        {name}
      </span>
      {sub && (
        <span className="truncate text-[13px] leading-[18px] text-fg-secondary">
          {sub}
        </span>
      )}
    </span>
  );
}

const STATUS = {
  online: "bg-success",
  busy: "bg-danger",
  away: "bg-fg-tertiary",
};

function UserMenu({
  user,
  items = DEFAULT_ITEMS,
  onSelect,
  renderLink = (_, props) => <a {...props} />,
  variant = "pill",
  placement = "bottom-end",
  defaultOpen,
  className,
}: UserMenuProps) {
  const compact = variant === "avatar";
  const [side, align] = placement.split("-") as [
    "top" | "bottom",
    "start" | "end",
  ];

  return (
    <DropdownMenu defaultOpen={defaultOpen}>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          data-slot="user-menu"
          aria-label={compact ? `${user.name}, account menu` : undefined}
          className={cn(
            "group inline-flex cursor-pointer items-center gap-3 rounded-pill text-left text-fg-primary",
            "transition-[box-shadow,translate,scale,border-color] duration-base ease-standard",
            compact
              ? "border-2 border-transparent bg-transparent p-0.5 data-[state=open]:border-button-text"
              : "h-[52px] glass pr-3 pl-1.5 shadow-glass hover:-translate-y-px hover:shadow-md active:scale-[0.98] motion-reduce:hover:translate-y-0",
            focusRing,
            className,
          )}
        >
          <span className="relative inline-flex">
            <Avatar user={user} />
            {user.status && (
              <span
                aria-hidden="true"
                className={cn(
                  "absolute -right-px -bottom-px box-border size-3 rounded-pill border-2 border-page",
                  STATUS[user.status],
                )}
              />
            )}
          </span>
          {!compact && (
            <>
              <NameBlock name={user.name} sub={user.role ?? user.email} />
              <Icon
                name="chevronDown"
                size={16}
                className="text-fg-secondary transition-transform duration-base ease-standard group-data-[state=open]:rotate-180"
              />
            </>
          )}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        aria-label="Account"
        side={side}
        align={align}
        sideOffset={8}
        className="w-auto min-w-60 shadow-xl"
      >
        <div className="flex items-center gap-3 px-3 py-2">
          <Avatar user={user} />
          <NameBlock name={user.name} sub={user.email} />
        </div>
        <DropdownMenuSeparator className="mx-2 my-1" />
        {items.map((it, i) =>
          it === "separator" ? (
            <DropdownMenuSeparator key={`s${i}`} className="mx-2 my-1" />
          ) : (
            <DropdownMenuItem
              key={it.label}
              variant={it.tone === "danger" ? "danger" : "default"}
              asChild={!!it.href}
              onSelect={() => {
                it.onSelect?.();
                onSelect?.(it.label, it);
              }}
            >
              {it.href ? (
                renderLink(it, {
                  href: it.href,
                  className: "no-underline",
                  children: <ItemBody item={it} />,
                })
              ) : (
                <ItemBody item={it} />
              )}
            </DropdownMenuItem>
          ),
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function ItemBody({ item }: { item: UserMenuItem }) {
  return (
    <>
      {item.icon != null && (
        <span className="size-5 flex-none" aria-hidden="true">
          {renderIcon(item.icon)}
        </span>
      )}
      <span className="flex-1">{item.label}</span>
      {item.shortcut && (
        <DropdownMenuShortcut>{item.shortcut}</DropdownMenuShortcut>
      )}
    </>
  );
}

export {
  UserMenu,
  type UserMenuItem,
  type UserMenuLinkProps,
  type UserMenuProps,
};
