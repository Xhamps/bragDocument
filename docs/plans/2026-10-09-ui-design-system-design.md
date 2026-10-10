# UI design system adoption — design

Date: 2026-10-09
Branch: `feat/ui-design-system` (one PR to `main`)

## Goal

Restyle `frontend/packages/ui` with the BragDoc glass design system exported to `ds/`, adopt the DS component APIs, port every DS component, and migrate the app to them.

## Decisions

- **Adopt DS APIs.** Components take the props in `ds/components/index.d.ts`; consumers are migrated.
- **Port all DS components** (27 + Icon). `Elevation` and `Cover` are reference/preview pages, not components, and are not ported.
- **Tailwind + cva**, not the DS `bundle.css`. `bd-*` rules are rewritten as utilities and variants.
- **Radix where it fits** (Tooltip, UserMenu, DropdownMenu, Dialog, Checkbox, RadioGroup, Toggle/Switch, Slider, MenuList flyout). `Select` stays a native `<select>`, as the DS API extends it.
- **Theme:** OS default, `data-theme` override, plus a moon/sun toggle persisted in `localStorage`.
- **One PR.**

## 1. Foundation

`packages/ui/src/styles.css` is rewritten:

- Remove the shadcn oklch tokens and the `.dark`/`.light` blocks.
- Copy DS tokens from `ds/tokens.css`:
  - Light on `:root, [data-theme="light"]`.
  - Dark on `[data-theme="dark"]` and `@media (prefers-color-scheme: dark) { :root:not([data-theme="light"]) }`.
  - Theme-independent: space, radius, blur, gradients, motion, fonts.
- `@theme inline` exposes them as utilities: colours (`page`, `surface`, `fg-primary|secondary|tertiary`, `container`, `container-border`, `container-divider`, `button`, `button-hover`, `button-fg`, `button-text`, `button-inactive`, `focus-ring`, `success`, `danger`, `chart-1..4`, `chart-muted`, glass tints), radii (`sm|md|lg|xl|pill` = 6/10/16/24/999), shadows (`sm|md|lg|xl|inset|glass|button|glow|glow-strong|cta`), durations, easings, fonts.
- Type scale as `@utility`: `text-display`, `text-title-1..3`, `text-body`, `text-callout`, `text-footnote`, `text-caption`, `text-code` (DS metrics and tracking).
- Helper utilities: `glass`, `bd-backdrop` (three mood radial washes on `page`), `text-gradient`, `text-gradient-secondary`, `animate-enter` (staggered via `--delay`).
- `@custom-variant dark` targets `[data-theme=dark]` and the OS media query.
- Global `prefers-reduced-motion` collapses durations.
- Inter loaded from Google Fonts (400/500/600/700) in the app's `index.html`.

Theme:

- `useTheme()` in `packages/ui`: reads/writes `localStorage["theme"]`, sets `document.documentElement.dataset.theme`; unset means OS decides.
- Inline script in `index.html` applies the stored theme before first paint.
- Toggle is an `IconButton` (moon/sun, `pressed`) in the header.

DS images in `ds/assets` are not copied.

## 2. Components

### Existing → DS API

| Today | Becomes | Migration |
|---|---|---|
| `Button` | DS `Button` (`variant` glass/primary/ghost/tinted/gradient, `size` sm/md/lg, `chevron`, `glow`, `icon`, `trailingIcon`, `pill`, `fullWidth`) + extensions: `danger` variant, `asChild` | default→primary, outline/secondary→glass, ghost/link→tinted, destructive→danger |
| `Button size="icon-sm"` | `IconButton` | 3 sites |
| `Input` + `Label` | `TextField` (`label`, `hint`, `error`, `icon`, `trailing`) | Labels fold in; delete `Input`/`Label` if unused |
| `Badge` | `Tag` (`variant` dot/outline/solid, `tone`) | 18 sites |
| `Card` + subparts | DS `Card` (`title`, `description`, `tint`, `interactive`, `elevation`, `animate`, `compact`) + extension: `action` slot | Subparts removed; footer becomes children |
| `Dialog` | Radix, API unchanged; glass, `radius-xl`, `shadow-xl`, right-aligned footer | none |
| `DropdownMenu` | Radix, API unchanged; glass panel, `shadow-md`, `radius-md` rows, `danger` item tone | none |

### New (23)

- Forms: Select, TextArea, Checkbox, RadioGroup, Toggle, Slider, Stepper, SearchField, FieldTile, SegmentedControl.
- Actions/overlays: IconButton, ButtonGroup, Tooltip.
- Navigation: MenuList, TopBar, UserMenu, PageHeader, Pagination.
- Content: DataTable, MediaCell, MediaCard, HeroHeader, NotificationItem.

Props follow `ds/components/index.d.ts`; behaviour and visuals follow `ds/components/bundle.js`, `bundle.css` and each `README.md`.

### Icons

String icon names (`icon="download"`) resolve through a name→lucide-react map; icons render at 20px with `strokeWidth={1.5}`. `ReactNode` icons also accepted.

### App migration

- `Root.tsx` header → `TopBar` (brand, nav items, theme `IconButton`, `UserMenu` with email and Sign out); page on `bd-backdrop`.
- 8 native `<select>` → `Select`; `<textarea>` → `TextArea`; Audit `<table>` → `DataTable`.
- Amber warning boxes in `LogFormDialog`/`LogRow` → DS `danger` / `Tag` tokens.
- Unused components (MediaCard, HeroHeader, NotificationItem, Slider, Stepper, FieldTile, …) appear in `KitchenSink` only. No screen redesign.

### Layout

One kebab-case file per component in `packages/ui/src/components/`; `index.ts` exports all.

## 3. Testing and verification

- Rewrite the 7 existing tests for the new APIs (`badge.test` → `tag.test`, `input.test` → `text-field.test`).
- One small test per component with logic: field error → `aria-invalid` + hint replaced; SegmentedControl/RadioGroup `onChange`; Toggle/Checkbox toggling; Stepper clamping; Pagination window; DataTable sort, selection + bulk actions, empty state; IconButton accessible name; MenuList expand + select; `useTheme` persistence and unset default.
- No tests for presentational-only components.
- Update app tests whose queries depend on changed markup.
- Before the PR, from `frontend/`: `npm run typecheck`, `test`, `lint`, `fmt:check`. Then run the app and screenshot main screens + KitchenSink in light and dark against `ds/preview`.

Accessibility kept: 2px `focus-ring` + 2px offset on all interactive elements, errors never colour-only, reduced motion respected, IconButton requires `label`.
