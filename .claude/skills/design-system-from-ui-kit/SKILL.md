---
name: "design-system-from-ui-kit"
description: "Build or extend a Design System artifact from UI kit screenshots or brand files (sampled tokens, hand-written React components, live previews, contrast checks) and export it to a local folder."
---

# Design system from a UI kit

Use when the user attaches UI kit screenshots, spec sheets or brand files and wants a design system artifact built from them, wants more components added to one, or wants it exported into a code project.

## 0. Before building

- Read the Design System artifact (Artifact `read`) to get the type's instructions and the publish url (`https://claude.ai/artifact/<id>`). Read `artifact-type/reference/format.md`, `cover.md`, `craft.md` and `demo.json` once.
- Check connected folders for more source files (spec sheets like "Color Styles Light/Dark", "Gradients", "Text"). Stage them and look at every image.
- Ask once with AskUserQuestion: the system's **name** (never reuse a third-party kit's brand name or logo unless the user says so), **font** handling (Google-hosted family vs. files to drop), and **scope** (tokens + core components vs. tokens only).

## 1. Tokens: sample, don't guess

- Sample colors with PIL at swatch centres. Spec-sheet PNGs often have alpha and sit on a translucent panel (check a corner pixel, e.g. `#ffffff1a` or `#0000001a`). Unmix: `a_c = (a_out − a_bg)/(1 − a_bg)` and `C = (C_out·a_out − C_bg·a_bg·(1 − a_c))/a_c`, then round to clean values like `rgba(0, 0, 0, 0.7)`.
- When two sources disagree (a spec PNG vs. a rendered JPG), map them to themes (light/dark) rather than averaging.
- `tokens.json` shape: every family except `type` is `{"tokens":[{name,value,usage}]}`. Colors and shadows are themed `{light, dark}` with the primary theme first. Use plain CSS values only (no `var()` or named colors, except aliases `{token}`). Extra families are fine: `gradient`, `blur`, `duration`, `easing`.
- Every token gets a usage note naming where it is used and the grounds a text color reads on.
- Gradients: put literal `linear-gradient`/`conic-gradient` strings in a `gradient` family. Themed gradient stops (text gradients) become color tokens composed in `bundle.css`.
- Shadows: elevation scale `shadow-sm → xl`, plus `shadow-inset`, glass, button, glow and glow-strong. Motion: `duration-fast/base/slow/enter` and `ease-standard/out/spring`.

## 2. Accessibility pass (do it, then report)

- Compute WCAG contrast with a small Python helper for every text/fill pair in every theme.
- Keep failing source values exact at first, and say so in the token note and the README.
- When the user asks to fix a pair, propose figures first and wait for OK.
- If one color can't serve both fill-with-white-label and text-on-dark-page, split it into two tokens: `button` (fills) and `button-text` (text, icons, rings). Re-point aliases such as `focus-ring` and `button-selected`.

## 3. Components

- One classic `components/bundle.js` IIFE using `React.createElement` (no JSX, imports or network). Its line-1 header `/* @ds-bundle: {"format":4,"namespace":"X","components":[...]} */` lists every component. It ends with `window.X = Object.assign(window.X || {}, {...})`.
- Rules in `components/bundle.css` on `var(--token)` only, with a `bd-`-style prefix. Add `@media (prefers-reduced-motion: reduce)`.
- Shared helpers worth reusing: `Icon` (stroke-1.5 inline paths, built-in names), `useId`, `omit`, `initials`.
- `useIndicator(value, deps)` measures `[data-bd-on="true"]` and drives a sliding pill or bar (segmented controls, menus, nav). It uses a ResizeObserver and `document.fonts.ready`.
- Controlled/uncontrolled pattern: `value`/`defaultValue`/`onChange`.
- Overlays (tooltip, user menu, flyout sub menu, nav dropdown): opaque `surface` panel, `shadow-xl`, pointer arrow `::before`, hover bridge `::after`, 160 ms close delay. Escape and arrow keys work, collapsed panels are `inert`, and spring scale-in. Raise the parent's z-index while open (`:has(...)`).
- Each component gets `components/<Name>/README.md` (first sentence = summary; what the consumer provides; do/don't) and `components/<Name>/preview.html` (line 1 `<!-- @dsCard group="..." height=N -->`, rendering via `window.X`, placed on the `bd-backdrop`).
- Keep `components/index.d.ts` props current; it documents the API.
- The cover goes last (`components/Cover/preview.html`) per `cover.md`: colour blocks + one pattern + the name, every fill bound to a token.

## 4. Verify locally before every publish

- Generate a test `tokens.css` from `tokens.json`. Copy `bundle.*` and the React libs from `demo.json` into a test folder.
- Inject the head links into each preview and screenshot with Playwright (`NODE_PATH=$(npm root -g)`), in light and dark, catching `pageerror`.
- For interactions (hover dropdowns, clicks, sliding indicators) script `hover`/`click`/`keyboard.press` and capture mid- and end-frames.
- Look at every screenshot. Fix overlap, clipping, z-index and contrast before publishing.

## 5. Publish (one call per change)

- Images: upload with `publish` + `asset:true` + `file_paths`. Record each under `assetGroups.<Group>.files` (`name`, `blob` id, `size`, `type`) and in `groups`. Add an `assets/<Group>/README.md`. Previews reference `/_blob/<id>`. Crop artwork free of UI text and logos.
- Right before publishing, `read` `project/design-system.json`. Keep every key and `createdOnFiles`, and set `lastChange` `{by, at, via:"Cowork", note}`.
- Publish with `url`, `root`, `file_path` = the index, and `files` = only the changed `project/...` paths. `.d.ts` needs `{"from":..., "contentType":"text/plain"}`.
- Never write `tokens.css`, `manifest.json` or `api/` into the artifact.

## 6. Export to a local project folder

- Request access to exactly the target folder (`device_request_folder_access`).
- Build an export: `project/*` minus `design-system.json`, plus a generated `tokens.css` (themes on `:root`/`[data-theme]`, one class per type style). Also include the React 18 libs in `components/lib/` and images in `assets/Images/` with previews' `/_blob/` urls rewritten to relative paths.
- Add standalone `preview/<Name>.html` pages (head links + `?theme=`) and a `preview/index.html` gallery with a light/dark switch. Append a "Using these files" section to README.
- Check a few exported pages in Playwright. Zip, copy to `/mnt/user-data/outputs/`, `device_commit_files`, then `unzip` with `device_bash`. Deletion is usually blocked, so tell the user the zip can be removed.

## Reporting

Keep replies short:
- what was built and from which sources,
- what was estimated (type sizes, shadows),
- the contrast pairs that fail and were kept,
- what remains (logo, icon set, fonts).

Never paste the artifact URL.