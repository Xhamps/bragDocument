import * as React from "react";
import { Popover } from "radix-ui";
import { cn, focusRing } from "#lib/utils";
import { useIndicator } from "#lib/use-indicator";
import { Icon, renderIcon, type IconProp } from "#components/icon";

type MenuItem = {
  label: string;
  value?: string;
  /** A node, or an icon name ("grid", "book", "settings"…). */
  icon?: IconProp;
  trailing?: React.ReactNode;
  /** Count pill on the right. */
  badge?: React.ReactNode;
  /** Renders the row as a link; selecting still moves the highlight. */
  href?: string;
  /** Opens in a new tab and shows an external-link icon. */
  external?: boolean;
  /** Nested rows: the row becomes a group (flyout or accordion). */
  children?: readonly MenuEntry[];
  /** Start this inline group expanded (groups holding the value open automatically). */
  defaultOpen?: boolean;
  onSelect?: (value: string) => void;
};

/** A row, "separator", or a plain string for an uppercase section heading. */
type MenuEntry = MenuItem | "separator" | string;

type MenuListProps = {
  items: readonly MenuEntry[];
  value?: string;
  defaultValue?: string;
  onSelect?: (value: string, item: MenuItem) => void;
  /** false drops the glass container (inside a sidebar that is already glass). */
  glass?: boolean;
  /** flyout = a panel beside the row (default); inline = an accordion. */
  submenu?: "flyout" | "inline";
  label?: string;
  className?: string;
};

const idOf = (it: MenuItem) => it.value ?? it.label;
const isItem = (it: MenuEntry): it is MenuItem => typeof it !== "string";
function containsId(it: MenuItem, v: string | undefined): boolean {
  return (it.children ?? [])
    .filter(isItem)
    .some((c) => idOf(c) === v || containsId(c, v));
}

const rowClass = cn(
  "relative box-border flex h-11 w-full cursor-pointer items-center gap-3 rounded-md border border-transparent bg-transparent px-4 text-left text-[15px] font-medium text-fg-secondary no-underline",
  "transition-[background-color,color,border-color,padding-left] duration-base ease-standard hover:text-fg-primary",
  "hover:not-[[aria-current]]:pl-[18px] [&[aria-current]]:text-fg-primary",
  focusRing,
);
// .bd-menu-sub-item (inline children) and .bd-menu-fly-item (flyout rows).
const subRowClass =
  "h-9 pl-3 text-sm font-normal hover:not-[[aria-current]]:pl-3.5 [&[aria-current]]:font-medium";
const flyRowClass =
  "h-[38px] pl-3 whitespace-nowrap hover:bg-container hover:not-[[aria-current]]:pl-3 focus-visible:bg-container [&[aria-current]]:bg-container [&[aria-current]]:text-button-text [&[aria-current]]:before:absolute [&[aria-current]]:before:inset-y-2.5 [&[aria-current]]:before:left-0 [&[aria-current]]:before:w-0.5 [&[aria-current]]:before:rounded-[2px] [&[aria-current]]:before:bg-button-text";
// ponytail: basic tabbable query; swap for a tabbable lib if menus hold odd controls.
const TABBABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';
const caretClass =
  "inline-flex text-fg-tertiary transition-transform duration-base ease-standard";

function MenuList({
  items,
  value: valueProp,
  defaultValue,
  onSelect,
  glass = true,
  submenu = "flyout",
  label,
  className,
}: MenuListProps) {
  const [inner, setInner] = React.useState(defaultValue);
  const value = valueProp ?? inner;
  const fly = submenu !== "inline";

  const [open, setOpen] = React.useState<Record<string, boolean>>(() => {
    const init: Record<string, boolean> = {};
    if (fly) return init;
    (function walk(list: readonly MenuEntry[]) {
      for (const it of list.filter(isItem)) {
        if (!it.children) continue;
        if (it.defaultOpen || containsId(it, value)) init[idOf(it)] = true;
        walk(it.children);
      }
    })(items);
    return init;
  });

  // Flyout: one panel open at a time; hover closes 160ms after leaving.
  const [flyOpen, setFly] = React.useState<string | null>(null);
  const timer = React.useRef<ReturnType<typeof setTimeout>>(undefined);
  const byKey = React.useRef(false); // ArrowRight opened it: focus the first row
  const restore = React.useRef(false); // keyboard close or pick: refocus the row
  React.useEffect(() => () => clearTimeout(timer.current), []);
  const hoverOpen = (id: string) => {
    clearTimeout(timer.current);
    setFly(id);
  };
  const hoverClose = () => {
    clearTimeout(timer.current);
    timer.current = setTimeout(() => setFly(null), 160);
  };

  const {
    ref: indRef,
    style: indStyle,
    animated,
  } = useIndicator<HTMLUListElement>([
    value,
    items.length,
    JSON.stringify(open),
  ]);

  function select(it: MenuItem) {
    const id = idOf(it);
    if (valueProp === undefined) setInner(id);
    onSelect?.(id, it);
    it.onSelect?.(id);
  }

  function rowInner(it: MenuItem, extra?: React.ReactNode) {
    return (
      <>
        {it.icon != null && (
          <span className="size-5 flex-none" aria-hidden="true">
            {renderIcon(it.icon)}
          </span>
        )}
        <span className="flex-1">{it.label}</span>
        {it.badge != null && (
          <span className="box-border h-5 min-w-5 rounded-pill bg-button px-1.5 text-center text-xs leading-5 font-semibold text-button-fg">
            {it.badge}
          </span>
        )}
        {extra ?? it.trailing}
      </>
    );
  }

  function renderList(
    list: readonly MenuEntry[],
    depth: number,
    inFly = false,
  ) {
    return list.map((it, i) => {
      if (it === "separator")
        return (
          <li
            key={`sep${i}`}
            role="separator"
            className="relative z-1 mx-3 my-2 h-px bg-divider"
          />
        );
      if (typeof it === "string")
        return (
          <li
            key={`h${i}`}
            role="presentation"
            className="relative z-1 px-4 pt-3 pb-1 type-caption text-fg-tertiary"
          >
            {it}
          </li>
        );

      const id = idOf(it);
      const on = id === value;
      const cls = cn(rowClass, depth > 0 && subRowClass);

      if (it.children) {
        const childOn = containsId(it, value);
        const groupCls = cn(cls, childOn && "text-fg-primary");

        if (fly) {
          const fOpen = flyOpen === id;
          return (
            <li
              key={id}
              className={cn("relative z-1", fOpen && "z-2")}
              onMouseEnter={() => hoverOpen(id)}
              onMouseLeave={hoverClose}
            >
              <Popover.Root
                open={fOpen}
                onOpenChange={(o) => setFly(o ? id : null)}
              >
                <Popover.Trigger asChild>
                  <button
                    type="button"
                    aria-haspopup="true"
                    data-on={childOn}
                    className={cn(groupCls, fOpen && "text-fg-primary")}
                    onKeyDown={(e) => {
                      if (e.key === "ArrowRight" && !fOpen) {
                        e.preventDefault();
                        byKey.current = true;
                        setFly(id);
                      }
                    }}
                  >
                    {rowInner(
                      it,
                      <span
                        aria-hidden="true"
                        className={cn(
                          caretClass,
                          fOpen && "translate-x-0.5 text-fg-primary",
                        )}
                      >
                        <Icon name="chevron" size={16} />
                      </span>,
                    )}
                  </button>
                </Popover.Trigger>
                <Popover.Content
                  role="group"
                  aria-label={it.label}
                  side="right"
                  align="start"
                  sideOffset={14}
                  alignOffset={-8}
                  onOpenAutoFocus={(e) => {
                    if (!byKey.current) e.preventDefault();
                    byKey.current = false;
                  }}
                  onCloseAutoFocus={(e) => {
                    if (!restore.current) e.preventDefault();
                    restore.current = false;
                  }}
                  onEscapeKeyDown={() => (restore.current = true)}
                  // Radix loops Tab inside the panel; leave it in DOM order
                  // instead, as the DS does (Shift+Tab back to the row).
                  onKeyDownCapture={(e) => {
                    if (e.key !== "Tab") return;
                    e.preventDefault();
                    e.stopPropagation();
                    const panel = e.currentTarget;
                    const row = panel.closest("li")?.querySelector("button");
                    restore.current = e.shiftKey;
                    setFly(null);
                    if (e.shiftKey || !row) return;
                    const all = Array.from(
                      document.querySelectorAll<HTMLElement>(TABBABLE),
                    ).filter(
                      (el) => !panel.contains(el) && !el.closest("[inert]"),
                    );
                    (all[all.indexOf(row) + 1] ?? row).focus();
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "ArrowLeft") {
                      e.preventDefault();
                      restore.current = true;
                      setFly(null);
                    }
                    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
                      e.preventDefault();
                      const els = Array.from(
                        e.currentTarget.querySelectorAll<HTMLElement>(
                          "[data-menu-row]",
                        ),
                      );
                      const k = els.indexOf(
                        document.activeElement as HTMLElement,
                      );
                      const step = e.key === "ArrowDown" ? 1 : -1;
                      els[(k + step + els.length) % els.length]?.focus();
                    }
                  }}
                  className={cn(
                    "z-50 min-w-[200px] rounded-lg border border-container-border bg-surface p-2 shadow-xl backdrop-blur-(--blur-heavy)",
                    "origin-(--radix-popover-content-transform-origin) data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95 data-[state=open]:slide-in-from-left-[6px]",
                  )}
                >
                  <div className="px-3 pt-1 pb-2 type-caption text-fg-tertiary">
                    {it.label}
                  </div>
                  <ul role="list" className="m-0 flex flex-col gap-0.5 p-0">
                    {renderList(it.children, depth + 1, true)}
                  </ul>
                  {/* Open path: the 1px border strokes the two slanted edges, not the base. */}
                  <Popover.Arrow asChild width={12} height={6}>
                    <svg>
                      <path
                        d="M0 0 L15 10 L30 0"
                        vectorEffect="non-scaling-stroke"
                        className="fill-surface stroke-container-border"
                      />
                    </svg>
                  </Popover.Arrow>
                </Popover.Content>
              </Popover.Root>
            </li>
          );
        }

        const isOpen = !!open[id];
        return (
          <li key={id} className="relative z-1">
            <button
              type="button"
              aria-expanded={isOpen}
              data-on={childOn && !isOpen}
              className={groupCls}
              onClick={() => setOpen((o) => ({ ...o, [id]: !o[id] }))}
            >
              {rowInner(
                it,
                <span
                  aria-hidden="true"
                  className={cn(caretClass, isOpen && "rotate-180")}
                >
                  <Icon name="chevronDown" size={16} />
                </span>,
              )}
            </button>
            {/* ponytail: no per-row stagger on expand; the grid-rows slide carries it. */}
            <div
              inert={!isOpen}
              className={cn(
                "grid transition-[grid-template-rows] duration-slow ease-standard",
                isOpen ? "grid-rows-[1fr]" : "grid-rows-[0fr]",
              )}
            >
              <ul
                role="list"
                className="relative m-0 flex min-h-0 flex-col gap-0.5 overflow-hidden p-0 pl-6 before:absolute before:inset-y-1 before:left-[22px] before:w-px before:bg-divider"
              >
                {renderList(it.children, depth + 1)}
              </ul>
            </div>
          </li>
        );
      }

      const props = {
        "data-menu-row": "",
        "data-on": on && !inFly,
        "aria-current": on ? (it.href ? ("page" as const) : true) : undefined,
        className: cn(cls, inFly && flyRowClass),
        onClick: (e: React.MouseEvent) => {
          if (inFly) {
            restore.current = true;
            setFly(null);
          }
          // A plain link with no handlers just navigates; the highlight follows.
          if (
            it.href &&
            it.href !== "#" &&
            !onSelect &&
            !it.onSelect &&
            valueProp === undefined
          ) {
            setInner(id);
            return;
          }
          if (it.href) e.preventDefault();
          select(it);
        },
      };
      return (
        <li key={id} className="relative z-1">
          {it.href ? (
            <a
              href={it.href}
              target={it.external ? "_blank" : undefined}
              rel={it.external ? "noopener noreferrer" : undefined}
              {...props}
            >
              {rowInner(
                it,
                it.external ? (
                  <Icon
                    name="external"
                    size={16}
                    className="text-fg-tertiary"
                  />
                ) : undefined,
              )}
            </a>
          ) : (
            <button type="button" {...props}>
              {rowInner(it)}
            </button>
          )}
        </li>
      );
    });
  }

  return (
    <ul
      ref={indRef}
      aria-label={label}
      data-slot="menu-list"
      className={cn(
        "relative isolate m-0 flex list-none flex-col gap-0.5 rounded-lg p-2",
        flyOpen != null && "z-30",
        glass && "glass shadow-glass",
        className,
      )}
    >
      <li
        aria-hidden="true"
        role="presentation"
        style={indStyle}
        className={cn(
          "pointer-events-none absolute top-0 left-0 z-0 box-border list-none rounded-md border border-container-border bg-container shadow-sm will-change-[transform,height]",
          animated &&
            "[transition:transform_var(--duration-slow)_var(--ease-standard),height_var(--duration-slow)_var(--ease-standard),width_var(--duration-slow)_var(--ease-standard),opacity_var(--duration-fast)_var(--ease-standard)]",
        )}
      >
        <span className="absolute inset-y-2 -left-px w-0.5 rounded-[2px] bg-button-text" />
      </li>
      {renderList(items, 0)}
    </ul>
  );
}

export { MenuList, type MenuListProps, type MenuItem, type MenuEntry };
