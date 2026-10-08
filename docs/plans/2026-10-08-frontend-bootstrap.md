# Frontend Bootstrap Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create `frontend/` as an npm workspace with a `@bragdoc/ui` design-system package and a `@bragdoc/app` React SPA, with lint, format, tests, Docker, and CI, per `docs/plans/2026-10-08-frontend-bootstrap-design.md`.

**Architecture:** npm workspaces at `frontend/`. `packages/ui` holds Tailwind v4 tokens (shadcn CSS variables) and shadcn-style components (Radix, cva, tailwind-merge) exported as source TSX. `packages/app` is a Vite + React 19 SPA using React Router in data mode with lazy routes, and imports `@bragdoc/ui` through the workspace symlink. One Tailwind pass runs in `app` and scans `ui/src` via `@source`. nginx serves the static build in Docker.

**Tech Stack:** Node 24, npm workspaces, Vite, React 19, TypeScript, Tailwind CSS v4 (`@tailwindcss/vite`), shadcn CLI, `radix-ui`, `class-variance-authority`, `tailwind-merge`, `clsx`, `lucide-react`, React Router (`react-router`), Vitest, Testing Library, jsdom, ESLint flat config, Prettier + `prettier-plugin-tailwindcss`, nginx, GitHub Actions.

**Conventions for every task:**
- Work on branch `feat/frontend-bootstrap` (already created). All paths are relative to the repo root unless stated.
- Install with `npm install <pkg>` so the lockfile pins versions; do not hand-edit versions into `package.json`. Latest at planning time: vite 8, react 19.3, react-router 8, tailwindcss 4.3, vitest 5, eslint 10, typescript 7, shadcn CLI 4.
- Run commands from `frontend/` unless stated.
- Commit message trailer on every commit:
  `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`
- If a library API differs from the snippet in this plan, read its docs (context7) and follow the docs; the test in each task is the contract, the snippet is a starting point.

---

### Task 1: Workspace root

**Files:**
- Create: `frontend/package.json`, `frontend/.nvmrc`, `frontend/.gitignore`, `frontend/packages/ui/package.json`, `frontend/packages/app/package.json`

**Step 1: Create the root manifest**

`frontend/package.json`:
```json
{
  "name": "bragdoc-frontend",
  "private": true,
  "type": "module",
  "workspaces": ["packages/*"],
  "engines": { "node": ">=24" },
  "scripts": {
    "dev": "npm run dev -w @bragdoc/app",
    "build": "npm run build -w @bragdoc/app",
    "typecheck": "npm run typecheck -ws --if-present",
    "test": "npm run test -ws --if-present",
    "lint": "eslint .",
    "fmt": "prettier --write .",
    "fmt:check": "prettier --check ."
  }
}
```

`frontend/.nvmrc`:
```
24
```

`frontend/.gitignore`:
```
node_modules/
dist/
.env
.env.*.local
```

**Step 2: Create the two package manifests**

`frontend/packages/ui/package.json`:
```json
{
  "name": "@bragdoc/ui",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "imports": {
    "#components/*": "./src/components/*.tsx",
    "#lib/*": "./src/lib/*.ts"
  },
  "exports": {
    ".": "./src/index.ts",
    "./styles.css": "./src/styles.css",
    "./components/*": "./src/components/*.tsx",
    "./lib/*": "./src/lib/*.ts"
  },
  "scripts": {
    "typecheck": "tsc --noEmit -p tsconfig.json",
    "test": "vitest run"
  }
}
```

`frontend/packages/app/package.json`:
```json
{
  "name": "@bragdoc/app",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview",
    "typecheck": "tsc --noEmit -p tsconfig.json",
    "test": "vitest run"
  }
}
```

**Step 3: Verify the workspace resolves**

Run: `cd frontend && npm install && npm ls --workspaces --depth 0`
Expected: both `@bragdoc/ui` and `@bragdoc/app` listed, no errors, `package-lock.json` created.

**Step 4: Commit**

```bash
git add frontend
git commit -m "feat(frontend): npm workspace with ui and app packages"
```

---

### Task 2: Shared dependencies, TypeScript, Vite, and Tailwind in `app`

**Files:**
- Create: `frontend/tsconfig.base.json`, `frontend/packages/app/tsconfig.json`, `frontend/packages/app/vite.config.ts`, `frontend/packages/app/index.html`, `frontend/packages/app/src/main.tsx`, `frontend/packages/app/src/index.css`, `frontend/packages/app/src/vite-env.d.ts`

**Step 1: Install runtime and build deps**

From `frontend/`:
```bash
npm install -w @bragdoc/app react react-dom react-router
npm install -w @bragdoc/app -D vite @vitejs/plugin-react tailwindcss @tailwindcss/vite typescript @types/react @types/react-dom @types/node
npm install -w @bragdoc/ui react react-dom
npm install -w @bragdoc/ui -D typescript @types/react @types/react-dom
```
(`react`/`react-dom` in `ui` as regular deps keeps one hoisted copy; npm dedupes them.)

**Step 2: Base tsconfig**

`frontend/tsconfig.base.json`:
```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2023", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "verbatimModuleSyntax": true,
    "isolatedModules": true,
    "skipLibCheck": true,
    "types": ["vite/client"]
  }
}
```

`frontend/packages/app/tsconfig.json`:
```json
{
  "extends": "../../tsconfig.base.json",
  "compilerOptions": { "types": ["vite/client", "vitest/globals", "@testing-library/jest-dom/vitest"] },
  "include": ["src", "vite.config.ts"]
}
```
(The `vitest`/`jest-dom` types are installed in Task 5; `tsc` will complain until then. That is expected; Task 5 clears it. If you prefer, add those two entries in Task 5.)

**Step 3: Vite config**

`frontend/packages/app/vite.config.ts`:
```ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: { port: 5173 },
});
```

**Step 4: Entry files**

`frontend/packages/app/index.html`:
```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Brag Document</title>
  </head>
  <body class="bg-background text-foreground">
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

`frontend/packages/app/src/index.css`:
```css
@import "@bragdoc/ui/styles.css";
@source "../../ui/src";
```

`frontend/packages/app/src/vite-env.d.ts`:
```ts
/// <reference types="vite/client" />
```

`frontend/packages/app/src/main.tsx` (temporary, replaced in Task 6):
```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <h1 className="p-4 text-2xl font-bold">Brag Document</h1>
  </StrictMode>,
);
```

**Step 5: Placeholder styles in `ui` so the import resolves**

`frontend/packages/ui/src/styles.css`:
```css
@import "tailwindcss";
```
(Replaced by the real tokens in Task 3.)

**Step 6: Verify build**

Run: `cd frontend && npm run build`
Expected: `packages/app/dist/index.html` and `dist/assets/*.css` exist, no errors.

**Step 7: Commit**

```bash
git add frontend
git commit -m "feat(frontend): vite, react, tailwind wiring for app"
```

---

### Task 3: Design tokens and shadcn init in `ui`

**Files:**
- Create: `frontend/packages/ui/components.json`, `frontend/packages/ui/tsconfig.json`, `frontend/packages/ui/src/lib/utils.ts`, `frontend/packages/ui/src/index.ts`
- Modify: `frontend/packages/ui/src/styles.css`

**Step 1: tsconfig for ui**

`frontend/packages/ui/tsconfig.json`:
```json
{
  "extends": "../../tsconfig.base.json",
  "compilerOptions": {
    "types": ["vitest/globals", "@testing-library/jest-dom/vitest"],
    "paths": { "#components/*": ["./src/components/*"], "#lib/*": ["./src/lib/*"] }
  },
  "include": ["src", "vite.config.ts"]
}
```

**Step 2: Install component deps**

From `frontend/`:
```bash
npm install -w @bragdoc/ui radix-ui class-variance-authority clsx tailwind-merge lucide-react tw-animate-css
```

**Step 3: shadcn config**

`frontend/packages/ui/components.json`:
```json
{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "new-york",
  "rsc": false,
  "tsx": true,
  "tailwind": {
    "config": "",
    "css": "src/styles.css",
    "baseColor": "neutral",
    "cssVariables": true
  },
  "iconLibrary": "lucide",
  "aliases": {
    "components": "#components",
    "ui": "#components",
    "lib": "#lib",
    "hooks": "#hooks",
    "utils": "#lib/utils"
  }
}
```
If `npx shadcn@latest add` (Task 4) rejects the `style` value, run `npx shadcn@latest init` inside `packages/ui`, accept the defaults it offers, and keep the aliases above.

**Step 4: `cn` helper and barrel**

`frontend/packages/ui/src/lib/utils.ts`:
```ts
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```

`frontend/packages/ui/src/index.ts`:
```ts
export { cn } from "#lib/utils";
```

**Step 5: Tokens**

Replace `frontend/packages/ui/src/styles.css` with the shadcn Tailwind v4 neutral theme. Get the canonical content by running, from `packages/ui`, `npx shadcn@latest init` and letting it write `src/styles.css` (preferred), or copy the "Manual installation, Tailwind v4" CSS from the shadcn docs. The result must contain:

```css
@import "tailwindcss";
@import "tw-animate-css";

@custom-variant dark (&:is(.dark *));

:root { --background: ...; --foreground: ...; --primary: ...; --radius: 0.625rem; /* full neutral palette */ }
.dark { /* dark palette */ }

@theme inline {
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  /* ... all shadcn color tokens, radius scale ... */
}

@layer base {
  * { @apply border-border outline-ring/50; }
  body { @apply bg-background text-foreground; }
}
```

Then make dark mode follow the OS (design decision: no toggle). Change the custom variant line to:
```css
@custom-variant dark (&:where(.dark, .dark *));
```
and append, after the `.dark { ... }` block:
```css
@media (prefers-color-scheme: dark) {
  :root:not(.light) { /* copy of the .dark variable block */ }
}
```
Keep the `.dark` class block so a toggle can be added later without touching tokens.

**Step 6: Verify**

Run: `cd frontend && npm run build && npm run typecheck -w @bragdoc/ui`
Expected: build succeeds; the CSS bundle in `packages/app/dist/assets/` contains `--background:oklch(` (not `--color-background`: `@theme inline` substitutes the token variables at build time, only the `:root`/`.dark` source variables survive). (`typecheck` for `ui` may fail on missing vitest types until Task 5; that is acceptable here.)

**Step 7: Commit**

```bash
git add frontend
git commit -m "feat(ui): shadcn config, design tokens, cn helper"
```

---

### Task 4: Components: Button, Input, Card, Dialog

**Files:**
- Create: `frontend/packages/ui/src/components/button.tsx`, `input.tsx`, `card.tsx`, `dialog.tsx`
- Modify: `frontend/packages/ui/src/index.ts`

**Step 1: Add components with the CLI**

From `frontend/packages/ui`:
```bash
npx shadcn@latest add button input card dialog
```
Expected: four files in `src/components/`, imports using `#lib/utils`. If the CLI writes to a different folder, move the files to `src/components/` and fix imports to `#lib/utils`.

Check each file: imports must come from `radix-ui` (e.g. `import { Dialog as DialogPrimitive } from "radix-ui"`) or `@radix-ui/react-*`. If the generated code imports `@radix-ui/react-dialog` and it is not installed, run `npm install -w @bragdoc/ui @radix-ui/react-dialog @radix-ui/react-slot`.

**Step 2: Export from the barrel**

`frontend/packages/ui/src/index.ts`:
```ts
export { cn } from "#lib/utils";
export { Button, buttonVariants } from "#components/button";
export { Input } from "#components/input";
export { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "#components/card";
export {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from "#components/dialog";
```
Adjust the names to what the generated files actually export (`grep "^export" src/components/*.tsx`).

**Step 3: Verify**

Run: `cd frontend && npm run build`
Expected: success. Use a temporary `<Button>` in `app/src/main.tsx` if you want to eyeball it with `npm run dev`; revert before committing.

**Step 4: Commit**

```bash
git add frontend/packages/ui
git commit -m "feat(ui): button, input, card, dialog components"
```

---

### Task 5: Vitest in both packages, component tests

**Files:**
- Create: `frontend/packages/ui/vite.config.ts`, `frontend/packages/ui/src/test/setup.ts`, `frontend/packages/ui/src/components/button.test.tsx`, `input.test.tsx`, `card.test.tsx`, `dialog.test.tsx`
- Create: `frontend/packages/app/src/test/setup.ts`
- Modify: `frontend/packages/app/vite.config.ts`

**Step 1: Install test deps at the root (shared dev tooling)**

From `frontend/`:
```bash
npm install -D vitest jsdom @testing-library/react @testing-library/jest-dom @testing-library/user-event @vitejs/plugin-react @tailwindcss/vite vite
```

**Step 2: Vitest config for ui**

`frontend/packages/ui/vite.config.ts`:
```ts
/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
  },
});
```

`frontend/packages/ui/src/test/setup.ts` and `frontend/packages/app/src/test/setup.ts` (identical):
```ts
import "@testing-library/jest-dom/vitest";
```

Add the same `test` block to `frontend/packages/app/vite.config.ts` (keep the `tailwindcss()` plugin there).

**Step 3: Write the failing tests**

`frontend/packages/ui/src/components/button.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import { Button } from "#components/button";

test("renders children and applies the destructive variant", () => {
  render(<Button variant="destructive">Delete</Button>);
  const btn = screen.getByRole("button", { name: "Delete" });
  expect(btn).toBeInTheDocument();
  expect(btn.className).toContain("bg-destructive");
});
```

`input.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Input } from "#components/input";

test("accepts typed text", async () => {
  render(<Input aria-label="name" />);
  await userEvent.type(screen.getByLabelText("name"), "hello");
  expect(screen.getByLabelText("name")).toHaveValue("hello");
});
```

`card.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import { Card, CardContent, CardHeader, CardTitle } from "#components/card";

test("renders title and content", () => {
  render(
    <Card>
      <CardHeader><CardTitle>Title</CardTitle></CardHeader>
      <CardContent>Body</CardContent>
    </Card>,
  );
  expect(screen.getByText("Title")).toBeInTheDocument();
  expect(screen.getByText("Body")).toBeInTheDocument();
});
```

`dialog.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Dialog, DialogContent, DialogTitle, DialogTrigger } from "#components/dialog";

test("opens on trigger click", async () => {
  render(
    <Dialog>
      <DialogTrigger>Open</DialogTrigger>
      <DialogContent><DialogTitle>Hello</DialogTitle></DialogContent>
    </Dialog>,
  );
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Open" }));
  expect(screen.getByRole("dialog")).toBeInTheDocument();
});
```

**Step 4: Run tests, verify they fail or pass for the right reason**

Run: `cd frontend && npm run test -w @bragdoc/ui`
Expected: the four tests run. They should pass because Task 4 already shipped the components; if a test fails, the failure identifies a wrong export name or class, fix the component or the test accordingly. Radix Dialog needs `ResizeObserver` and `PointerEvent` in jsdom; if the dialog test fails on those, add to `src/test/setup.ts`:
```ts
globalThis.ResizeObserver ??= class { observe() {} unobserve() {} disconnect() {} } as never;
```

**Step 5: Typecheck both packages**

Run: `cd frontend && npm run typecheck`
Expected: PASS for `ui` and `app`.

**Step 6: Commit**

```bash
git add frontend
git commit -m "test(ui): vitest setup and component tests"
```

---

### Task 6: App shell, router, routes, kitchen sink, env

**Files:**
- Create: `frontend/packages/app/src/env.ts`, `src/router.tsx`, `src/routes/Root.tsx`, `src/routes/Home.tsx`, `src/routes/KitchenSink.tsx`, `src/routes/NotFound.tsx`, `src/routes/Root.test.tsx`, `frontend/packages/app/.env.example`
- Modify: `frontend/packages/app/src/main.tsx`, `frontend/packages/app/package.json` (add `"@bragdoc/ui": "*"` dependency)

**Step 1: Declare the workspace dependency**

From `frontend/`: `npm install -w @bragdoc/app @bragdoc/ui@*`
Expected: `packages/app/package.json` has `"@bragdoc/ui": "*"` and `node_modules/@bragdoc/ui` is a symlink.

**Step 2: Write the failing shell test**

`frontend/packages/app/src/routes/Root.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router";
import { routes } from "../router";

test("shell renders the app title and the home page", async () => {
  const router = createMemoryRouter(routes, { initialEntries: ["/"] });
  render(<RouterProvider router={router} />);
  expect(await screen.findByRole("heading", { name: "Brag Document" })).toBeInTheDocument();
  expect(await screen.findByText(/nothing here yet/i)).toBeInTheDocument();
});

test("unknown path renders not found", async () => {
  const router = createMemoryRouter(routes, { initialEntries: ["/nope"] });
  render(<RouterProvider router={router} />);
  expect(await screen.findByText(/not found/i)).toBeInTheDocument();
});
```

**Step 3: Run it, verify it fails**

Run: `cd frontend && npm run test -w @bragdoc/app`
Expected: FAIL, `../router` not found.

**Step 4: Implement**

`frontend/packages/app/src/env.ts`:
```ts
function required(name: keyof ImportMetaEnv): string {
  const v = import.meta.env[name];
  if (!v) throw new Error(`Missing environment variable ${name}`);
  return v;
}

export const env = {
  apiUrl: required("VITE_API_URL"),
  supabaseUrl: import.meta.env.VITE_SUPABASE_URL ?? "",
  supabaseAnonKey: import.meta.env.VITE_SUPABASE_ANON_KEY ?? "",
};
```
Add to `src/vite-env.d.ts`:
```ts
interface ImportMetaEnv {
  readonly VITE_API_URL: string;
  readonly VITE_SUPABASE_URL?: string;
  readonly VITE_SUPABASE_ANON_KEY?: string;
}
```

`frontend/packages/app/.env.example`:
```
VITE_API_URL=http://localhost:8080
VITE_SUPABASE_URL=http://127.0.0.1:54321
VITE_SUPABASE_ANON_KEY=
```
Also create `frontend/packages/app/.env` with the same content locally (gitignored) so `npm run dev` works, and add `VITE_API_URL=http://localhost:8080` to the test env via `test.env` in `vite.config.ts` or by not importing `env.ts` from any route in the skeleton (preferred: nothing imports `env.ts` yet; it exists for the first API consumer).

`frontend/packages/app/src/routes/Root.tsx`:
```tsx
import { Link, Outlet } from "react-router";

export function Component() {
  return (
    <div className="min-h-screen">
      <header className="border-b">
        <nav className="mx-auto flex max-w-5xl items-center gap-6 p-4">
          <h1 className="text-lg font-semibold">
            <Link to="/">Brag Document</Link>
          </h1>
          {import.meta.env.DEV && (
            <Link to="/kitchen-sink" className="text-muted-foreground text-sm">
              Kitchen sink
            </Link>
          )}
        </nav>
      </header>
      <main className="mx-auto max-w-5xl p-4">
        <Outlet />
      </main>
    </div>
  );
}
```

`src/routes/Home.tsx`:
```tsx
import { Card, CardContent, CardHeader, CardTitle } from "@bragdoc/ui";

export function Component() {
  return (
    <Card>
      <CardHeader><CardTitle>Welcome</CardTitle></CardHeader>
      <CardContent>Nothing here yet. Documents will appear here.</CardContent>
    </Card>
  );
}
```

`src/routes/NotFound.tsx`:
```tsx
export function Component() {
  return <p className="text-muted-foreground">Page not found.</p>;
}
```

`src/routes/KitchenSink.tsx`: render every exported component once in a grid (Button in each variant and size, Input, Card, Dialog with trigger). Headings per section. No logic.

`src/router.tsx`:
```tsx
import { createBrowserRouter, type RouteObject } from "react-router";

export const routes: RouteObject[] = [
  {
    path: "/",
    lazy: () => import("./routes/Root"),
    children: [
      { index: true, lazy: () => import("./routes/Home") },
      ...(import.meta.env.DEV ? [{ path: "kitchen-sink", lazy: () => import("./routes/KitchenSink") }] : []),
      { path: "*", lazy: () => import("./routes/NotFound") },
    ],
  },
];

export const router = createBrowserRouter(routes);
```
(`lazy` resolving to a module that exports `Component` is the documented React Router data-mode API; if v8 requires `.then(m => ({ Component: m.Component }))`, do that.)

`src/main.tsx`:
```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "react-router";
import { router } from "./router";
import "./index.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
);
```

**Step 5: Run tests, build, typecheck**

Run: `cd frontend && npm test && npm run typecheck && npm run build`
Expected: all PASS; `packages/app/dist/assets/` contains more than one JS chunk (lazy routes). `grep -l "kitchen" packages/app/dist/assets/*.js` returns nothing (dev-only route tree-shaken).

**Step 6: Commit**

```bash
git add frontend
git commit -m "feat(app): router, shell, home, kitchen sink, env"
```

---

### Task 7: ESLint and Prettier

**Files:**
- Create: `frontend/eslint.config.js`, `frontend/.prettierrc`, `frontend/.prettierignore`

**Step 1: Install**

From `frontend/`:
```bash
npm install -D eslint @eslint/js typescript-eslint eslint-plugin-react-hooks eslint-plugin-react-refresh eslint-config-prettier globals prettier prettier-plugin-tailwindcss
```

**Step 2: Config**

`frontend/eslint.config.js`:
```js
import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";
import prettier from "eslint-config-prettier";

export default tseslint.config(
  { ignores: ["**/dist", "**/node_modules"] },
  {
    files: ["**/*.{ts,tsx}"],
    extends: [js.configs.recommended, ...tseslint.configs.recommended, reactHooks.configs["recommended-latest"], reactRefresh.configs.vite],
    languageOptions: { ecmaVersion: 2022, globals: globals.browser },
  },
  prettier,
);
```
(Match the plugin config names to what each plugin's README exports for flat config at the installed version. The shadcn components export `buttonVariants` next to a component; if `react-refresh/only-export-components` fires on `packages/ui`, allow it with `{ files: ["packages/ui/**"], rules: { "react-refresh/only-export-components": "off" } }`.)

`frontend/.prettierrc`:
```json
{ "plugins": ["prettier-plugin-tailwindcss"], "tailwindStylesheet": "./packages/ui/src/styles.css" }
```

`frontend/.prettierignore`:
```
node_modules
dist
package-lock.json
```

**Step 3: Run and fix**

Run: `cd frontend && npm run fmt && npm run lint && npm run fmt:check`
Expected: lint exits 0 (fix any findings in source, do not disable rules without a reason), format check exits 0.

**Step 4: Commit**

```bash
git add frontend
git commit -m "chore(frontend): eslint and prettier"
```

---

### Task 8: Makefile, Dockerfile, nginx

**Files:**
- Create: `frontend/Makefile`, `frontend/Dockerfile`, `frontend/nginx.conf`, `frontend/.dockerignore`

**Step 1: Makefile**

`frontend/Makefile`:
```make
.PHONY: dev build test lint fmt typecheck docker

dev: ; npm run dev
build: ; npm run build
test: ; npm test
lint: ; npm run lint && npm run fmt:check
fmt: ; npm run fmt
typecheck: ; npm run typecheck
docker: ; docker build -t bragdoc-frontend .
```

**Step 2: Dockerfile**

`frontend/Dockerfile`:
```dockerfile
FROM node:24-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
COPY packages/ui/package.json packages/ui/
COPY packages/app/package.json packages/app/
RUN npm ci
COPY . .
ARG VITE_API_URL
ARG VITE_SUPABASE_URL
ARG VITE_SUPABASE_ANON_KEY
RUN npm run build

FROM nginx:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/packages/app/dist /usr/share/nginx/html
EXPOSE 80
```

`frontend/nginx.conf`:
```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    location /assets/ {
        add_header Cache-Control "public, max-age=31536000, immutable";
        try_files $uri =404;
    }

    location / {
        add_header Cache-Control "no-cache";
        try_files $uri /index.html;
    }
}
```

`frontend/.dockerignore`:
```
node_modules
**/node_modules
**/dist
**/.env
**/.env.*.local
```

**Step 3: Verify**

Run from repo root: `docker compose --profile app build frontend`
Expected: image builds. Then `docker run --rm -p 8081:80 bragdocument-frontend` (image name from compose output) and `curl -s localhost:8081/anything | grep -c '<div id="root">'` prints `1` (SPA fallback works). Stop the container.

**Step 4: Commit**

```bash
git add frontend
git commit -m "build(frontend): makefile, dockerfile, nginx"
```

---

### Task 9: CI workflow

**Files:**
- Create: `.github/workflows/frontend.yml`

**Step 1: Workflow**

```yaml
name: frontend
on:
  push:
    branches: [main]
    paths: ["frontend/**", ".github/workflows/frontend.yml"]
  pull_request:
    paths: ["frontend/**", ".github/workflows/frontend.yml"]
jobs:
  check:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: frontend
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version-file: frontend/.nvmrc
          cache: npm
          cache-dependency-path: frontend/package-lock.json
      - run: npm ci
      - run: npm run lint
      - run: npm run fmt:check
      - run: npm run typecheck
      - run: npm test
      - run: npm run build
        env:
          VITE_API_URL: http://localhost:8080
```

**Step 2: Verify locally**

Run the same commands in order from `frontend/` on a clean install: `rm -rf node_modules packages/*/node_modules && npm ci && npm run lint && npm run fmt:check && npm run typecheck && npm test && npm run build`
Expected: all exit 0.

**Step 3: Commit**

```bash
git add .github/workflows/frontend.yml
git commit -m "ci: frontend lint, typecheck, test, build"
```

---

### Task 10: Docs and PR

**Files:**
- Modify: `README.md` (add a "Frontend" section after "Backend"), `docs/README.md` (repository layout block: `frontend/` now has `packages/ui` and `packages/app`), `.gitignore` (replace the two `frontend/` lines with `frontend/**/node_modules/` and `frontend/**/dist/`)

**Step 1: README section**

```md
## Frontend

```sh
cd frontend
npm install
cp packages/app/.env.example packages/app/.env
make dev       # vite on :5173; /kitchen-sink shows the design system
make test      # vitest in ui and app
make lint      # eslint + prettier check
make build     # static bundle in packages/app/dist
```

Workspace: `packages/ui` is the design system (`@bragdoc/ui`, shadcn-style components on Tailwind v4), `packages/app` is the SPA. Add a component with `npx shadcn@latest add <name>` from `packages/ui`, then export it from `src/index.ts`. Design: `docs/plans/2026-10-08-frontend-bootstrap-design.md`.
```

**Step 2: Commit**

```bash
git add README.md docs/README.md .gitignore
git commit -m "docs: frontend run instructions and layout"
```

**Step 3: Final verification**

From repo root: `git status` clean; `cd frontend && npm run lint && npm run typecheck && npm test && npm run build` all pass; `docker compose config` exits 0.

**Step 4: Push and open the PR**

```bash
git push -u origin feat/frontend-bootstrap
gh pr create --base main --title "feat(frontend): bootstrap workspace with ui and app packages" --body "$(cat <<'EOF'
Implements ADR-0002 (React + Tailwind + Vite) and ADR-0003 (monorepo) for the frontend half. Design: docs/plans/2026-10-08-frontend-bootstrap-design.md.

- npm workspace at `frontend/`: `@bragdoc/ui` (Tailwind v4 tokens, shadcn-style Button/Input/Card/Dialog) and `@bragdoc/app` (Vite + React 19 + React Router, lazy routes, dev-only kitchen sink)
- Vitest + Testing Library in both packages, ESLint flat config, Prettier with the Tailwind plugin
- Multi-stage Dockerfile (node build, nginx serve with SPA fallback), already wired in docker-compose
- GitHub Actions workflow path-filtered to `frontend/**`

Deferred, each lands with its first consumer: Supabase client, API client, TanStack Query, Recharts, theme toggle.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
