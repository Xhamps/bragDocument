BragDoc UI is a glass interface: a coloured mood backdrop sets the atmosphere, translucent containers absorb it, and soft shadows lift them. Two themes share one set of names — **Light** (pastel wash, white glass) and **Dark** (deep navy, smoked glass). The first theme is the fallback.

## Content fundamentals

- Sentence case everywhere: "Browse templates", "Mark all as read", "Yes, go ahead". Title Case only for proper nouns.
- Buttons start with a verb and stay two or three words. A trailing chevron means "go somewhere"; no chevron means "do it here".
- Address the person as *you* ("Unlock full access to premium features"); the product never says *I*.
- Notifications read as *name + what they did*: "**Eva Solain** invited you to a chat", with a relative time ("5m ago", "1d ago").
- Uppercase is reserved for `caption` tags and eyebrows ("UI/UX DESIGN", "CREDIT CARD").
- No emoji, no exclamation marks.

## Visual foundations

### Backdrop and glass

- Every screen starts on `page` with the mood backdrop painted over it from `mood-1` (top), `mood-2` (lower left) and `mood-3` (lower right) as large blurred radial washes. Use the `bd-backdrop` class from `components/bundle.css`.
- A container is glass: `container-bg` fill, `backdrop-filter: blur(var(--blur-glass))`, a 1px `container-border`, `shadow-glass`, corner `radius-lg`. Use the `Card` component; never paint an opaque card on top of the backdrop.
- Cards carry a transparent gradient over the glass: by default a white `glass-sheen` catching light from the top-left; for a featured card, a faint `glass-tint-*` wash (blue, violet, rose, teal) via the Card `tint` prop. Tints are washes, never fills: text colours do not change on them.
- Rows inside a container are separated by `container-divider` hairlines or by nested glass at a smaller radius (`radius-md`), as in the notification list.
- Use `surface` only where nothing sits behind (a plain list in a settings page, a course lesson row in Light).

### Color

- Text: `fg-primary` for headings, names and values; `fg-secondary` for body copy; `fg-tertiary` for timestamps and labels only (4.7:1 on the light page; keep it off busy backdrops).
- Action: the accent blue comes in two tokens. `button` is the fill under a `button-fg` label (primary buttons, checked boxes, toggles, the current page; white on it is 5.3:1 Light, 4.8:1 Dark). `button-text` is the same accent used as text, icons and rings on page or glass (links, ghost and tinted buttons, the selected segment's label, sort arrows; 4.8:1 Light, 6.4:1 Dark). Hover and pressed use `button-hover`; disabled uses `button-inactive`.
- Both blues are slightly deeper than the kit's `#007aff` so every label passes AA; Dark keeps the kit's `#3395ff` for text, where it reads well.
- Data: series in order `chart-1`, `chart-2`, `chart-3`, `chart-4`; inactive bars and empty tracks `chart-muted`. Highlight the current period in `chart-1` and leave the rest muted.
- `success` marks completed items and always travels with a check shape.
- Gradients are illustration, not chrome: use the `gradient-*` tokens for hero art, thumbnails and the single Get Started call to action (`gradient-red-3`) or Learn (`gradient-learn`). The `gradient-angular-*` tokens are the Glow layer behind a floating card. Text gradients (`tg-primary-*`, `tg-secondary-*`) are for `display` and `title-1` text only, via `.bd-text-gradient` / `.bd-text-gradient-secondary`.

### Type

- One family, Inter (`--font-sans`), loaded from Google Fonts at 400/500/600/700. Code uses `--font-mono`.
- Scale: `display` → `title-1` → `title-2` → `title-3` for headings; `body` for paragraphs; `callout` for anything interactive; `footnote` for metadata; `caption` for uppercase tags; `code` for code.
- Headlines get negative tracking (built into the styles); never track body text.

### Spacing, radii, elevation

- Spacing steps are `space-1` (4) through `space-12` (48). Cards pad `space-6`; gaps between cards are `space-4`; page gutters `space-12`.
- `radius-md` (10px) is the default corner for buttons, inputs and rows; `radius-lg` for cards; `radius-xl` for app frames; `radius-pill` for search, segmented controls, toggles and avatars. A nested element's radius is one step smaller than its container's.
- Elevation is light, not darkness. The scale is `shadow-sm` (tags, resting rows) → `shadow-md` (hovered controls, selected segment) → `shadow-lg` (hovered cards, tooltips) → `shadow-xl` (dialogs, floating cards). Containers rest on `shadow-glass`, glass buttons on `shadow-button`; tracks and pressed states sink with `shadow-inset`. `shadow-glow` once per view, growing to `shadow-glow-strong` on hover.

### Motion

- Motion confirms, it never decorates. Hovers lift (buttons 1px, tiles 2px, interactive cards 4px) and raise one shadow step; presses sink back (scale 0.98, `shadow-inset`).
- Timing: `duration-fast` for colour and press, `duration-base` for hovers, toggles and checks, `duration-slow` for card lifts and sheen sweeps, `duration-enter` for entrances. Curves: `ease-standard` by default, `ease-out` for things arriving, `ease-spring` for small objects that settle (toggle thumbs, check marks, radio dots, icon presses).
- Entrances use `bd-enter` (fade and rise 12px), staggered 60ms per item, only on first load, never on every re-render.
- Trailing arrows and chevrons nudge 2px forward on hover; the gradient button sweeps a sheen; unread dots pulse softly.
- Everything collapses to instant under `prefers-reduced-motion`.

### States

- Hover: glass buttons brighten one step (more opaque `container-bg`); color buttons move to `button-hover`.
- Selected: in a segmented control the chosen option becomes a raised glass pill; in a menu the row gets a glass fill and a 2px `button` bar on its leading edge.
- Focus: a solid 2px `focus-ring` outline with a 2px offset, on every interactive element, in every theme.
- Disabled: `button-inactive` fill or border, no shadow, no pointer.

### Forms

- Inputs are glass: `container-bg`, a 1px `container-border`, `radius-md`, 44px tall. Selects and search are pills (`radius-pill`).
- Labels sit above the control in `footnote` weight 500, `fg-secondary`; hints below in `footnote`, `fg-secondary`; errors replace the hint in `danger` and turn the border `danger`. Never show an error by colour alone.
- Selection marks are round: checkboxes fill with `button` and a `button-fg` check; radios show a `button` ring and dot; the label of a chosen option moves to `fg-primary`.
- Pick the control by the choice: one of 2–4 that switches a view → SegmentedControl; one of 2–5 in a form → RadioGroup; more → Select; independent yes/no → Checkbox; applies instantly → Toggle; a small count → Stepper; an approximate amount → Slider; a picked value shown large (place, date, class) → FieldTile.
- A form ends in one primary button, centred under a search form or right-aligned in a dialog.

## Iconography

- Thin outline icons, 1.5px stroke, 20px box, drawn in `currentColor` so they take the text color (house, sliders, search, chevron, bell, calendar, pin).
- The source kit's icon set was not attached; the components draw their few glyphs (chevron, search, check) as inline SVG in that style. Add the real set to `assets/Icons/` when available.
- No emoji and no filled multicolour icons in chrome. Avatars are circular photos at `radius-pill`.

## Logo

No logo was supplied: set "BragDoc UI" in `title-3` at weight 600 in `fg-primary`.

---

## Using these files

- `tokens.json` is the source of truth; `tokens.css` is generated from it (CSS custom properties per theme on `:root` / `[data-theme="dark"]`, plus one class per type style).
- Load, in order: `tokens.css`, `components/bundle.css`, React 18 (`components/lib/`), then `components/bundle.js`, which defines `window.BragDocUI` (Button, Card, DataTable, TopBar…). Types and props are in `components/index.d.ts`; each component's guidelines are in `components/<Name>/README.md`.
- Set `data-theme="light"` or `data-theme="dark"` on `<html>`.
- Open `preview/index.html` in a browser to see every component live, with a light/dark switch. Each `preview/<Name>.html` also opens on its own.
- Images live in `assets/Images/`.
