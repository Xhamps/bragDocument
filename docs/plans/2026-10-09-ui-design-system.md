# UI Design System Adoption Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restyle `@bragdoc/ui` with the BragDoc glass design system in `ds/`, adopt the DS component APIs, port all 27 DS components, and migrate the app.

**Architecture:** DS tokens become raw CSS custom properties in `packages/ui/src/tokens.css`, exposed to Tailwind v4 through `@theme inline`. Components are TSX + cva re-implementations of `ds/components/bundle.js` + `bundle.css`, with Radix underneath for overlays and form primitives. Theme is OS-driven, overridable through `data-theme` on `<html>` and persisted in `localStorage`.

**Tech Stack:** React 19, Tailwind CSS v4, class-variance-authority, `cn` (clsx + tailwind-merge), radix-ui, lucide-react, Vitest + Testing Library.

Design: `docs/plans/2026-10-09-ui-design-system-design.md`. Branch: `feat/ui-design-system`.

---

## Ground rules (read first)

- **Sources of truth:** props from `ds/components/index.d.ts`; behaviour from `ds/components/bundle.js`; visuals from `ds/components/bundle.css`; usage notes from `ds/components/<Name>/README.md`; live reference `ds/preview/index.html` (open in a browser). For each component, read those four before writing it.
- **Translating CSS:** every `bd-*` rule becomes Tailwind utilities inside the component's cva/classes. Use the token utilities defined in Task 1 (`bg-container`, `text-fg-secondary`, `rounded-md`, `shadow-glass`, `duration-base`, `ease-standard` …). Arbitrary values only for one-offs (`h-[30px]`).
- **Type utilities are `type-*`, not `text-*`.** `cn` uses tailwind-merge, which treats any unknown `text-foo` as a colour and would drop it next to `text-fg-primary`. So the scale is `type-display`, `type-title-1` … `type-code`.
- **Glass:** `glass` utility = `container-bg` fill + 1px `container-border` + backdrop blur. It does **not** set a shadow; add `shadow-glass` (containers) or `shadow-button` (glass controls) explicitly.
- **Focus:** every interactive element gets `focusRing` (`outline-hidden focus-visible:outline-2 focus-visible:outline-solid …`; in Tailwind v4 `outline-none` + `outline-2` renders NO outline) (or `focus-within:` on the wrapper for TextField/Select). Export this string as `focusRing` from `#lib/utils`.
- **Icons:** string icon names resolve through `#components/icon` (Task 3). Never hand-draw SVGs.
- **`data-slot`:** keep the existing convention: every component root gets `data-slot="<kebab-name>"`.
- **Commands** (run from `frontend/`):
  - one UI test file: `npx vitest run -c packages/ui/vite.config.ts packages/ui/src/components/<file>.test.tsx` (or `npm test -w @bragdoc/ui -- <file>`)
  - full: `npm run typecheck && npm test && npm run lint && npm run fmt:check`
- **Commits:** one per task, message ends with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- During Tasks 1–17 the app will not typecheck (it still imports the old APIs). That is expected; only run the UI package's tests and typecheck until Task 18. Do **not** push until Task 24.

---

### Task 0: Baseline

**Step 1:** `cd frontend && npm run typecheck && npm test && npm run lint && npm run fmt:check`
Expected: all green. If not, stop and report; do not start on a red baseline.

---

### Task 1: Tokens and Tailwind theme

**Files:**
- Create: `frontend/packages/ui/src/tokens.css`
- Modify (rewrite): `frontend/packages/ui/src/styles.css`
- Modify: `frontend/packages/ui/src/lib/utils.ts`

**Step 1: Copy tokens (without the type classes)**

```bash
cd /Users/xhamps/projects/golang/bragDocument
sed -n '1,135p' ds/tokens.css > frontend/packages/ui/src/tokens.css
{ echo '@media (prefers-color-scheme: dark) {'
  awk '/^\[data-theme="dark"\] \{/,/^\}/' ds/tokens.css | sed '1s/.*/  :root:not([data-theme="light"]) {/'
  echo '}'; } >> frontend/packages/ui/src/tokens.css
```

Check: `grep -c '{' frontend/packages/ui/src/tokens.css` → 5 blocks (light, dark, `:root` constants, media, inner). Line 1 comment says "generated from tokens.json"; add a second comment line: `/* Copied from ds/tokens.css; re-copy with the commands in docs/plans/2026-10-09-ui-design-system.md Task 1. */`.

**Step 2: Rewrite `styles.css`**

Keep the existing `@custom-variant data-*` block unchanged (lines 3–55 today). Replace everything else with:

```css
@import "tailwindcss";
@import "tw-animate-css";
@import "./tokens.css";

/* …existing @custom-variant data-open … data-vertical blocks, unchanged… */

@custom-variant dark {
  &:where([data-theme="dark"], [data-theme="dark"] *) {
    @slot;
  }
  @media (prefers-color-scheme: dark) {
    &:where(:root:not([data-theme="light"]) *) {
      @slot;
    }
  }
}

@theme inline {
  --color-page: var(--page);
  --color-surface: var(--surface);
  --color-fg-primary: var(--fg-primary);
  --color-fg-secondary: var(--fg-secondary);
  --color-fg-tertiary: var(--fg-tertiary);
  --color-container: var(--container-bg);
  --color-container-border: var(--container-border);
  --color-divider: var(--container-divider);
  --color-glass-tint-blue: var(--glass-tint-blue);
  --color-glass-tint-violet: var(--glass-tint-violet);
  --color-glass-tint-rose: var(--glass-tint-rose);
  --color-glass-tint-teal: var(--glass-tint-teal);
  --color-button: var(--button);
  --color-button-hover: var(--button-hover);
  --color-button-fg: var(--button-fg);
  --color-button-text: var(--button-text);
  --color-button-inactive: var(--button-inactive);
  --color-focus-ring: var(--focus-ring);
  --color-success: var(--success);
  --color-danger: var(--danger);
  --color-chart-1: var(--chart-1);
  --color-chart-2: var(--chart-2);
  --color-chart-3: var(--chart-3);
  --color-chart-4: var(--chart-4);
  --color-chart-muted: var(--chart-muted);

  --radius-sm: var(--radius-sm);
  --radius-md: var(--radius-md);
  --radius-lg: var(--radius-lg);
  --radius-xl: var(--radius-xl);
  --radius-pill: var(--radius-pill);

  --shadow-sm: var(--shadow-sm);
  --shadow-md: var(--shadow-md);
  --shadow-lg: var(--shadow-lg);
  --shadow-xl: var(--shadow-xl);
  --shadow-glass: var(--shadow-glass);
  --shadow-button: var(--shadow-button);
  --shadow-glow: var(--shadow-glow);
  --shadow-glow-strong: var(--shadow-glow-strong);
  --shadow-cta: var(--shadow-cta);
  --inset-shadow-ds: var(--shadow-inset);

  --ease-standard: var(--ease-standard);
  --ease-out: var(--ease-out);
  --ease-spring: var(--ease-spring);

  --font-sans: var(--font-sans);
  --font-mono: var(--font-mono);
}

/* Durations as utilities (Tailwind has no duration namespace). */
@utility duration-fast { transition-duration: var(--duration-fast); }
@utility duration-base { transition-duration: var(--duration-base); }
@utility duration-slow { transition-duration: var(--duration-slow); }

/* Type scale — named type-* so tailwind-merge never mistakes it for a colour. */
@utility type-display { font-size: 56px; line-height: 60px; font-weight: 700; letter-spacing: -0.03em; }
@utility type-title-1 { font-size: 34px; line-height: 40px; font-weight: 600; letter-spacing: -0.02em; }
@utility type-title-2 { font-size: 24px; line-height: 30px; font-weight: 600; letter-spacing: -0.01em; }
@utility type-title-3 { font-size: 17px; line-height: 24px; font-weight: 500; }
@utility type-body { font-size: 15px; line-height: 22px; font-weight: 400; }
@utility type-callout { font-size: 14px; line-height: 20px; font-weight: 500; }
@utility type-footnote { font-size: 13px; line-height: 18px; font-weight: 400; }
@utility type-caption { font-size: 11px; line-height: 14px; font-weight: 500; letter-spacing: 0.06em; text-transform: uppercase; }
@utility type-code { font-family: var(--font-mono); font-size: 13px; line-height: 20px; font-weight: 400; }

@utility glass {
  background-color: var(--container-bg);
  border: 1px solid var(--container-border);
  -webkit-backdrop-filter: blur(var(--blur-glass));
  backdrop-filter: blur(var(--blur-glass));
}

/* Copy the three radial washes from `.bd-backdrop` in ds/components/bundle.css line 6–7 verbatim. */
@utility bd-backdrop {
  background-color: var(--page);
  /* background-image: <three radial-gradients from bundle.css .bd-backdrop>; */
  background-attachment: fixed;
}

@utility text-gradient { background: linear-gradient(135deg, var(--tg-primary-start), var(--tg-primary-end)); -webkit-background-clip: text; background-clip: text; color: transparent; }
@utility text-gradient-secondary { background: linear-gradient(135deg, var(--tg-secondary-start), var(--tg-secondary-mid), var(--tg-secondary-end)); -webkit-background-clip: text; background-clip: text; color: transparent; }
@utility text-gradient-accent { background: linear-gradient(90deg, var(--button-text), var(--chart-2) 55%, var(--chart-3)); -webkit-background-clip: text; background-clip: text; color: transparent; }

@keyframes bd-enter { from { opacity: 0; transform: translateY(12px) scale(0.98); } to { opacity: 1; transform: none; } }
@keyframes bd-pulse { 0%, 100% { box-shadow: 0 0 0 0 var(--glass-tint-blue); } 50% { box-shadow: 0 0 0 5px var(--glass-tint-blue); } }
@utility animate-enter { animation: bd-enter var(--duration-enter) var(--ease-out) both; animation-delay: var(--delay, 0ms); }
@utility animate-pulse-dot { animation: bd-pulse 2s var(--ease-standard) infinite; }

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}

@layer base {
  * { border-color: var(--container-divider); }
  html { font-family: var(--font-sans); }
  body { @apply min-h-screen bg-page text-fg-primary type-body antialiased; }
}
```

Note on `--radius-*`/`--shadow-*` mapping to the same name: `tokens.css` is unlayered, so its `:root` values beat Tailwind's layered theme vars; `inline` makes utilities read `var(--shadow-sm)` directly. Step 4 verifies this. `--inset-shadow-ds` gives `inset-shadow-ds` for pressed states.

**Step 3: Add `focusRing` to `lib/utils.ts`**

```ts
export { cn } from "cn";

export const focusRing =
  "outline-hidden focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-offset-2 focus-visible:outline-focus-ring";
```

**Step 4: Verify the CSS compiles and tokens resolve**

```bash
cd frontend && npx vite build packages/app --outDir /tmp/claude-ds-css --emptyOutDir 2>&1 | tail -3
grep -o '\.shadow-glass{[^}]*}' /tmp/claude-ds-css/assets/*.css | head -1
grep -o '\.rounded-md{[^}]*}' /tmp/claude-ds-css/assets/*.css | head -1
grep -c 'var(--radius-md)' /tmp/claude-ds-css/assets/*.css
```

The build itself may fail on app type errors — vite does not typecheck, so it should succeed. Expected: `.rounded-md{border-radius:var(--radius-md)}` and a `.shadow-glass` rule referencing `var(--shadow-glass)`. If Tailwind emits a self-referencing `--radius-md:var(--radius-md)` inside `@layer theme` and the computed value is empty in the browser, fall back to literal values in `@theme` for radii (`--radius-md: 10px` etc.) and to `@utility shadow-glass { box-shadow: var(--shadow-glass) }` per shadow.

**Step 5: Commit**

```bash
git add frontend/packages/ui/src/tokens.css frontend/packages/ui/src/styles.css frontend/packages/ui/src/lib/utils.ts
git commit -m "feat(ui): DS tokens, Tailwind theme and glass utilities"
```

---

### Task 2: Theme hook and pre-paint bootstrap

**Files:**
- Create: `frontend/packages/ui/src/lib/theme.ts`, `frontend/packages/ui/src/lib/theme.test.ts`
- Modify: `frontend/packages/app/index.html`

**Step 1: Failing test** `lib/theme.test.ts`

```ts
import { act, renderHook } from "@testing-library/react";
import { useTheme } from "#lib/theme";

beforeEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

test("leaves data-theme unset when nothing is stored", () => {
  renderHook(() => useTheme());
  expect(document.documentElement.dataset.theme).toBeUndefined();
});

test("toggle persists the choice and sets data-theme", () => {
  const { result } = renderHook(() => useTheme());
  act(() => result.current.toggle());
  expect(localStorage.getItem("theme")).toBe("dark");
  expect(document.documentElement.dataset.theme).toBe("dark");
  act(() => result.current.toggle());
  expect(document.documentElement.dataset.theme).toBe("light");
});

test("restores a stored theme", () => {
  localStorage.setItem("theme", "dark");
  const { result } = renderHook(() => useTheme());
  expect(result.current.theme).toBe("dark");
});
```

**Step 2:** run it → FAIL (module not found). The test file is `.ts`; check `vite.config.ts` include picks it up (default `**/*.test.ts` — yes).

**Step 3: Implement** `lib/theme.ts`

```ts
import { useEffect, useState } from "react";

export type Theme = "light" | "dark";
const KEY = "theme";

function stored(): Theme | null {
  try {
    const v = localStorage.getItem(KEY);
    return v === "light" || v === "dark" ? v : null;
  } catch {
    return null;
  }
}

function system(): Theme {
  return typeof matchMedia === "function" &&
    matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

// Unset data-theme means "follow the OS"; a stored choice overrides it.
export function useTheme() {
  const [choice, setChoice] = useState<Theme | null>(stored);
  const theme = choice ?? system();

  useEffect(() => {
    const root = document.documentElement;
    if (choice) root.dataset.theme = choice;
    else delete root.dataset.theme;
  }, [choice]);

  function setTheme(t: Theme) {
    try {
      localStorage.setItem(KEY, t);
    } catch {
      // ponytail: private mode — the choice lasts for this tab only.
    }
    setChoice(t);
  }

  return { theme, setTheme, toggle: () => setTheme(theme === "dark" ? "light" : "dark") };
}
```

**Step 4:** run test → PASS.

**Step 5: `index.html`** — replace the `<body class="bg-background text-foreground">` with `<body>` and add to `<head>`:

```html
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet" />
<script>
  try {
    var t = localStorage.getItem("theme");
    if (t === "light" || t === "dark") document.documentElement.dataset.theme = t;
  } catch (e) {}
</script>
```

**Step 6:** export from `packages/ui/src/index.ts`: `export { useTheme, type Theme } from "#lib/theme";`. Commit: `feat(ui): theme hook with OS default and stored override`.

---

### Task 3: Icon

**Files:** Create `frontend/packages/ui/src/components/icon.tsx`, `icon.test.tsx`

**Step 1: Failing test**

```tsx
import { render } from "@testing-library/react";
import { Icon, renderIcon } from "#components/icon";

test("renders a named icon at 20px with a 1.5 stroke", () => {
  const { container } = render(<Icon name="search" />);
  const svg = container.querySelector("svg")!;
  expect(svg).toHaveAttribute("width", "20");
  expect(svg).toHaveAttribute("stroke-width", "1.5");
  expect(svg).toHaveAttribute("aria-hidden", "true");
});

test("renderIcon passes nodes through and ignores unknown names", () => {
  const { container } = render(<>{renderIcon(<b>x</b>)}{renderIcon("nope")}</>);
  expect(container.innerHTML).toBe("<b>x</b>");
});
```

**Step 2:** run → FAIL.

**Step 3: Implement** — map every name in `ds/components/bundle.js` `ICONS` (lines 8–44) to lucide:

```tsx
import type * as React from "react";
import {
  ArrowLeft, ArrowRight, Bell, BookOpen, ChartColumn, Check, ChevronDown, ChevronRight,
  ChevronUp, ChevronsUpDown, CircleHelp, CirclePlus, CreditCard, Download, Ellipsis,
  ExternalLink, Eye, House, LayoutGrid, LogOut, Mail, Menu, Minus, Moon, Pencil, Play,
  Plus, Search, Settings, Share, SlidersHorizontal, Sun, Trash2, User, X,
  type LucideIcon,
} from "lucide-react";

const ICONS: Record<string, LucideIcon> = {
  chevron: ChevronRight, chevronDown: ChevronDown, search: Search, check: Check,
  plus: Plus, minus: Minus, mail: Mail, arrowLeft: ArrowLeft, arrowRight: ArrowRight,
  download: Download, card: CreditCard, plusCircle: CirclePlus, moon: Moon, sun: Sun,
  play: Play, home: House, bell: Bell, sliders: SlidersHorizontal, eye: Eye, share: Share,
  close: X, external: ExternalLink, menu: Menu, user: User, signOut: LogOut,
  settings: Settings, grid: LayoutGrid, book: BookOpen, chart: ChartColumn, help: CircleHelp,
  sort: ChevronsUpDown, sortUp: ChevronUp, sortDown: ChevronDown, more: Ellipsis,
  edit: Pencil, trash: Trash2,
};

export type IconProp = React.ReactNode | string;

function Icon({ name, size = 20, className }: { name: string; size?: number; className?: string }) {
  const C = ICONS[name];
  return C ? <C size={size} strokeWidth={1.5} aria-hidden="true" className={className} /> : null;
}

function renderIcon(icon: IconProp, size = 20) {
  return typeof icon === "string" ? <Icon name={icon} size={size} /> : icon;
}

export { Icon, renderIcon };
```

**Step 4:** PASS. **Step 5:** commit `feat(ui): Icon with DS names mapped to lucide`.

---

### Task 4: Button (DS API)

**Files:** Modify (rewrite) `components/button.tsx`, `components/button.test.tsx`

Reference: `bundle.js` `Button` (l.49–55); `bundle.css` "Button" (l.22–38) plus the hover/press motion rules (search `bd-btn:hover`, `bd-btn:active`, `bd-btn-gradient`, `bd-btn-tinted`, `bd-btn-pill`, `bd-btn-block`).

**Step 1: Failing test** (replace file)

```tsx
import { render, screen } from "@testing-library/react";
import { Button } from "#components/button";

test("defaults to a glass md button of type button", () => {
  render(<Button>Save</Button>);
  const b = screen.getByRole("button", { name: "Save" });
  expect(b).toHaveAttribute("type", "button");
  expect(b).toHaveAttribute("data-variant", "glass");
  expect(b).toHaveAttribute("data-size", "md");
});

test("chevron and named icons render svgs", () => {
  render(<Button icon="download" chevron>Export</Button>);
  expect(screen.getByRole("button").querySelectorAll("svg")).toHaveLength(2);
});

test("asChild renders the child element with icons around its text", () => {
  render(<Button asChild variant="primary" trailingIcon="arrowRight"><a href="/x">Go</a></Button>);
  const link = screen.getByRole("link", { name: "Go" });
  expect(link).toHaveAttribute("data-variant", "primary");
  expect(link.querySelector("svg")).not.toBeNull();
});

test("danger variant is supported", () => {
  render(<Button variant="danger">Delete</Button>);
  expect(screen.getByRole("button")).toHaveAttribute("data-variant", "danger");
});
```

**Step 2:** FAIL.

**Step 3: Implement**

```tsx
import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { Slot } from "radix-ui";
import { cn, focusRing } from "#lib/utils";
import { Icon, renderIcon, type IconProp } from "#components/icon";

const buttonVariants = cva(
  [
    "group/button inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 rounded-md border border-transparent whitespace-nowrap select-none type-callout",
    "transition-[background-color,border-color,color,box-shadow,translate,scale] duration-base ease-standard",
    "hover:-translate-y-px active:translate-y-0 active:scale-[0.98]",
    "disabled:pointer-events-none disabled:cursor-default disabled:shadow-none",
    "[&_svg]:size-4 [&_svg]:shrink-0",
    focusRing,
  ],
  {
    variants: {
      variant: {
        glass: "glass text-fg-primary shadow-button hover:border-fg-tertiary hover:shadow-md active:inset-shadow-ds disabled:border-button-inactive disabled:bg-transparent disabled:text-button-inactive",
        primary: "bg-button font-semibold text-button-fg hover:bg-button-hover hover:shadow-md active:bg-button-hover disabled:bg-button-inactive",
        ghost: "border-button-text bg-transparent text-button-text hover:border-button-hover hover:text-button-hover dark:hover:border-fg-primary dark:hover:text-fg-primary disabled:border-button-inactive disabled:text-button-inactive",
        tinted: "bg-transparent text-button-text hover:bg-glass-tint-blue disabled:text-button-inactive",
        gradient: "bg-(image:--gradient-red-3) font-semibold text-white shadow-cta hover:shadow-glow-strong disabled:opacity-50",
        // Extension: the DS has a danger token but no destructive button.
        danger: "border-danger bg-transparent text-danger hover:bg-glass-tint-rose disabled:border-button-inactive disabled:text-button-inactive",
      },
      size: {
        sm: "h-7 rounded-sm px-2 type-footnote font-medium",
        md: "h-9 px-3",
        lg: "h-11 px-4 type-body font-semibold",
      },
      pill: { true: "rounded-pill" },
      fullWidth: { true: "w-full" },
      glow: { true: "shadow-glow hover:shadow-glow-strong" },
    },
    defaultVariants: { variant: "glass", size: "md" },
  },
);

type ButtonProps = Omit<React.ComponentProps<"button">, "children"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
    chevron?: boolean;
    icon?: IconProp;
    trailingIcon?: IconProp;
    children?: React.ReactNode;
  };

function Button({
  className, variant = "glass", size = "md", pill, fullWidth, glow,
  asChild = false, chevron, icon, trailingIcon, children, ...props
}: ButtonProps) {
  const Comp = asChild ? Slot.Root : "button";
  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      {...(asChild ? {} : { type: "button" as const })}
      className={cn(buttonVariants({ variant, size, pill, fullWidth, glow }), className)}
      {...props}
    >
      {icon != null && renderIcon(icon, 16)}
      <Slot.Slottable>{children}</Slot.Slottable>
      {trailingIcon != null && renderIcon(trailingIcon, 16)}
      {chevron && <Icon name="chevron" size={16} className="transition-transform duration-base group-hover/button:translate-x-0.5" />}
    </Comp>
  );
}

export { Button, buttonVariants, type ButtonProps };
```

Note `type` comes before `{...props}` so callers can pass `type="submit"`.

**Step 4:** PASS. Also `npm run typecheck -w @bragdoc/ui` (ignore errors in files later tasks rewrite; button.tsx itself must be clean).

**Step 5:** commit `feat(ui): Button on the DS API (+danger, asChild)`.

---

### Task 5: Tooltip (Radix)

**Files:** Create `components/tooltip.tsx`, `tooltip.test.tsx`

Reference: `bundle.js` `Tooltip` (l.476–487), `bundle.css` `.bd-tip*`, `ds/components/Tooltip/README.md`.

Props (`TooltipProps`): `children` (one focusable trigger), `content`, `placement` (top default), `shortcut`, `open`.

**Step 1: Failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Tooltip } from "#components/tooltip";

test("shows content and shortcut on focus", async () => {
  render(<Tooltip content="Search" shortcut="⌘K"><button>s</button></Tooltip>);
  await userEvent.tab();
  expect(await screen.findByRole("tooltip")).toHaveTextContent("Search⌘K");
});

test("open forces it visible", () => {
  render(<Tooltip content="Hi" open><button>s</button></Tooltip>);
  expect(screen.getByRole("tooltip")).toHaveTextContent("Hi");
});
```

**Step 3: Implement** with `import { Tooltip as T } from "radix-ui"`: wrap in `<T.Provider delayDuration={300}>` inside the component (self-contained; no app-level provider needed), `<T.Root open={open}>`, `<T.Trigger asChild>{children}</T.Trigger>`, `<T.Portal><T.Content side={placement} sideOffset={8} className="glass shadow-lg rounded-md px-3 py-1.5 type-footnote text-fg-primary data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0 z-50">{content}{shortcut && <kbd className="ml-2 rounded-sm border border-container-border px-1.5 type-caption normal-case text-fg-secondary">{shortcut}</kbd>}</T.Content></T.Portal>`. If jsdom lacks `ResizeObserver`, add a minimal stub in `src/test/setup.ts`:

```ts
globalThis.ResizeObserver ??= class { observe() {} unobserve() {} disconnect() {} } as never;
```

**Step 4:** PASS. **Step 5:** commit `feat(ui): Tooltip`.

---

### Task 6: IconButton and ButtonGroup

**Files:** Create `components/icon-button.tsx`, `icon-button.test.tsx`, `components/button-group.tsx`

Reference: `bundle.js` l.488–502; `bundle.css` `.bd-ibtn*`, `.bd-bgroup*`.

**Step 1: Failing test**

```tsx
import { render, screen } from "@testing-library/react";
import { IconButton } from "#components/icon-button";
import { ButtonGroup } from "#components/button-group";

test("label is the accessible name and pressed sets aria-pressed", () => {
  render(<IconButton icon="moon" label="Dark mode" pressed tooltip={false} />);
  expect(screen.getByRole("button", { name: "Dark mode" })).toHaveAttribute("aria-pressed", "true");
});

test("numeric badge renders", () => {
  render(<IconButton icon="bell" label="Notifications" badge={3} tooltip={false} />);
  expect(screen.getByText("3")).toBeInTheDocument();
});

test("ButtonGroup is a labelled group", () => {
  render(<ButtonGroup label="View"><button>a</button></ButtonGroup>);
  expect(screen.getByRole("group", { name: "View" })).toBeInTheDocument();
});
```

**Step 3: Implement** IconButton: reuse `buttonVariants({ variant })` for colour only and add its own cva for `size` (`sm` 28px/16 icon, `md` 36/20, `lg` 44/24, `xl` 56/32) and `shape` (`circle` → `rounded-pill`, `square` → `rounded-md`, `diamond` → `rotate-45 rounded-md [&>svg]:-rotate-45`). Badge: `true` → 8px `bg-danger` dot top-right with `animate-pulse-dot`; number/string → min-w-4 h-4 pill `bg-danger text-white type-caption normal-case`. Wrap in `<Tooltip content={tooltip ?? label} placement={tooltipPlacement} shortcut={shortcut}>` unless `tooltip === false`. `label` is required in the type. ButtonGroup: `role="group"`, `glass shadow-button rounded-pill inline-flex p-0.5 gap-0.5`, children `[&>[data-slot=button]]:rounded-pill [&>[data-slot=button]]:shadow-none [&>[data-slot=button]]:border-transparent`; `size="sm"` shrinks padding.

**Step 4:** PASS. **Step 5:** commit `feat(ui): IconButton and ButtonGroup`.

---

### Task 7: Field frame, TextField, TextArea, Select (replace Input/Label)

**Files:**
- Create: `components/field.tsx` (internal, not exported), `text-field.tsx`, `text-field.test.tsx`, `text-area.tsx`, `select.tsx`, `select.test.tsx`
- Delete: `components/input.tsx`, `input.test.tsx`, `label.tsx`, `label.test.tsx` (the design says delete if unused; after Task 20 nothing uses them — verify with grep there, re-add Label only if a real use remains)

Reference: `bundle.js` l.374–415; `bundle.css` "Form field frame", "Text field", "Text area", "Select".

**Step 1: Failing tests**

`text-field.test.tsx`

```tsx
import { render, screen } from "@testing-library/react";
import { TextField } from "#components/text-field";
import { TextArea } from "#components/text-area";

test("label is associated and hint describes the input", () => {
  render(<TextField label="Email" hint="Work address" />);
  const input = screen.getByLabelText("Email");
  expect(input).toHaveAccessibleDescription("Work address");
  expect(input).not.toHaveAttribute("aria-invalid");
});

test("error replaces the hint and marks the input invalid", () => {
  render(<TextField label="Email" hint="Work address" error="Required" />);
  const input = screen.getByLabelText("Email");
  expect(input).toHaveAttribute("aria-invalid", "true");
  expect(input).toHaveAccessibleDescription("Required");
  expect(screen.getByRole("alert")).toHaveTextContent("Required");
  expect(screen.queryByText("Work address")).toBeNull();
});

test("TextArea shares the frame", () => {
  render(<TextArea label="Notes" error="Too long" />);
  expect(screen.getByLabelText("Notes")).toHaveAttribute("aria-invalid", "true");
});
```

`select.test.tsx`

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Select } from "#components/select";

test("renders string and object options and reports changes", async () => {
  const onChange = vi.fn();
  render(<Select aria-label="Role" options={["viewer", { value: "editor", label: "Editor" }]} defaultValue="viewer" onChange={onChange} />);
  await userEvent.selectOptions(screen.getByRole("combobox", { name: "Role" }), "editor");
  expect(onChange).toHaveBeenCalled();
  expect(screen.getByRole("option", { name: "Editor" })).toBeInTheDocument();
});

test("placeholder is a disabled empty option", () => {
  render(<Select aria-label="X" options={["a"]} placeholder="Pick one" defaultValue="" />);
  expect(screen.getByRole("option", { name: "Pick one" })).toBeDisabled();
});
```

**Step 3: Implement**

`field.tsx`:

```tsx
import * as React from "react";
import { cn } from "#lib/utils";

export type FieldFrameProps = { label?: React.ReactNode; hint?: React.ReactNode; error?: React.ReactNode };

// Shared label / hint / error frame for TextField, TextArea and Select.
export function Field({
  id, label, hint, error, disabled, className, children,
}: FieldFrameProps & { id: string; disabled?: boolean; className?: string;
  children: (a: { "aria-invalid"?: true; "aria-describedby"?: string }) => React.ReactNode }) {
  const msgId = `${id}-msg`;
  const msg = error ?? hint;
  return (
    <div data-slot="field" data-invalid={error ? "" : undefined}
      className={cn("group/field flex min-w-0 flex-col gap-1.5", disabled && "pointer-events-none opacity-50", className)}>
      {label && <label htmlFor={id} className="type-footnote font-medium text-fg-secondary">{label}</label>}
      {children({ "aria-invalid": error ? true : undefined, "aria-describedby": msg ? msgId : undefined })}
      {error ? (
        <div id={msgId} role="alert" className="type-footnote text-danger">{error}</div>
      ) : hint ? (
        <div id={msgId} className="type-footnote text-fg-secondary">{hint}</div>
      ) : null}
    </div>
  );
}

export const fieldControl =
  "glass rounded-md text-fg-primary transition-colors duration-fast focus-within:outline-2 focus-within:outline-solid focus-within:outline-offset-2 focus-within:outline-focus-ring group-data-invalid/field:border-danger";
```

`text-field.tsx`: `TextField({ label, hint, error, icon, trailing, className, id, disabled, ...props })` → `const autoId = React.useId(); const fid = id ?? autoId;` render `<Field …>{(aria) => <div className={cn(fieldControl, "flex h-11 items-center gap-2 px-4")}>{icon && <span aria-hidden className="inline-flex text-fg-primary">{renderIcon(icon)}</span>}<input id={fid} type="text" disabled={disabled} className="h-full min-w-0 flex-1 bg-transparent type-body outline-none placeholder:text-fg-secondary" {...aria} {...props} />{trailing}</div>}</Field>`. Props type: `Omit<React.ComponentProps<"input">, "children"> & FieldFrameProps & { icon?: IconProp; trailing?: React.ReactNode }`.

`text-area.tsx`: same pattern, `<textarea rows={4} className={cn(fieldControl, "block min-h-24 w-full resize-y px-4 py-3 type-body placeholder:text-fg-secondary outline-none")} …/>` (focus ring via `focus-visible:` since it is the element itself — swap `focus-within` for `focus-visible` with `cn`).

`select.tsx`: options normalised `typeof o === "string" ? { value: o, label: o } : o`; wrapper `cn(fieldControl, "relative inline-flex h-11 items-center gap-1 rounded-pill pr-4 pl-5")`; optional `prefix` span (`type-body font-medium`, `aria-hidden`); `<select className="min-w-0 flex-1 cursor-pointer appearance-none bg-transparent pr-7 type-body font-medium outline-none [&>option]:text-black">`; placeholder `<option value="" disabled>`; `<Icon name="chevronDown" className="pointer-events-none absolute right-4" />`. Props: `Omit<React.ComponentProps<"select">, "children" | "prefix"> & FieldFrameProps & { options: Array<string | SelectOption>; prefix?: React.ReactNode; placeholder?: string }`. Export `SelectOption`.

**Step 4:** PASS. **Step 5:** `git rm` input/label files; commit `feat(ui): TextField, TextArea, Select with a shared field frame`.

---

### Task 8: Checkbox, RadioGroup, Toggle (Radix)

**Files:** Create `components/checkbox.tsx`, `radio-group.tsx`, `toggle.tsx`, `choice.test.tsx`

Reference: `bundle.js` `Toggle` (l.362), `Checkbox` (l.416), `RadioGroup` (l.424); `bundle.css` "Toggle", "Checkbox and radio"; READMEs. Radix: `Checkbox`, `RadioGroup`, `Switch` from `radix-ui`.

**Step 1: Failing test** `choice.test.tsx`

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Checkbox } from "#components/checkbox";
import { RadioGroup } from "#components/radio-group";
import { Toggle } from "#components/toggle";

test("Checkbox toggles via its label", async () => {
  const onCheckedChange = vi.fn();
  render(<Checkbox label="Remember me" description="30 days" onCheckedChange={onCheckedChange} />);
  await userEvent.click(screen.getByText("Remember me"));
  expect(screen.getByRole("checkbox", { name: "Remember me" })).toBeChecked();
  expect(onCheckedChange).toHaveBeenCalledWith(true);
});

test("RadioGroup reports the chosen value", async () => {
  const onChange = vi.fn();
  render(<RadioGroup label="Trip" options={["Roundtrip", { value: "one", label: "One way" }]} defaultValue="Roundtrip" onChange={onChange} />);
  await userEvent.click(screen.getByRole("radio", { name: "One way" }));
  expect(onChange).toHaveBeenCalledWith("one");
  expect(screen.getByRole("radiogroup", { name: "Trip" })).toBeInTheDocument();
});

test("Toggle is a switch that reports changes", async () => {
  const onChange = vi.fn();
  render(<Toggle label="Notifications" onChange={onChange} />);
  await userEvent.click(screen.getByRole("switch", { name: "Notifications" }));
  expect(onChange).toHaveBeenCalledWith(true);
});
```

**Step 3: Implement**
- **Checkbox:** DS props `label`, `description`, `children` (alt label), plus Radix `checked`/`defaultChecked`/`onCheckedChange`/`disabled`/`name`/`required`. (The DS type extends the input's props; Radix's is the closer analogue — use `Omit<React.ComponentProps<typeof RadixCheckbox.Root>, "children"> & { label?; description?; children? }`.) Markup: `<label className="inline-flex items-start gap-3 cursor-pointer type-body text-fg-secondary has-data-checked:text-fg-primary">`, `Root` = 22px round box `rounded-pill border-[1.5px] border-fg-secondary data-checked:border-button data-checked:bg-button text-button-fg` + `focusRing`, `Indicator` → `<Icon name="check" size={14} />` with `ease-spring` scale-in. Text column: label, then description `type-footnote text-fg-secondary`.
- **RadioGroup:** props per `RadioGroupProps`; `onChange` mapped to Radix `onValueChange`; `aria-label={label}`; `direction` → `flex-col gap-4` / `flex-row flex-wrap gap-6`. Each item `<label>` with `RadioGroup.Item` (22px ring `border-button` when checked) + `Indicator` (12px `bg-button` dot).
- **Toggle:** props per `ToggleProps`; Radix `Switch.Root` 52×30 `rounded-pill glass data-checked:bg-button data-checked:border-button`; `Switch.Thumb` 22px `bg-fg-tertiary data-checked:bg-button-fg data-checked:translate-x-[22px] transition-transform duration-base ease-spring`. `onChange` ← `onCheckedChange`. Label beside it as `<label>` wrapping both.

**Step 4:** PASS. **Step 5:** commit `feat(ui): Checkbox, RadioGroup, Toggle`.

---

### Task 9: SegmentedControl, SearchField, Slider, Stepper, FieldTile

**Files:** Create `segmented-control.tsx`, `search-field.tsx`, `slider.tsx`, `stepper.tsx`, `field-tile.tsx`, `controls.test.tsx`

Reference: `bundle.js` `useIndicator` (l.69), `SegmentedControl` (l.120), `SearchField` (l.141), `Slider` (l.438), `Stepper` (l.455), `FieldTile` (l.467); matching CSS sections; READMEs.

**Step 1: Failing test** `controls.test.tsx`

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SegmentedControl } from "#components/segmented-control";
import { SearchField } from "#components/search-field";
import { Slider } from "#components/slider";
import { Stepper } from "#components/stepper";
import { FieldTile } from "#components/field-tile";

test("SegmentedControl selects an option", async () => {
  const onChange = vi.fn();
  render(<SegmentedControl label="Period" options={["Day", "Week"]} defaultValue="Day" onChange={onChange} />);
  await userEvent.click(screen.getByRole("radio", { name: "Week" }));
  expect(onChange).toHaveBeenCalledWith("Week");
  expect(screen.getByRole("radio", { name: "Week" })).toBeChecked();
});

test("SearchField is a labelled searchbox", () => {
  render(<SearchField label="Search logs" />);
  expect(screen.getByRole("searchbox", { name: "Search logs" })).toBeInTheDocument();
});

test("Stepper clamps to min and max", async () => {
  const onChange = vi.fn();
  render(<Stepper label="Bags" min={0} max={1} defaultValue={1} onChange={onChange} />);
  const inc = screen.getByRole("button", { name: /increase/i });
  expect(inc).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: /decrease/i }));
  expect(onChange).toHaveBeenLastCalledWith(0);
  expect(screen.getByRole("button", { name: /decrease/i })).toBeDisabled();
});

test("Slider exposes a formatted value", () => {
  render(<Slider label="Budget" defaultValue={50} format={(v) => `$${v}`} />);
  expect(screen.getByRole("slider")).toHaveAttribute("aria-valuetext", "$50");
});

test("FieldTile shows label and placeholder", () => {
  render(<FieldTile label="From" placeholder="Choose" />);
  expect(screen.getByRole("button", { name: /From.*Choose/ })).toBeInTheDocument();
});
```

**Step 3: Implement**
- **SegmentedControl:** Radix `ToggleGroup` would give `radio` role only with `type="single"` + `rovingFocus`; simpler and matching the DS: a `role="radiogroup"` div with native `<input type="radio" className="sr-only">` + `<label>` per option, so arrow keys and `toBeChecked` work natively. Sliding indicator: port `useIndicator` (measures the checked label's `offsetLeft/offsetWidth` in a `useLayoutEffect`, positions an absolutely placed pill). `tone="neutral"` → indicator `glass shadow-md`; `accent` → selected label `text-button-text`. Sizes `sm` 26px / `md` 30px.
- **SearchField:** `<input type="search">` in a `glass rounded-pill h-11 pl-5 pr-4` wrapper with leading `<Icon name="search" />`; `aria-label={label ?? "Search"}`.
- **Slider:** Radix `Slider` (`Root`/`Track`/`Range`/`Thumb`), `aria-valuetext` on the Thumb from `format`, value bubble above the thumb when `showValue`, header with `label` and a "Clear" `bd-link`-style button when `onClear`. Track `bg-chart-muted h-2 rounded-pill`, range `bg-button`.
- **Stepper:** `role="group" aria-label={label}`; two `IconButton`s (`minus`/`plus`, `size="sm"`, `tooltip={false}`, labels `Decrease ${label}` / `Increase ${label}`), value in between with `aria-live="polite"`; clamp `Math.min(max, Math.max(min, n))`; buttons disabled at the bounds. Controlled/uncontrolled like the DS.
- **FieldTile:** a `<button>` styled `glass shadow-glass rounded-lg p-4 text-left` with optional 24px icon, `type-caption` label, value in `type-title-3` (or placeholder in `text-fg-tertiary`).

**Step 4:** PASS. **Step 5:** commit `feat(ui): SegmentedControl, SearchField, Slider, Stepper, FieldTile`.

---

### Task 10: Tag (replaces Badge)

**Files:** Create `components/tag.tsx`, `tag.test.tsx`; delete `badge.tsx`, `badge.test.tsx`

Reference: `bundle.js` l.503–507; `bundle.css` l.219–229, 307, 434–435 and every `.bd-tone-*` rule (they set `--tag-c`).

**Step 1: Failing test**

```tsx
import { render, screen } from "@testing-library/react";
import { Tag } from "#components/tag";

test("dot variant renders a dot and the text", () => {
  const { container } = render(<Tag tone="success">Active</Tag>);
  expect(screen.getByText("Active")).toHaveAttribute("data-slot", "tag");
  expect(container.querySelector("[data-slot=tag-dot]")).not.toBeNull();
});

test("solid has no dot", () => {
  const { container } = render(<Tag variant="solid">Design</Tag>);
  expect(container.querySelector("[data-slot=tag-dot]")).toBeNull();
});
```

`getByText("Active")` must return the root span: render the dot and the text as siblings inside the root (text node directly in root).

**Step 3: Implement** cva on the root with `variant` (`dot` = `glass shadow-sm h-6 px-2 rounded-pill type-footnote font-medium`, `outline` = `h-6 px-2 rounded-pill border border-(--tag-c) bg-transparent`, `solid` = `h-[22px] px-2 rounded-sm type-caption text-button-fg bg-(--tag-c)`) and `tone` setting `[--tag-c:…]`: `neutral` → `var(--fg-tertiary)` (solid → `var(--fg-secondary)`), `accent` → `var(--button-text)` (solid → `var(--button)`), `success` → `var(--success)`, `danger` → `var(--danger)`, `purple` → `var(--chart-3)`, `teal` → `var(--chart-4)`, `violet` → `var(--chart-2)` — confirm each against the `.bd-tone-*` rules in `bundle.css` and use theirs if different. Dot: `size-2 rounded-pill bg-(--tag-c)` with `data-slot="tag-dot"`.

**Step 4:** PASS. **Step 5:** commit `feat(ui): Tag replaces Badge`.

---

### Task 11: Card (DS API + action slot)

**Files:** Rewrite `components/card.tsx`, `card.test.tsx`

Reference: `bundle.js` l.148–155; `bundle.css` "Card", `.bd-card-tint-*`, `.bd-card-interactive`, `.bd-elev-*`, sheen-sweep keyframes.

**Step 1: Failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Card } from "#components/card";

test("renders title, description, action and children", () => {
  render(<Card title="Q3" description="Wins" action={<button>Edit</button>}>body</Card>);
  expect(screen.getByRole("heading", { name: "Q3" })).toBeInTheDocument();
  expect(screen.getByText("Wins")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
  expect(screen.getByText("body")).toBeInTheDocument();
});

test("onClick makes it interactive and keyboard-focusable", async () => {
  const onClick = vi.fn();
  render(<Card title="Open" onClick={onClick} />);
  const card = screen.getByText("Open").closest("[data-slot=card]")!;
  expect(card).toHaveAttribute("tabindex", "0");
  expect(card).toHaveAttribute("data-interactive");
  await userEvent.click(card);
  expect(onClick).toHaveBeenCalled();
});
```

**Step 3: Implement** props per `CardProps` (+ `action?: React.ReactNode`, `className`, and `...rest` div props except `title`, which is the heading). `as` default `"div"`. Root: `glass shadow-glass rounded-lg p-6 text-fg-primary` (`compact` → `p-4`), `tint` via cva over a `bg-linear-*` overlay: `sheen` (default) → `bg-[linear-gradient(135deg,var(--glass-sheen),transparent_60%)]`, `accent` → `glass-tint-blue → glass-tint-violet`, `violet` → radial corner `glass-tint-violet`, `rose` → bottom `glass-tint-rose`, `cool` → `glass-tint-teal → glass-tint-blue`, `none` → nothing. Copy the exact gradient geometry from `.bd-card-tint-*`. `elevation` → `shadow-sm|md|lg|xl`. `interactive || onClick` → `data-interactive`, `cursor-pointer transition-[translate,box-shadow] duration-slow ease-standard hover:-translate-y-1 hover:shadow-lg`, `tabIndex={0}`, and Enter/Space call `onClick` (`onKeyDown`). `animate` → `animate-enter` with `style={{ "--delay": `${delay}ms` }}`. Header: when `title || action`, a `flex items-start gap-4` row: `<h3 className="type-title-3 mb-2 flex-1">` and `action`; `description` → `<p className="type-body text-fg-secondary">`.

**Step 4:** PASS. **Step 5:** commit `feat(ui): Card on the DS API`.

---

### Task 12: Restyle Dialog and DropdownMenu (APIs unchanged)

**Files:** Modify `components/dialog.tsx`, `components/dropdown-menu.tsx`

**Step 1:** run their existing tests → PASS (baseline).

**Step 2: Dialog** — change only classes:
- Overlay: `fixed inset-0 z-50 bg-black/30 backdrop-blur-sm` (+ existing animate-in/out).
- Content: `glass shadow-xl rounded-xl p-6 gap-4 text-fg-primary` (keep positioning and animation classes; remove `bg-background`, `ring-*`, `rounded-*` that conflict).
- Close button: replace with `<IconButton icon="close" label="Close" variant="tinted" size="sm" tooltip={false} />` inside `DialogPrimitive.Close asChild`.
- Title: `type-title-3`. Description: `type-body text-fg-secondary`. Footer: `flex flex-col-reverse gap-2 sm:flex-row sm:justify-end` (drop the shadcn muted footer background).

**Step 3: DropdownMenu** — Content: `glass shadow-md rounded-lg p-2 min-w-48 z-50`; Item: `flex h-10 items-center gap-3 rounded-md px-3 type-callout text-fg-secondary outline-none data-highlighted:bg-container data-highlighted:text-fg-primary data-[variant=danger]:text-danger [&_svg]:size-5`; add `variant?: "default" | "danger"` prop on `DropdownMenuItem` (sets `data-variant`); Separator: `my-2 mx-3 h-px bg-divider`. Replace any `variant="destructive"` support with `danger` (grep the app in Task 19).

**Step 4:** tests PASS (update any assertion on removed classes). **Step 5:** commit `style(ui): glass Dialog and DropdownMenu`.

---

### Task 13: MenuList

**Files:** Create `components/menu-list.tsx`, `menu-list.test.tsx`

Reference: `bundle.js` l.166–233 (`idOf`, `containsId`, `MenuList`); `bundle.css` "Menu list" and flyout rules; `ds/components/MenuList/README.md`.

**Step 1: Failing test**

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MenuList } from "#components/menu-list";

const items = [
  "Workspace",
  { label: "Dashboard", value: "dash", icon: "grid" },
  { label: "Reports", children: [{ label: "Weekly", value: "weekly" }] },
  "separator",
  { label: "Docs", href: "https://x.dev", external: true },
] as const;

test("selecting a row updates aria-current and calls onSelect", async () => {
  const onSelect = vi.fn();
  render(<MenuList label="Main" items={[...items]} defaultValue="dash" onSelect={onSelect} submenu="inline" />);
  expect(screen.getByRole("button", { name: "Dashboard" })).toHaveAttribute("aria-current", "page");
  await userEvent.click(screen.getByRole("button", { name: "Reports" }));
  await userEvent.click(screen.getByRole("button", { name: "Weekly" }));
  expect(onSelect).toHaveBeenCalledWith("weekly", expect.objectContaining({ label: "Weekly" }));
});

test("inline groups expand with aria-expanded; the group holding the value starts open", () => {
  render(<MenuList items={[...items]} value="weekly" submenu="inline" />);
  expect(screen.getByRole("button", { name: "Reports" })).toHaveAttribute("aria-expanded", "true");
});

test("external links open in a new tab and headings render", () => {
  render(<MenuList items={[...items]} />);
  expect(screen.getByRole("link", { name: /Docs/ })).toHaveAttribute("target", "_blank");
  expect(screen.getByText("Workspace")).toBeInTheDocument();
});
```

**Step 3: Implement** per DS: `<nav aria-label={label}>` → `<ul>` with `glass shadow-glass rounded-lg p-2` unless `glass={false}`. Row: `h-11 rounded-md px-4 gap-3 type-body font-medium text-fg-secondary hover:bg-container hover:text-fg-primary`, selected row `bg-container text-fg-primary shadow-sm` + leading 2px `bg-button` bar (`before:` pseudo). String item → `type-caption text-fg-tertiary px-4 pt-3 pb-1` heading; `"separator"` → `h-px bg-divider mx-3 my-2`. `badge` → count pill. Groups: `submenu="inline"` → accordion with rotating chevron (`aria-expanded`, opens if `defaultOpen` or `containsId(item, value)`); `submenu="flyout"` (default) → Radix `Popover` (`side="right"`, open on hover/focus/click/ArrowRight; content styled like the Tooltip panel, rows rendered by the same row component). Controlled/uncontrolled `value`/`defaultValue`.

**Step 4:** PASS. **Step 5:** commit `feat(ui): MenuList`.

---

### Task 14: UserMenu, TopBar, PageHeader

**Files:** Create `user-menu.tsx`, `top-bar.tsx`, `page-header.tsx`, `nav.test.tsx`

Reference: `bundle.js` `UserMenu` (l.234), `TopBar` (l.277), `PageHeader` (l.345); CSS sections; READMEs; `index.d.ts` for `TopBarProps`/`PageHeaderProps`.

**Step 1: Failing test** `nav.test.tsx`

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { UserMenu } from "#components/user-menu";
import { TopBar } from "#components/top-bar";
import { PageHeader } from "#components/page-header";

test("UserMenu opens and reports the chosen item", async () => {
  const onSelect = vi.fn();
  render(<UserMenu user={{ name: "Ada", role: "Admin" }} items={[{ label: "Sign out", tone: "danger" }]} onSelect={onSelect} />);
  await userEvent.click(screen.getByRole("button", { name: /Ada/ }));
  await userEvent.click(await screen.findByRole("menuitem", { name: "Sign out" }));
  expect(onSelect).toHaveBeenCalledWith("Sign out", expect.objectContaining({ tone: "danger" }));
});

test("TopBar renders brand heading, nav links, actions and the user", () => {
  render(
    <TopBar
      brand={{ name: "Brag Document", href: "/" }}
      items={[{ label: "Settings", href: "/settings" }]}
      actions={<button>theme</button>}
      user={{ user: { name: "a@acme.com" } }}
    />,
  );
  expect(screen.getByRole("heading", { name: "Brag Document" })).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute("href", "/settings");
  expect(screen.getByRole("button", { name: "theme" })).toBeInTheDocument();
  expect(screen.getByText("a@acme.com")).toBeInTheDocument();
});

test("PageHeader renders an h1 and actions", () => {
  render(<PageHeader title="Documents" actions={<button>New</button>} />);
  expect(screen.getByRole("heading", { level: 1, name: "Documents" })).toBeInTheDocument();
});
```

**Step 3: Implement**
- **UserMenu** on the package's own `DropdownMenu` (Task 12). Trigger: `variant="pill"` → glass pill with avatar (photo or initials, status dot), name, role, chevronDown; `variant="avatar"` → avatar only with `aria-label={user.name}`. Default items: Account, Billing, separator, Sign out (danger). `placement` → Radix `side`/`align`. Items with `href` render `asChild` anchors; `shortcut` right-aligned `type-footnote text-fg-tertiary`.
- **TopBar:** `<header>` glass bar `rounded-xl shadow-glass h-16 px-4 flex items-center gap-6`. Brand: `<h1 className="type-title-3 font-semibold">` wrapping a link when `href` (logo optional). Nav: links for items with `href` (`aria-current="page"` when `value` matches); items with `children` open a `DropdownMenu` whose rows show icon, label and description. `search` → `SearchField`; `actions`; `user` → `UserMenu` (`variant` forced to the TopBar default from the DS, `avatar` — **except** when `user.variant` is set explicitly, which the app will do to show the email); `cta`. Under `md:` the nav collapses behind an `IconButton icon="menu"` that opens the items in a `DropdownMenu`. **App links:** TopBar must accept a `renderLink?: (item, props) => ReactNode` so the app can use react-router `<Link>` — default renders `<a href>`. (This is the one API addition; the DS uses plain anchors.)
- **PageHeader:** props per `PageHeaderProps` in `index.d.ts` (title, subtitle/description, eyebrow, actions, breadcrumb if present); `<h1 className="type-title-1">`.

**Step 4:** PASS. **Step 5:** commit `feat(ui): UserMenu, TopBar, PageHeader`.

---

### Task 15: Presentational content components

**Files:** Create `media-cell.tsx`, `media-card.tsx`, `hero-header.tsx`, `notification-item.tsx`, `content.test.tsx`

Reference: `bundle.js` `HeroHeader` (l.57), `MediaCard` (l.91), `NotificationItem` (l.157, `initials` l.156), `MediaCell` (l.514); CSS sections `.bd-hero*`, `.bd-mcard*`, `.bd-notif*`, `.bd-mc*`.

**Step 1: Smoke test** (these are presentational; one test per component for the one rule that matters)

```tsx
import { render, screen } from "@testing-library/react";
import { MediaCell } from "#components/media-cell";
import { MediaCard } from "#components/media-card";
import { HeroHeader } from "#components/hero-header";
import { NotificationItem } from "#components/notification-item";

test("MediaCell draws initials without an image", () => {
  render(<MediaCell title="Eva Solain" />);
  expect(screen.getByText("ES")).toBeInTheDocument();
});

test("MediaCard with href makes the title a link", () => {
  render(<MediaCard title="Course" href="/c" />);
  expect(screen.getByRole("link", { name: "Course" })).toHaveAttribute("href", "/c");
});

test("HeroHeader uses the requested heading level", () => {
  render(<HeroHeader as="h2" lead="Power your" title="brag doc" />);
  expect(screen.getByRole("heading", { level: 2 })).toHaveTextContent("Power your brag doc");
});

test("NotificationItem reads name + action + time", () => {
  render(<NotificationItem name="Eva" action="invited you" time="5m ago" unread />);
  expect(screen.getByText("Eva")).toBeInTheDocument();
  expect(screen.getByText("5m ago")).toBeInTheDocument();
});
```

**Step 3: Implement** each by translating its DS function + CSS. Put the `initials(name)` helper in `media-cell.tsx` and import it in `notification-item.tsx`. Image strings starting with `gradient-` render a span with `style={{ background: `var(--${image})` }}`. HeroHeader gradients: `primary` → `text-gradient`, `secondary` → `text-gradient-secondary`, `accent` → `text-gradient-accent`, `none` → plain; sizes → `type-display|type-title-1|type-title-2`. MediaCard `progress` → `role="progressbar"` with `aria-valuenow`.

**Step 4:** PASS. **Step 5:** commit `feat(ui): MediaCell, MediaCard, HeroHeader, NotificationItem`.

---

### Task 16: DataTable and Pagination

**Files:** Create `data-table.tsx`, `pagination.tsx`, `lib/page-list.ts`, `data-table.test.tsx`, `pagination.test.tsx`

Reference: `bundle.js` l.509–622 (shown in full in the design session; port the logic as-is); CSS `.bd-dt*`, `.bd-pg*`.

**Step 1: Failing tests**

`pagination.test.tsx`

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { pageList } from "#lib/page-list";
import { Pagination } from "#components/pagination";

test("pageList windows around the current page", () => {
  expect(pageList(1, 3, 1)).toEqual([1, 2, 3]);
  expect(pageList(6, 12, 1)).toEqual([1, "gap-l", 5, 6, 7, "gap-r", 12]);
  expect(pageList(1, 12, 1)).toEqual([1, 2, 3, 4, "gap-r", 12]);
  expect(pageList(12, 12, 1)).toEqual([1, "gap-l", 9, 10, 11, 12]);
});

test("Pagination moves pages and marks the current one", async () => {
  const onChange = vi.fn();
  render(<Pagination pageCount={5} defaultPage={2} onChange={onChange} total={48} pageSize={10} />);
  expect(screen.getByRole("button", { name: "Page 2" })).toHaveAttribute("aria-current", "page");
  await userEvent.click(screen.getByRole("button", { name: "Next page" }));
  expect(onChange).toHaveBeenCalledWith(3);
  expect(screen.getByText("21–30 of 48")).toBeInTheDocument();
});
```

Before relying on the expected arrays above, run the DS `pageList` in node once to confirm them (`node -e` with the function pasted) and correct the test if the DS output differs — the DS is the spec.

`data-table.test.tsx`

```tsx
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { DataTable } from "#components/data-table";

const rows = [
  { id: 1, name: "Beta", n: 2 },
  { id: 2, name: "alpha", n: 10 },
];
const columns = [
  { key: "name", header: "Name", sortable: true, primary: true },
  { key: "n", header: "Count", sortable: true, align: "right" as const },
];
const names = () => screen.getAllByRole("row").slice(1).map((r) => within(r).getAllByRole("cell")[0].textContent);

test("sorts asc, desc, then clears", async () => {
  render(<DataTable columns={columns} rows={rows} />);
  const sortBtn = screen.getByRole("button", { name: /Name/ });
  await userEvent.click(sortBtn);
  expect(names()).toEqual(["alpha", "Beta"]);
  expect(screen.getByRole("columnheader", { name: /Name/ })).toHaveAttribute("aria-sort", "ascending");
  await userEvent.click(sortBtn);
  expect(names()).toEqual(["Beta", "alpha"]);
  await userEvent.click(sortBtn);
  expect(screen.getByRole("columnheader", { name: /Name/ })).toHaveAttribute("aria-sort", "none");
});

test("selection shows the count and bulk actions; clear resets", async () => {
  const bulk = vi.fn();
  render(<DataTable columns={columns} rows={rows} selectable rowLabel={(r) => r.name}
    bulkActions={(keys) => <button onClick={() => bulk(keys)}>Archive</button>} />);
  await userEvent.click(screen.getByRole("checkbox", { name: "Select row Beta" }));
  expect(screen.getByText("1 selected")).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Archive" }));
  expect(bulk).toHaveBeenCalledWith([1]);
  await userEvent.click(screen.getByRole("checkbox", { name: "Select all rows" }));
  expect(screen.getByText("2 selected")).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Clear" }));
  expect(screen.queryByText(/selected/)).toBeNull();
});

test("empty state spans the table", () => {
  render(<DataTable columns={columns} rows={[]} empty="Nothing yet" />);
  expect(screen.getByRole("cell", { name: "Nothing yet" })).toHaveAttribute("colspan", "2");
});
```

**Step 3: Implement**
- `lib/page-list.ts`: the DS `pageList` verbatim, typed `(page: number, count: number, sib: number): Array<number | "gap-l" | "gap-r">`.
- `data-table.tsx`: generic `DataTable<Row>`; port the DS logic (controlled/uncontrolled sort and selection, `cmp` comparator, `pick`, indeterminate header checkbox via ref, `manualSort`). Checkboxes: native `<input type="checkbox">` styled as the DS square box (`rounded-sm`, `bg-button` when checked) — keep native here for the `indeterminate` property. Column renderers: `render` > `media` (MediaCell) > `subtitle` (two-line) and `tags` (Tag list). Bar: title + toolbar, or `N selected` (`aria-live`) + bulk actions + `Button variant="tinted" size="sm"` Clear. Container `glass shadow-glass rounded-lg overflow-hidden`; header cells `type-footnote font-medium text-fg-secondary`; rows separated by `border-b border-divider`; selected row `bg-glass-tint-blue`; `density` → row heights 40/52/64. `<caption className="sr-only">` when `caption`. Row actions column with `sr-only` "Actions" header.
- `pagination.tsx`: port as-is using `IconButton`, `Select`, `Icon`; page buttons `size-8 rounded-pill type-callout`, current `bg-button text-button-fg`.

**Step 4:** PASS. **Step 5:** commit `feat(ui): DataTable and Pagination`.

---

### Task 17: Package exports

**Files:** Rewrite `frontend/packages/ui/src/index.ts`

**Step 1:** export everything, alphabetical by file, with prop types:

```ts
export { cn, focusRing } from "#lib/utils";
export { useTheme, type Theme } from "#lib/theme";
export { Button, buttonVariants, type ButtonProps } from "#components/button";
export { ButtonGroup } from "#components/button-group";
export { Card } from "#components/card";
export { Checkbox } from "#components/checkbox";
export { DataTable, type DataTableColumn, type DataTableSort } from "#components/data-table";
export { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogOverlay, DialogPortal, DialogTitle, DialogTrigger } from "#components/dialog";
export { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "#components/dropdown-menu";
export { FieldTile } from "#components/field-tile";
export { HeroHeader } from "#components/hero-header";
export { Icon, type IconProp } from "#components/icon";
export { IconButton } from "#components/icon-button";
export { MediaCard } from "#components/media-card";
export { MediaCell } from "#components/media-cell";
export { MenuList, type MenuItem } from "#components/menu-list";
export { NotificationItem } from "#components/notification-item";
export { PageHeader } from "#components/page-header";
export { Pagination } from "#components/pagination";
export { RadioGroup } from "#components/radio-group";
export { SearchField } from "#components/search-field";
export { SegmentedControl } from "#components/segmented-control";
export { Select, type SelectOption } from "#components/select";
export { Slider } from "#components/slider";
export { Stepper } from "#components/stepper";
export { Tag } from "#components/tag";
export { TextArea } from "#components/text-area";
export { TextField } from "#components/text-field";
export { Toggle } from "#components/toggle";
export { Tooltip } from "#components/tooltip";
export { TopBar } from "#components/top-bar";
export { UserMenu } from "#components/user-menu";
```

**Step 2:** `npm run typecheck -w @bragdoc/ui && npm test -w @bragdoc/ui` → both PASS. `npm run fmt` then `npm run lint` on the package.

**Step 3:** commit `feat(ui): export the full DS component set`.

---

### Task 18: App shell — TopBar, theme toggle, backdrop

**Files:** Modify `frontend/packages/app/src/routes/Root.tsx`, `routes/Root.test.tsx` (only if needed)

**Step 1:** Rewrite the header:

```tsx
import { Link, NavLink, Outlet, useNavigate } from "react-router";
import { IconButton, TopBar, useTheme } from "@bragdoc/ui";
// …existing hooks…
const { theme, toggle } = useTheme();
const items = [
  me?.role === "admin" && { label: me.tenant.name, href: "/tenant" },
  canAudit && { label: "Audit log", href: "/audit" },
  import.meta.env.DEV && { label: "Kitchen sink", href: "/kitchen-sink" },
  { label: "Settings", href: "/settings" },
].filter(Boolean);

return (
  <div className="min-h-screen bd-backdrop">
    <div className="mx-auto max-w-5xl p-4">
      <TopBar
        brand={{ name: "Brag Document", href: "/" }}
        items={items}
        renderLink={(item, props) => <NavLink to={item.href!} {...props}>{item.label}</NavLink>}
        actions={
          <IconButton icon={theme === "dark" ? "sun" : "moon"} label="Toggle dark mode"
            pressed={theme === "dark"} onClick={toggle} />
        }
        user={{ variant: "pill", user: { name: me?.email ?? "" },
          items: [{ label: "Sign out", icon: "signOut", tone: "danger", onSelect: signOut }] }}
      />
    </div>
    <main className="mx-auto max-w-5xl p-4"><Outlet /></main>
  </div>
);
```

Adjust the `items` typing (`.filter((i): i is TopBarItem => Boolean(i))`) and `renderLink`'s props to whatever Task 14 defined; keep the brand a heading named "Brag Document".

**Step 2:** `npx vitest run packages/app/src/routes/Root.test.tsx` → existing tests (heading "Brag Document", email text, Settings link) PASS. If any test clicked "Sign out" directly, update it to open the user menu first.

**Step 3:** commit `feat(app): TopBar shell with theme toggle`.

---

### Task 19: App — Button migration

**Files:** every app file importing `Button` (see `grep -rln '<Button' frontend/packages/app/src`)

**Step 1: Mechanical rename** (from `frontend/packages/app/src`):

```bash
grep -rl '<Button' . | xargs sed -i '' \
  -e 's/variant="outline"/variant="glass"/g' \
  -e 's/variant="secondary"/variant="glass"/g' \
  -e 's/variant="ghost"/variant="tinted"/g' \
  -e 's/variant="link"/variant="tinted"/g' \
  -e 's/variant="destructive"/variant="danger"/g'
```

This sed also touches `DropdownMenuItem variant="destructive"` → `danger`, which matches Task 12. Check `git diff --stat` and eyeball: no non-Button `variant=` props should have changed (grep `variant="glass"` and confirm each is on a `<Button`).

**Step 2: By hand**
- Buttons with no `variant` (shadcn default) that are the form's primary action → add `variant="primary"`. List them: `grep -rn '<Button' . | grep -v variant=`. Default now means glass, so every one of these needs a decision: submit / main CTA → `primary`, others stay glass.
- `size="icon-sm"` (3) → `<IconButton icon=… label=… size="sm" />`; take the `label` from the existing `aria-label`/`sr-only` text and the icon from the lucide child (pass the lucide element as `icon`).
- `size="xs"`/`"lg"`/`"default"` → `sm`/`lg`/drop.
- `asChild` (5) — unchanged API.
- `className` overrides that reference removed tokens (`text-muted-foreground`, `bg-muted`, `text-destructive`) — handled in Task 21.

**Step 3:** `npm run typecheck -w @bragdoc/app` — remaining errors should only be Input/Label/Badge/Card (next task). Commit `refactor(app): Buttons on DS variants`.

---

### Task 20: App — TextField, Tag, Card

**Files:** consumers of `Input`, `Label`, `Badge`, `Card*` (from the import list: `LogFilters`, `Audit`, `Tenant`, `AuthCard`, `SharePanel`, `ExportDialog`, `DocumentFormDialog`, `LogFormDialog`, `LogRow`, `DocumentCard`, `DocumentLogs`, `Settings`, `Documents`, `KitchenSink` (rewritten in Task 22 — skip it here))

**Step 1: Input + Label → TextField.** Pattern:

```tsx
// before
<Label htmlFor="title">Title</Label>
<Input id="title" value={title} onChange={…} aria-invalid={!!err} />
{err && <p className="text-sm text-destructive">{err}</p>}
// after
<TextField id="title" label="Title" value={title} onChange={…} error={err} />
```

Keep `id`s and `name`s so existing tests using `getByLabelText` keep working. Inputs without a Label keep their `aria-label` and become `<TextField aria-label=… />`.

**Step 2: Badge → Tag.** Map: `variant="secondary"`/none → `<Tag>` (dot, neutral); `outline` → `variant="outline"`; `destructive` → `tone="danger"`; status-like badges (Active/Archived/Shared/role) pick a tone: success for active/owner, accent for editor, neutral for viewer/archived. Decide per site by reading what the badge says.

**Step 3: Card subparts → DS Card.**

```tsx
// before
<Card><CardHeader><CardTitle>X</CardTitle><CardDescription>Y</CardDescription><CardAction>{menu}</CardAction></CardHeader>
  <CardContent>{body}</CardContent><CardFooter>{f}</CardFooter></Card>
// after
<Card title="X" description="Y" action={menu}>
  {body}
  <div className="mt-4 flex items-center gap-2 border-t border-divider pt-4">{f}</div>
</Card>
```

`DocumentCard` with an `onClick`/link → `interactive`.

**Step 4:** `npm run typecheck -w @bragdoc/app` (only KitchenSink errors left) and `npm test -w @bragdoc/app` — fix test queries that relied on removed markup (e.g. `data-slot="badge"` → `"tag"`). Commit `refactor(app): TextField, Tag and Card on DS APIs`.

---

### Task 21: App — native controls and leftover tokens

**Files:** `sharing/SharePanel.tsx`, `routes/Dashboard.tsx`, `routes/Audit.tsx`, `logs/LogFormDialog.tsx`, `logs/LogRow.tsx`, plus any file using old token classes

**Step 1: `<select>` → `Select`** (8 sites). Pattern from `Audit.tsx`:

```tsx
<Select
  aria-label="User"
  value={filters.actor}
  onChange={(e) => set("actor", e.target.value)}
  options={[{ value: "", label: "All users" },
    ...(options.data?.actors ?? []).map((a) => ({ value: a.id, label: a.name || a.email }))]}
/>
```

Delete `FIELD`/`selectClass` constants and the `RoleOptions` helper and its `ponytail:` comment in `SharePanel.tsx` (replace with a `const roleOptions = [{ value: "viewer", label: "Viewer" }, { value: "editor", label: "Editor" }]`).

**Step 2: `<textarea>` → `TextArea`** in `LogFormDialog.tsx:230`, keeping `id`/`name`/`aria-*` and moving its label/error into props.

**Step 3: Audit `<table>` → `DataTable`.** Build `columns` from the existing `<th>`s; each cell's current JSX becomes that column's `render`. Rows keyed by entry id. Keep infinite-scroll "Load more" as the `footer`. Keep the existing empty-state text as `empty`. Run `Audit.test.tsx` and fix queries (`getAllByRole("row")` still works).

**Step 4: Token sweep.** `grep -rnE 'muted-foreground|bg-muted|text-destructive|bg-background|text-foreground|bg-card|border-input|ring-ring|amber-' frontend/packages/app/src` and replace: `text-muted-foreground` → `text-fg-secondary`; `bg-muted` → `bg-container`; `text-destructive` → `text-danger`; `bg-background`/`bg-card` → `glass` or drop; headings `text-xl font-semibold` → `type-title-2`. Amber warning boxes (`LogFormDialog.tsx:137`, `LogRow.tsx:58`) → `rounded-md border border-danger bg-glass-tint-rose p-3 type-footnote text-fg-primary` and `text-danger` respectively (warnings keep an icon or text, never colour alone). Chart colours in `dashboard/` → `chart-1..4` / `chart-muted` utilities if they hard-code colours.

**Step 5:** `npm run typecheck && npm test` (KitchenSink may still error) → commit `refactor(app): DS Select, TextArea, DataTable and tokens`.

---

### Task 22: KitchenSink

**Files:** Rewrite `frontend/packages/app/src/routes/KitchenSink.tsx`

**Step 1:** One `<section>` per component group (Buttons & IconButtons & ButtonGroup & Tooltip; Forms; Tags; Cards & MediaCard; Navigation (MenuList, TopBar, UserMenu, PageHeader, SegmentedControl); Content (HeroHeader, NotificationItem, MediaCell); DataTable + Pagination; Dialog + DropdownMenu), each showing every variant/size/tone from `ds/preview/<Name>.html`. Section headings `type-title-2`. No tests (dev-only route).

**Step 2:** `npm run typecheck && npm test && npm run lint && npm run fmt:check` → all PASS.

**Step 3:** commit `feat(app): KitchenSink shows the full DS`.

---

### Task 23: Visual verification

**Step 1:** Use the `run` skill to start the app (`npm run dev` in `frontend/`) and the browser to screenshot, in light and dark (toggle in the header, and once with no stored theme + OS dark):
- Sign in, Documents, a document's logs (with the log form dialog open), Dashboard, Settings, Sharing panel, Audit, KitchenSink.

**Step 2:** Compare KitchenSink against `ds/preview/index.html` (open it directly). Fix visual drift (wrong radius, missing blur, contrast on `fg-tertiary`). Check focus rings by tabbing through the header and a dialog. Check `prefers-reduced-motion` in devtools rendering emulation.

**Step 3:** commit any fixes `fix(ui): visual parity with DS preview`.

---

### Task 24: Full CI and PR

**Step 1:** from `frontend/`: `npm run typecheck && npm test && npm run lint && npm run fmt:check`; also run any other workflow steps in `.github/workflows/` that touch `frontend/` (read them; e.g. `npm run build`). All green, with output captured.

**Step 2:** `git push -u origin feat/ui-design-system` and open the PR to `main`:

```bash
gh pr create --base main --title "feat(ui): adopt the BragDoc glass design system" --body "$(cat <<'EOF'
## Summary
- DS tokens, light/dark theme (OS default + stored toggle), glass utilities in `@bragdoc/ui`
- Button/Card/TextField/Tag move to the DS APIs; Dialog and DropdownMenu restyled
- 23 new components ported from `ds/` (forms, navigation, DataTable, Pagination, content)
- App migrated: TopBar shell, DS Select/TextArea/DataTable replace native controls
- Design: docs/plans/2026-10-09-ui-design-system-design.md

## Test plan
- [ ] typecheck, test, lint, fmt:check green
- [ ] Screens checked in light and dark against ds/preview

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
