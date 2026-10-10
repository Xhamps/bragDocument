import * as React from "react";
import { cn, focusRing } from "#lib/utils";
import { useIndicator } from "#lib/use-indicator";
import { Icon, renderIcon, type IconProp } from "#components/icon";
import { IconButton } from "#components/icon-button";
import { SearchField, type SearchFieldProps } from "#components/search-field";
import { UserMenu, type UserMenuProps } from "#components/user-menu";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "#components/dropdown-menu";

type TopBarItem = {
  label: string;
  value?: string;
  href?: string;
  icon?: IconProp;
  description?: string;
  children?: TopBarItem[];
};

/** Props for one rendered link; spread them onto your router's link (map `href` to `to`). */
type TopBarLinkProps = {
  href: string;
  className?: string;
  "data-on"?: boolean;
  "aria-current"?: "page";
  onClick?: (e: React.MouseEvent<HTMLAnchorElement>) => void;
  children: React.ReactNode;
};

type TopBarProps = {
  brand?: { name: string; href?: string; logo?: React.ReactNode };
  items?: TopBarItem[];
  value?: string;
  defaultValue?: string;
  onSelect?: (value: string, item: TopBarItem) => void;
  /** true for a default SearchField, or SearchField props. */
  search?: boolean | SearchFieldProps;
  /** IconButtons (notifications, theme) before the user. */
  actions?: React.ReactNode;
  /** UserMenu props; the avatar variant unless `variant` is given. */
  user?: UserMenuProps;
  cta?: React.ReactNode;
  /** bar = full-width glass bar with a bottom hairline; floating = centred glass pill. */
  variant?: "bar" | "floating";
  /** Sticks to the top; on scroll the bar shrinks and lifts. */
  sticky?: boolean;
  label?: string;
  /**
   * Renders every link (brand, nav, menus). Defaults to a plain `<a>`.
   * With `onSelect` set, link clicks are preventDefault'ed, so the caller must navigate itself.
   */
  renderLink?: (item: TopBarItem, props: TopBarLinkProps) => React.ReactNode;
  className?: string;
};

const idOf = (it: TopBarItem) => it.value ?? it.label;
const containsId = (it: TopBarItem, v?: string): boolean =>
  (it.children ?? []).some((c) => idOf(c) === v || containsId(c, v));
const defaultLink: NonNullable<TopBarProps["renderLink"]> = (_, props) => (
  <a {...props} />
);

const linkClass = cn(
  "inline-flex h-[38px] cursor-pointer items-center gap-2 rounded-pill border-0 bg-transparent px-3.5 text-[15px] font-medium whitespace-nowrap text-fg-secondary no-underline transition-colors duration-fast ease-standard hover:text-fg-primary data-[on=true]:text-fg-primary",
  focusRing,
);

function TopBar({
  brand = { name: "Brand" },
  items = [],
  value: valueProp,
  defaultValue,
  onSelect,
  search,
  actions,
  user,
  cta,
  variant = "bar",
  sticky,
  label = "Main",
  renderLink = defaultLink,
  className,
}: TopBarProps) {
  const [inner, setInner] = React.useState(
    defaultValue ?? (items[0] && idOf(items[0])),
  );
  const value = valueProp ?? inner;
  const {
    ref: indRef,
    style: indStyle,
    animated,
  } = useIndicator<HTMLUListElement>([value, items.length]);

  const [scrolled, setScrolled] = React.useState(false);
  React.useEffect(() => {
    if (!sticky) return;
    const on = () => setScrolled(window.scrollY > 4);
    on();
    window.addEventListener("scroll", on, { passive: true });
    return () => window.removeEventListener("scroll", on);
  }, [sticky]);

  function select(it: TopBarItem) {
    const id = idOf(it);
    if (valueProp === undefined) setInner(id);
    onSelect?.(id, it);
  }

  // DS: with onSelect the bar handles navigation itself, so links don't follow.
  const onClickFor = (it: TopBarItem) => (e: React.MouseEvent) => {
    if (it.href === "#" || onSelect) e.preventDefault();
    select(it);
  };

  function body(it: TopBarItem) {
    return (
      <>
        {it.icon != null && (
          <span className="inline-flex" aria-hidden="true">
            {renderIcon(it.icon, 18)}
          </span>
        )}
        <span>{it.label}</span>
      </>
    );
  }

  function navItem(it: TopBarItem) {
    const id = idOf(it);
    const on = id === value || containsId(it, value);
    if (it.children) {
      return (
        <li key={id} className="relative z-1">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                data-on={on}
                className={cn(linkClass, "group")}
              >
                {body(it)}
                <Icon
                  name="chevronDown"
                  size={14}
                  className="-ml-0.5 text-fg-tertiary transition-transform duration-base ease-standard group-data-[state=open]:rotate-180"
                />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="center"
              sideOffset={14}
              className="w-auto min-w-[280px] bg-surface shadow-xl"
            >
              {it.children.map((c) => (
                <DropdownMenuChild
                  key={idOf(c)}
                  item={c}
                  on={idOf(c) === value}
                  onClick={onClickFor(c)}
                  renderLink={renderLink}
                />
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </li>
      );
    }
    const props = {
      className: linkClass,
      "data-on": on,
      "aria-current": on ? ("page" as const) : undefined,
      onClick: onClickFor(it),
    };
    return (
      <li key={id} className="relative z-1">
        {it.href ? (
          renderLink(it, { ...props, href: it.href, children: body(it) })
        ) : (
          <button type="button" {...props}>
            {body(it)}
          </button>
        )}
      </li>
    );
  }

  const brandBody = (
    <>
      {brand.logo ?? (
        <span
          aria-hidden="true"
          className="size-[26px] rounded-sm bg-[linear-gradient(135deg,var(--button-text),var(--chart-2)_55%,var(--chart-3))] shadow-sm"
        />
      )}
      <span>{brand.name}</span>
    </>
  );
  const brandClass = cn(
    "inline-flex items-center gap-2 rounded-sm text-[17px] leading-6 font-semibold tracking-[-0.01em] text-fg-primary no-underline",
    focusRing,
    "focus-visible:outline-offset-4",
  );

  const floating = variant === "floating";
  return (
    <header
      data-slot="top-bar"
      className={cn(
        "relative z-40",
        sticky && "sticky top-0",
        floating && "px-4 pt-4",
        className,
      )}
    >
      <div
        className={cn(
          "box-border flex items-center gap-4 glass px-4 transition-[box-shadow,height,background-color] duration-base ease-standard lg:gap-6",
          floating
            ? cn(
                "mx-auto h-[60px] max-w-[960px] rounded-pill pr-2 pl-4 shadow-lg md:pl-5",
                scrolled && "shadow-xl",
              )
            : cn(
                "rounded-none border-x-0 border-t-0 md:px-6",
                scrolled ? "h-[60px] shadow-lg" : "h-[72px]",
              ),
        )}
      >
        <h1 className="m-0 flex-none">
          {brand.href ? (
            renderLink(
              { label: brand.name, href: brand.href },
              { href: brand.href, className: brandClass, children: brandBody },
            )
          ) : (
            <span className={brandClass}>{brandBody}</span>
          )}
        </h1>
        <nav
          aria-label={label}
          className="flex min-w-0 flex-1 justify-center max-md:hidden"
        >
          <ul
            ref={indRef}
            className="relative isolate m-0 flex list-none items-center gap-0.5 p-0"
          >
            <li
              aria-hidden="true"
              role="presentation"
              style={indStyle}
              className={cn(
                "pointer-events-none absolute top-0 left-0 z-0 box-border list-none rounded-pill border border-container-border bg-container shadow-md",
                animated &&
                  "[transition:transform_var(--duration-slow)_var(--ease-spring),width_var(--duration-slow)_var(--ease-spring),height_var(--duration-slow)_var(--ease-spring),opacity_var(--duration-fast)_var(--ease-standard)]",
              )}
            />
            {items.map(navItem)}
          </ul>
        </nav>
        <div className="ml-auto flex flex-none items-center gap-2">
          {search && (
            <SearchField
              placeholder="Search"
              {...(search === true ? {} : search)}
              className={cn(
                "h-10 w-40 min-w-0 max-md:hidden lg:w-[200px]",
                search !== true && search.className,
              )}
            />
          )}
          {actions}
          {user && <UserMenu variant="avatar" {...user} />}
          {cta}
          {items.length > 0 && (
            <MobileMenu
              items={items}
              value={value}
              onClickFor={onClickFor}
              renderLink={renderLink}
            />
          )}
        </div>
      </div>
    </header>
  );
}

function DropdownMenuChild({
  item,
  on,
  onClick,
  renderLink,
}: {
  item: TopBarItem;
  on: boolean;
  onClick: (e: React.MouseEvent) => void;
  renderLink: NonNullable<TopBarProps["renderLink"]>;
}) {
  const content = (
    <>
      {item.icon != null && (
        <span
          aria-hidden="true"
          className="inline-flex size-9 flex-none items-center justify-center rounded-md border border-container-border bg-container text-button-text shadow-sm"
        >
          {renderIcon(item.icon)}
        </span>
      )}
      <span className="flex flex-col">
        <span
          className={cn(
            "text-sm leading-5 font-medium text-fg-primary",
            on && "text-button-text",
          )}
        >
          {item.label}
        </span>
        {item.description && (
          <span className="type-footnote text-fg-secondary">
            {item.description}
          </span>
        )}
      </span>
    </>
  );
  const cls = "items-start p-3";
  return item.href ? (
    <DropdownMenuItem asChild className={cls}>
      {renderLink(item, {
        href: item.href,
        "aria-current": on ? "page" : undefined,
        onClick,
        children: content,
      })}
    </DropdownMenuItem>
  ) : (
    <DropdownMenuItem
      className={cls}
      aria-current={on ? "page" : undefined}
      onClick={onClick}
    >
      {content}
    </DropdownMenuItem>
  );
}

// Below md the nav collapses into one menu listing every link (children indented).
function MobileMenu({
  items,
  value,
  onClickFor,
  renderLink,
}: {
  items: TopBarItem[];
  value?: string;
  onClickFor: (it: TopBarItem) => (e: React.MouseEvent) => void;
  renderLink: NonNullable<TopBarProps["renderLink"]>;
}) {
  const [open, setOpen] = React.useState(false);
  const row = (it: TopBarItem, sub = false) => {
    const on = idOf(it) === value;
    const cls = cn(
      "min-h-11 no-underline",
      sub && "min-h-[38px] pl-8 text-sm",
      on && "bg-container text-fg-primary",
    );
    return it.href ? (
      <DropdownMenuItem key={idOf(it)} asChild className={cls}>
        {renderLink(it, {
          href: it.href,
          "aria-current": on ? "page" : undefined,
          onClick: onClickFor(it),
          children: it.label,
        })}
      </DropdownMenuItem>
    ) : (
      <DropdownMenuItem
        key={idOf(it)}
        className={cls}
        aria-current={on ? "page" : undefined}
        onClick={onClickFor(it)}
      >
        {it.label}
      </DropdownMenuItem>
    );
  };
  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <IconButton
          icon={open ? "close" : "menu"}
          label={open ? "Close menu" : "Open menu"}
          tooltip={false}
          variant="tinted"
          className="md:hidden"
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        sideOffset={8}
        className="w-[calc(100vw-32px)] bg-surface shadow-xl"
      >
        {items.map((it) =>
          it.children ? (
            <React.Fragment key={idOf(it)}>
              <DropdownMenuLabel>{it.label}</DropdownMenuLabel>
              {it.children.map((c) => row(c, true))}
            </React.Fragment>
          ) : (
            row(it)
          ),
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export { TopBar, type TopBarProps, type TopBarItem, type TopBarLinkProps };
