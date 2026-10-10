# Home page — design

Date: 2026-10-10
Branch: `feat/home-page` (off `feat/ui-design-system`; PR to `main` after #21)

## Goal

A public home page for signed-out visitors: a full-viewport hero with a sign-in button, then three sections describing what the system does.

## Decisions

- **Route:** `/` shows Home to signed-out visitors and the existing Root + Documents to signed-in users. URLs don't change. Other protected routes keep redirecting to `/sign-in`.
- **Visuals:** live, non-interactive previews built from `@bragdoc/ui` components with sample data (`aria-hidden`, `inert`). No image assets, theme-aware, no real user data.
- **Structure:** `routes/Home.tsx` plus small preview components in `src/home/`. Nothing new in `@bragdoc/ui`.

## Sections

1. **Hero** (`min-h-dvh`, centred): `HeroHeader` title "Brag Document" (text gradient), subtitle "Log what you did and why it mattered, from the web or Telegram. Share it with your manager and export a review-ready PDF.", one `Button variant="gradient" size="lg"` "Sign in" with chevron → `/sign-in`, scroll hint at the bottom.
2. **Log every win** — web or Telegram, with impact, status and tags. Preview: log row (impact Tag, status Tag, #tags) and a Telegram-style message bubble.
3. **See your impact** — dashboard: logs per month, top tags, coverage. Preview: stat tiles and a div bar chart (current month `chart-1`, others `chart-muted`).
4. **Share and export** — invite a manager as viewer or editor; export a PDF for review season. Preview: share row (MediaCell + role Tag), "Export PDF" Button, gradient document thumbnail.

Feature sections: two columns on desktop (preview side alternates), stacked on mobile; text = `type-caption` eyebrow, `type-title-1` heading, body. Closing line under section 4 repeats the CTA. Copy follows DS rules (sentence case, no exclamation marks, "you").

## Testing

- `Home.test.tsx`: signed out at `/` → heading "Brag Document" and a "Sign in" link to `/sign-in`; signed in at `/` → documents page.
- Root test "unauthenticated visitor is sent to sign-in" moves from `/` to a protected path (`/settings`).
