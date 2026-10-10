# Home page refresh (shpyrd.io reference) — design

Date: 2026-10-10 · Branch: `feat/home-page` · Scope: home page only (signed-in app unchanged).

## Animated background: `home/DotWave.tsx`
- 2D `<canvas>`, no new dependency. ~120×40 dot grid, height `sin(x·f + t)·cos(z·g + t)`, perspective-projected toward a horizon; nearer dots larger and brighter.
- Colours from DS tokens (`--chart-1` → `--chart-3`) read via `getComputedStyle`; re-read when `<html data-theme>` changes (MutationObserver).
- Absolutely positioned behind the hero, `pointer-events-none`, `aria-hidden`, mask `linear-gradient(to bottom, transparent, black 15%, black 70%, transparent)`.
- DPR-aware, ResizeObserver; rAF pauses when off-screen (IntersectionObserver) or tab hidden; `prefers-reduced-motion` → one static frame. Returns early when `getContext("2d")` is null (jsdom).

## Hero (two columns desktop, stacked mobile, `min-h-dvh`)
- Left: eyebrow Tag "Your year, in writing"; two-tone h1 "You did the work." / "We keep the receipts." (second line accent gradient); description; gradient **Sign in** → `/sign-in` + glass "See how it works" (scrolls to `#features`).
- Right: static glass app-window mockup (traffic dots, `brag document · job 1`) containing the log preview and Telegram bubble.
- DotWave behind both columns at the bottom.

## Features: bento grid (`#features`)
- Eyebrow "Everything review season needs", centred h2 "From a quick note to a review-ready story", subtitle.
- 3-column grid of glass cards: icon + h3 title on one line, short text, preview panel on a faint grid-line pattern (repeating-linear-gradient on `--container-divider`):
  - Log every win (wide) — log card + bubble
  - See your impact (tall) — stat tiles + bars
  - Share and export — share rows + Export PDF
  - Telegram bot (small) — mono `/log …` line
  - Audit trail (small) — two mono activity lines
- Closing CTA stays. Previews stay `aria-hidden` + `inert`.

## Tests
`Home.test`: h1 is the headline; "Sign in" link → `/sign-in`; h2 is the bento title; the five h3 card titles render. DotWave needs no test (no canvas in jsdom; it bails out).
