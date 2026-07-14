# AGENTS.md — AdGuard Home client_v2 (SolidJS Frontend)

## Table of Contents

- [AGENTS.md — AdGuard Home client\_v2 (SolidJS Frontend)](#agentsmd--adguard-home-client_v2-solidjs-frontend)
  - [Table of Contents](#table-of-contents)
- [Project Overview](#project-overview)
- [Technical Context](#technical-context)
- [Project Structure](#project-structure)
- [Build And Test Commands](#build-and-test-commands)
- [Contribution Instructions](#contribution-instructions)
- [Code Guidelines](#code-guidelines)
  - [System Design](#system-design)
  - [Architecture](#architecture)
  - [Code Quality](#code-quality)
  - [Testing](#testing)
  - [Dependency Management](#dependency-management)
  - [Configuration \& Documentation](#configuration--documentation)
  - [Markdown Formatting](#markdown-formatting)
  - [Other](#other)
    - [Accessibility](#accessibility)
    - [Translations](#translations)

# Project Overview

`client_v2` is the next-generation web UI for **AdGuard Home**, written in
**SolidJS** and TypeScript. It is a multi-entry single-page application that
runs in the browser and communicates with the AdGuard Home backend over its
HTTP `/control` JSON API.

The app ships four separate HTML entry points that share the same codebase,
styles, and theme system:

- `main` — the dashboard / control panel (HashRouter SPA)
- `install` — the first-run setup wizard
- `login` — the sign-in page
- `forgot_password` — the password-reset flow

It replaces the legacy React-based client (`client/`) and is built with
Webpack, styled with PostCSS + CSS Modules, and tested with Vitest (unit) and
Playwright (e2e).

# Technical Context

- **Language/Version**: TypeScript 5.4+ (`strict: false`, `noImplicitAny: true`)
- **UI Framework**: SolidJS 1.9 (`solid-js`, `@solidjs/router` 0.15 with
  `HashRouter`)
- **UI Primitives**: `@ark-ui/solid` (headless components),
  `@modular-forms/solid` (form state)
- **Build Tooling**: Webpack 5 (`webpack.common.js` / `webpack.dev.js` /
  `webpack.prod.js`), Babel (`babel-preset-solid`), PostCSS
- **Styling**: PostCSS (`postcss-import`, `postcss-nested`, `autoprefixer`) +
  CSS Modules (`.module.pcss`); design tokens via CSS custom properties
- **State**: SolidJS `createStore` module-scoped stores (no Redux, no Context
  providers)
- **HTTP**: Native `fetch` wrapped in a single `Api` class
- **i18n**: `@adguard/translate` with 13 locale JSON files in `src/__locales/`
- **Testing**: Vitest 4 + `@solidjs/testing-library` (unit);
  Playwright 1.56 (e2e against a real backend)
- **Linting/Formatting**: ESLint 8 (`@typescript-eslint`, `eslint-plugin-solid`,
  `eslint-plugin-prettier`), Prettier 3
- **Storage**: Browser `localStorage` (language, theme); no database
- **Target Platform**: Modern evergreen browsers (see `browserslist` in
  `package.json`); output is static assets served by the Go backend
- **Project Type**: Website (single, browser-resident SPA)
- **Performance Goals**: N/A (not formally documented)
- **Constraints**: Output is embedded into the Go binary's `build/static/`;
  the dev server proxies to a running AdGuard Home instance
- **Scale/Scope**: Single-user admin UI (one operator per AdGuard Home
  installation)

# Project Structure

```text
client_v2/
├── src/
│   ├── index.tsx                  # Main app entry (renders <App/>)
│   ├── index.pcss                 # Global CSS (vars, reset)
│   ├── initialState.ts            # Domain model types + initial form state
│   ├── api/                       # Single Api class — all HTTP calls
│   ├── common/
│   │   ├── controls/              # Input primitives (Checkbox, Input, Select, Switch, …)
│   │   ├── ui/                    # Higher-level UI (Button, Dialog, Table, Tabs, Sidebar, …)
│   │   ├── intl/                  # i18n setup (@adguard/translate)
│   │   └── styles/                # Theme tokens (vars.css, colors/light|dark|adg.css)
│   ├── components/                # Feature/page components (Dashboard, Clients, QueryLog, …)
│   │   ├── App/                   # Root component — HashRouter + layout + <Route>s
│   │   └── Routes/Paths.ts        # Route path constants
│   ├── helpers/                   # Pure utilities, validators, theme helpers
│   ├── hooks/                     # Shared SolidJS hooks
│   ├── lib/                       # Theme aggregation, misc utils
│   ├── store/types.ts             # Legacy type re-exports (post-Redux)
│   ├── stores/                    # SolidJS createStore modules — one per domain
│   ├── types/                     # css-modules.d.ts ambient declarations
│   ├── login/                     # Standalone login entry
│   ├── install/                   # Standalone install-wizard entry
│   ├── forgot_password/           # Standalone password-reset entry
│   ├── __locales/                 # Translation JSON (en, ru, de, …)
│   └── __tests__/                 # Vitest unit/integration tests
├── tests/e2e/                     # Playwright e2e specs
├── scripts/                       # check-translations.js, translation-audit.js
├── webpack.common.js              # 4 entry points, loaders, path aliases
├── webpack.dev.js                 # Dev server + backend proxy
├── webpack.prod.js                # Production build
├── tsconfig.json                  # Path aliases: panel/* → src/*, Twosky → ../.twosky.json
├── vitest.config.ts               # jsdom + vite-plugin-solid
├── playwright.config.ts           # e2e config (base URL :3000)
├── postcss.config.js              # postcss-import + nested + autoprefixer
├── package.json                   # Scripts + dependencies
└── constants.js                   # BUILD_ENVS, BASE_URL = 'control'
```

# Build And Test Commands

All commands are run from the `client_v2/` directory.

| Task                                 | Command                        |
| ------------------------------------ | ------------------------------ |
| Dev server (proxies to backend)      | `npm run dev`                  |
| Production build                     | `npm run build-prod`           |
| Dev build (no server)                | `npm run build-dev`            |
| Watch build                          | `npm run watch`                |
| Type check                           | `npm run typecheck`            |
| Lint                                 | `npm run lint`                 |
| Lint + format fix                    | `npm run lint:fix`             |
| Unit tests (single run)              | `npm run test`                 |
| Unit tests (watch)                   | `npm run test:watch`           |
| E2e tests                            | `npm run test:e2e`             |
| E2e interactive UI                   | `npm run test:e2e:interactive` |
| Translation check                    | `npm run translations:check`   |
| Full check (lint + typecheck + test) | `npm run check`                |

# Contribution Instructions

- You MUST verify your work with the linter, formatter, and type checker.

    Use the following commands:
    - `npm run typecheck` to check for type errors
    - `npm run lint` to run the linter
    - `npm run lint:fix` to fix linting and formatting issues automatically

- You MUST update the unit tests for changed code. New stores, helpers, and
  components should have corresponding tests under `src/__tests__/`.

- You MUST run tests with `npm run test` to verify that your changes do not
  break existing functionality. Use `npm run check` for the full gate
  (lint + typecheck + test).

- When making changes to the project structure, ensure the Project Structure
  section in `AGENTS.md` is updated and remains valid.

- If the prompt essentially asks you to refactor or improve existing code,
  check if you can phrase it as a code guideline. If it is possible, add it to
  the relevant Code Guidelines section in `AGENTS.md`.

- After completing the task you MUST verify that the code you have written
  follows the Code Guidelines in this file.

# Code Guidelines

## System Design

`client_v2` is a browser-resident SPA — it runs entirely client-side and has
no server process of its own. Design for the browser environment:

- **Stateless client, stateful backend.** The UI holds no durable state; all
  configuration and data persist in the AdGuard Home backend via the `/control`
  HTTP API. Treat the browser as a thin view layer.
- **Single source of HTTP.** All backend calls go through the singleton
  `apiClient` in `src/api/Api.ts`. Do not call `fetch` directly from components
  or stores — add a typed method to the `Api` class instead.
- **No shared server memory.** Multiple browser tabs / reloads are independent;
  never rely on in-memory module state surviving a reload. Re-fetch data in
  component `onMount` rather than assuming a store is already populated.
- **Disposability.** The app boots in the browser on demand; clean up
  subscriptions and timers in `onCleanup` to avoid leaks across route changes.
- **HashRouter routing.** The main app uses `HashRouter`; route paths are
  defined as constants in `src/components/Routes/Paths.ts`. The three
  standalone pages (`install`, `login`, `forgot_password`) are separate entry
  points, not routes inside the main app.
- **Theme via attributes, not JS state.** Theming is driven by the
  `data-theme` attribute on `<html>` and CSS custom properties; switch themes
  through `setUITheme` in `src/helpers/helpers.tsx`, not by toggling
  component-level classes.
- **Static asset output.** The production build emits static files into
  `../build/static/` which the Go binary embeds. Do not introduce runtime
  server requirements (Node, databases, etc.) — the built output must be
  plain static files.

## Architecture

Universal design principles the codebase follows:

- **Separation of Concerns** — UI components, state stores, API access, and
  pure helpers live in distinct directories.
- **Single Responsibility Principle** — each store module owns one domain;
  each component one page or control.
- **Dependency Direction** — dependencies point downward: components → stores
  → api → backend. Stores never import components; api never imports stores.
- **Explicit Boundaries** — modules interact through named exports; no
  reaching into another module's internals.
- **Data Flow Clarity** — user action → store action function → `apiClient`
  → `setState` → reactive UI update. Data moves in one predictable path.
- **Minimize Coupling, Maximize Cohesion** — stores are self-contained and
  imported directly (no Context providers); components depend on narrow store
  exports.
- **Make Invalid States Impossible** — domain types in `initialState.ts`
  constrain shapes; form builders (e.g. `buildClientConfig`) normalize/sanitize
  before submission.
- **Observability Built-in** — errors surface as user-facing toasts via the
  `toasts` store; `console.warn`/`console.error` are allowed but
  `console.log` is lint-disallowed.
- **Keep It Boring** — prefer SolidJS primitives (`createSignal`,
  `createMemo`, `createStore`) over clever abstractions.

The easiest way to achieve these principles is **layered architecture**.
This project's layers, from top to bottom:

| Layer      | Responsibility                     | Examples                                               |
| ---------- | ---------------------------------- | ------------------------------------------------------ |
| Components | Render UI, handle user interaction | `src/components/Dashboard/`, `src/components/Clients/` |
| Common UI  | Reusable controls and primitives   | `src/common/controls/Select/`, `src/common/ui/Button/` |
| Stores     | Domain state + async actions       | `src/stores/clients.ts`, `src/stores/queryLogs.ts`     |
| API        | HTTP transport to backend          | `src/api/Api.ts`                                       |
| Helpers    | Pure utilities, validators         | `src/helpers/`, `src/lib/`                             |

```text
Components (pages, controls)
     ↓
Stores (domain state + actions)
     ↓
API (apiClient → fetch /control)
     ↓
AdGuard Home backend (Go)
```

Components may call stores and helpers. Stores may call the API. The API layer
must not depend on stores or components. Helpers are pure and dependency-free.

**Known exclusions** (to be fixed):

- `src/store/types.ts` is a leftover from the Redux era that only re-exports
  types from `initialState.ts`. New code should import types directly from
  `panel/initialState`; this shim exists for backward compatibility.

## Code Quality

- **Path aliases**: Use the `panel/*` alias for all `src/` imports
  (e.g. `import { apiClient } from 'panel/api/Api'`). Use `Twosky` for the
  root `.twosky.json` config. Avoid deep relative paths (`../../..`).
- **Component conventions**: PascalCase directories and component files.
  Each component lives in its own directory with an `index.tsx` and a
  co-located `*.module.pcss`. Sub-components go in a `blocks/` subfolder.
- **Props**: Use `splitProps` from `solid-js` to separate local props from
  forwarded props (see `src/common/ui/Button`). Name the props type after the
  component (e.g. `ButtonProps`) and export it when consumers need it; prefer
  `type` over `interface`. Prefer explicit prop types over `any`.
- **CSS Modules**: Import component styles as `s` — `import s from
'./Foo.module.pcss'` — and reference classes as `s.button`, `s[variant]`.
  Compose conditional classes with `clsx`, imported as `cn`:
  `import cn from 'clsx'` (e.g. `cn(s.button, s.primary, { [s.active]: on() })`).
- **Colors**: Never hardcode color hex values in `.pcss` or inline styles.
  Always reference the CSS custom properties defined in
  `src/common/styles/vars.css` and `src/common/styles/colors/*.css`
  (`var(--default-main-text)`, `var(--default-page-background)`, …). Light and
  dark themes are toggled via the `data-theme` attribute on `<html>`, so using
  the variables handles dark mode automatically.
- **No inline styles**: Do not use the `style` attribute on elements. All
  styling belongs in co-located CSS Modules (`.module.pcss`) using class
  names. If a value must be dynamic, drive it through a CSS custom property or
  a conditional class via `cn(...)`, not an inline `style`. Exception: a single
  computed pixel offset for a dragging/positioning edge case may be acceptable
  when no class-based solution exists — justify it in a comment.
- **Reactivity**: Use `createSignal` for local state, `createMemo` for
  derived values, `createEffect` for side effects, `onMount` for initial data
  fetches, and `onCleanup` for teardown. Do not read signals imperatively
  inside event handlers without `untrack` when needed.
- **Memo placement in `<Tabs>` children**: Don't put a `createMemo` that reads
  an async store inside a component rendered as `<Tabs tabs={[...]}>` content.
  `intl.getMessage()` calls in the `tabs` array subscribe to `lang()` —
  when it changes SolidJS swaps the child instance, disposing its memos.
  Instead, **lift the memo to the parent** and pass the value as a prop.
  If the child uses that prop in a `render` closure inside another
  `createMemo` (e.g., table columns), **access the prop in the memo body**
  so SolidJS tracks the dependency.
- **Stores**: Module-scoped `createStore` singletons exported directly — no
  Context/Provider. Async actions set a `processing*` flag, call `apiClient`,
  then `setState`. Errors are reported via `addErrorToast`.
- **Naming**: Files `PascalCase.tsx` for components, `camelCase.ts` for
  stores/helpers. CSS module files `*.module.pcss`.
- **Error handling**: API errors throw from `Api.ts`; store actions catch and
  surface a toast. Do not swallow errors silently.
- **Logging**: `console.warn` and `console.error` are allowed; `console.log`
  is disallowed by ESLint.
- **Static analysis gates**: ESLint, Prettier, and `tsc --noEmit` must pass.
  `@typescript-eslint/no-explicit-any` is off (legacy), but prefer concrete
  types for new code. Unused vars must be prefixed with `_`.
- **Avoid non-null assertions**: Do not use the TypeScript `!` operator. Use
  type narrowing, optional chaining, or SolidJS control-flow render props to
  guarantee values are defined.
- **Do not modify linter/formatter configs** without justification; they
  enforce the conventions above.

## Testing

- **Unit tests**: Vitest, configured in `vitest.config.ts` (jsdom environment,
  `vite-plugin-solid`). Test files live under `src/__tests__/` and match
  `*.{test,spec}.{ts,tsx}`.
- **Component tests**: Use `@solidjs/testing-library` — `render(() => <Comp/>)`,
  `screen.getByTestId`, `fireEvent`/`userEvent`. Setup file
  `src/__tests__/setup.ts` auto-cleans after each test and adds
  `@testing-library/jest-dom` matchers.
- **Store/helper tests**: Call exported action/builder functions directly and
  assert on returned values or state (e.g.
  `src/__tests__/clientForm/buildClientConfig.test.ts`). Mock `apiClient` when
  a test would otherwise hit the network.
- **Naming**: Mirror the source path under `__tests__/`
  (e.g. `stores/clients.ts` → `__tests__/stores/clients.test.ts`).
- **E2E tests**: Playwright, configured in `playwright.config.ts`
  (`testDir: ./tests/e2e`, base URL `http://127.0.0.1:3000`). Specs run against
  a real AdGuard Home backend prepared by `scripts/prepareConfig.mjs`. E2E
  tests are not part of `npm run check`; run them explicitly with
  `npm run test:e2e`.
- **All unit tests must pass before merge** (`npm run test`).

## Dependency Management

- **Pin all dependency versions explicitly.** Do not introduce version ranges
  that allow automatic upgrades to untested versions.
- **Prefer vanilla solutions.** Use the browser's built-in APIs (`fetch`,
  `crypto.randomUUID` via `nanoid`, CSS custom properties) when they adequately
  solve the problem. Only add a dependency when it provides significant value
  over a vanilla implementation.
- **Reputable sources only.** Dependencies must come from well-established,
  actively maintained projects.
- **Avoid unpopular libraries.** Do not add niche or obscure packages with
  limited community adoption.
- **Minimize dependency count.** Each new dependency increases bundle size and
  maintenance burden; justify every addition.
- **Use the latest stable version** when adding a new dependency — check the
  npm registry for the current stable release rather than copying a stale
  version.

**Rationale**: Fewer, well-vetted dependencies reduce security
vulnerabilities, supply-chain risk, and long-term maintenance cost.

**Known exclusions** (to be fixed):

- Most dependencies in `package.json` use caret ranges (`^`) rather than exact
  pins. This should be tightened to exact versions over time.
- `lodash` is a large dependency; prefer importing only the specific functions
  needed or replacing with native equivalents where possible.

## Configuration & Documentation

- **Runtime configuration**: The dev server (`webpack.dev.js`) reads the
  backend host/port from the root `AdguardHome.yaml` and proxies `/control`
  requests to it. The dev server runs on `backendPort + 8000`.
- **Build configuration**: `BUILD_ENV` (`dev`/`prod`) is set via `cross-env`
  in npm scripts and read from `constants.js`. `BASE_URL = 'control'` is the
  API path prefix.
- **Path aliases**: Defined in `tsconfig.json` (`panel/*`, `Twosky`) and
  consumed by Webpack, Vitest, and ESLint. Update `tsconfig.json` when adding
  new aliases.
- **i18n**: Locale JSON files live in `src/__locales/`. The base locale and
  language list come from the root `.twosky.json` (via the `Twosky` alias in
  `src/helpers/twosky.ts`). Run `npm run translations:check` after editing
  translation files.
- **Documentation to keep in sync**: Update this `AGENTS.md` Project Structure
  when directories change. Update `CHANGELOG.md` (root) for user-facing
  changes following the config in `changelog.config.js`.
- **No secrets in code.** Never hardcode credentials, tokens, or API keys.
  The app authenticates at runtime via the backend login flow.

## Markdown Formatting

All Markdown files MUST follow these formatting rules:

- **Line length**: Keep lines at most 80 characters, but don't overwrap the
  lines artificially short just to hit the limit, keep them close to 80
  characters where possible. This is not a hard lint gate, but SHOULD be
  followed for readability. Lines inside fenced code blocks are exempt from
  this limit.
- **Unordered lists**: Use dashes (`-`) for bullet points. Indent nested
  list items by 4 spaces.
- **Continuation lines**: When a list item wraps to the next line,
  align the continuation with the first character of the item text,
  not the list marker. This applies to all list types (ordered and
  unordered).
- **Emphasis**: Use asterisks (`*`) for emphasis (`*italic*`,
  `**bold**`). Do NOT use underscores.
- **Headings**: Duplicate heading names are allowed only among sibling
  headings (same parent level). Avoid duplicates across different levels.
- **Inline HTML**: Avoid raw HTML in Markdown. The only allowed elements
  are `<a>`, `<p>`, `<details>`, `<summary>`, and `<img>`.
- **Trailing spaces**: Do NOT leave trailing whitespace on any line. Do
  NOT use two-space line breaks — use a blank line instead.
- **Bare URLs**: Bare URLs are permitted and do not need to be wrapped
  in angle brackets.
- **Table formatting**: Align table columns with padding when the table fits
  within 80 characters. If the table exceeds 80 characters or triggers an MD060
  linter warning, switch to a compact format using single spaces only. This
  applies to the separator row as well—it should be written as `| --- |`,
  not `|--|`.

    Example of correct layout:

    ```markdown
    | Col1   | Col2   |
    | ------ | ------ |
    | Value1 | Value2 |
    ```

    Do NOT use extra padding or alignment characters beyond single spaces.

**Rationale**: Uniform Markdown formatting improves readability for both
humans and AI agents that consume project documentation.

## Other

### Accessibility

All interactive components must be keyboard-operable and screen-reader
friendly:

- Use semantic HTML controls for every interaction (`button`, `a`, `input`,
  `select`). Do **not** attach `onClick` to `div`, `span`, or icon elements —
  wrap them in a `<button>`.
- Every interactive element must have an accessible name. Icon-only buttons
  must carry `aria-label={intl.getMessage('key')}` — the label string must be
  a dedicated translation key (see existing usage in `Pagination.tsx`,
  `CopiedText.tsx`, `Input.tsx`).
- Visible labels pair a `<label htmlFor={id}>` with a matching input `id`.
- Focusable rows acting as buttons must be keyboard-activatable: `tabindex` of
  `0` plus an `onKeyDown` handler for Enter/Space.
- Any `id` used for labeling must be unique per rendered instance.

### Translations

- All user-facing strings must be localized. Add new keys to the base locale
  `src/__locales/en.json`; other locales are managed externally via Twosky.
- Access translations via the `intl` object from `panel/common/intl`:
    - `intl.getMessage('key', values?)` — returns a plain localized string.
      Pass interpolation values as the second argument, referenced inside the
      translation value as `%name%` placeholders (e.g.
      `intl.getMessage('client_blocked', { name })` → `"%name% blocked"`).
    - `intl.getPlural('key', number, values?)` — returns a pluralized string
      selected by `number`. Plural values are pipe-delimited into three forms:
      `zero | singular | plural`, each separated by `|`. The leading `|`
      denotes the zero form (used when `number === 0`); it is usually left
      empty, which is why values start with `|`. Use `%count%` (and other
      `%name%`) placeholders for the interpolated count, e.g.
      `"| %count% query total | %count% queries total"`. Always pass the count
      as the second argument and reuse the existing plural key rather than
      branching into separate `getMessage` calls (see
      `Dashboard/blocks/Header/Header.tsx`, `Settings/helpers.ts`).
- **Never use dynamic keys.** Always pass a string literal as the `key`
  argument to `getMessage`/`getPlural`. Do not build keys from variables,
  template literals, or concatenation (e.g. `getMessage(\`btn\_\${action}\`)`).
  Dynamic keys cannot be statically detected, so the translation tooling will
  not pick them up and the strings will go untranslated.
- Do not add duplicate keys to any locale file.
- Run `npm run translations:check` after adding or editing translation keys.
- Every `aria-*` string must have its own dedicated translation key.
