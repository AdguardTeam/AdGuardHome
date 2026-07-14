# Development Guide — client_v2

This document explains how to set up a development environment for the
`client_v2` frontend, run it locally, and contribute code. It is intended for
developers working on the next-generation AdGuard Home web UI.

For code guidelines, architecture, and project structure, see
[AGENTS.md](./AGENTS.md). For user-facing documentation, see the root
[README.md](../README.md).

## Table of Contents

- [Development Guide — client\_v2](#development-guide--client_v2)
  - [Table of Contents](#table-of-contents)
  - [Prerequisites](#prerequisites)
  - [Getting Started](#getting-started)
    - [Clone and Install](#clone-and-install)
    - [Running the Dev Server](#running-the-dev-server)
    - [Production Build](#production-build)
  - [Development Workflow](#development-workflow)
    - [Branching and Pull Requests](#branching-and-pull-requests)
    - [Code Style and Formatting](#code-style-and-formatting)
    - [Type Checking](#type-checking)
    - [Running Tests](#running-tests)
    - [Translations](#translations)
  - [Common Tasks](#common-tasks)
    - [Adding a Component](#adding-a-component)
    - [Adding a Store](#adding-a-store)
    - [Adding an API Method](#adding-an-api-method)
    - [Adding a Translation Key](#adding-a-translation-key)
    - [Debugging in VS Code](#debugging-in-vs-code)
  - [Troubleshooting](#troubleshooting)
    - [Dev server cannot find `../AdguardHome.yaml`](#dev-server-cannot-find-adguardhomeyaml)
    - [API requests return 404 or connection refused](#api-requests-return-404-or-connection-refused)
    - [E2e tests fail to start](#e2e-tests-fail-to-start)
    - ["Multiple instances of Solid" errors in tests](#multiple-instances-of-solid-errors-in-tests)
    - [Port already in use](#port-already-in-use)
    - [Type errors after adding imports](#type-errors-after-adding-imports)
  - [Additional Resources](#additional-resources)

## Prerequisites

Before you begin, install the following tools:

- **Node.js** 20 LTS or newer. The project uses `@types/node` 22; a current
  LTS release is recommended.
- **npm** 10.x or newer (bundled with Node.js).
- **An AdGuard Home backend** running locally. The dev server proxies
  `/control` API requests to it, so a built `AdGuardHome` binary in the
  repository root (or any reachable instance) is required for the UI to
  function. See the root [README.md](../README.md) and
  [HACKING.md](../HACKING.md) for building the Go backend.

No global packages are required — all tooling (Webpack, ESLint, Vitest,
Playwright, TypeScript) is installed locally via `npm install`.

## Getting Started

### Clone and Install

All commands in this guide are run from the `client_v2/` directory.

```sh
git clone https://github.com/AdguardTeam/AdGuardHome.git
cd AdGuardHome/client_v2
npm install
```

`npm install` also runs the Playwright browser install postinstall step. If it
does not, run `npx playwright install` manually before running e2e tests.

### Running the Dev Server

The dev server reads the backend host and port from the root
`AdguardHome.yaml` file (the `bind_host` and `bind_port` fields) and proxies
`/control` requests to that backend. Make sure a backend config exists at
`../AdguardHome.yaml` and that the AdGuard Home backend is running.

Start the dev server:

```sh
npm run dev
```

By default the dev server listens on `bind_port + 8000` (for example, if the
backend runs on port `3000`, the dev server runs on `11000`). Override the
port with the `DEV_SERVER_PORT` environment variable:

```sh
DEV_SERVER_PORT=8080 npm run dev
```

The server opens the main dashboard automatically. The four HTML entry points
are:

- `main` — the dashboard / control panel (HashRouter SPA)
- `install.html` — the first-run setup wizard
- `login.html` — the sign-in page
- `forgot_password.html` — the password-reset flow

If the backend is at `http://127.0.0.1:3000`, the dev server is at
`http://127.0.0.1:11000`, and you can open `install.html`, `login.html`, or
`forgot_password.html` directly.

### Production Build

The production build emits static assets into `../build/static/`, which the Go
binary embeds at compile time.

```sh
npm run build-prod
```

To produce a development build without starting a server:

```sh
npm run build-dev
```

To watch for changes and rebuild continuously:

```sh
npm run watch
```

## Development Workflow

### Branching and Pull Requests

- Create a feature branch from `master` (for example,
  `feature/my-new-feature`).
- Keep changes focused; one concern per pull request.
- Ensure the full check passes before opening a PR (see
  [Running Tests](#running-tests)).
- Follow the contribution workflow described in the root
  [CONTRIBUTING.md](../CONTRIBUTING.md).

### Code Style and Formatting

Lint the source:

```sh
npm run lint
```

Auto-fix lint issues and format with Prettier:

```sh
npm run lint:fix
```

Linting and formatting rules are defined in `.eslintrc` files and `.prettierrc`.
Do not modify these configs without justification. For the full set of code
conventions (path aliases, component structure, CSS modules, reactivity, etc.),
see [AGENTS.md](./AGENTS.md).

### Type Checking

Run the TypeScript compiler in check-only mode:

```sh
npm run typecheck
```

Watch mode for continuous type checking:

```sh
npm run typecheck:watch
```

### Running Tests

Run the full local gate (lint + typecheck + unit tests) with a single command:

```sh
npm run check
```

Unit tests (Vitest, jsdom environment):

```sh
npm run test          # single run
npm run test:watch    # watch mode
```

Unit tests live under `src/__tests__/` and match `*.{test,spec}.{ts,tsx}`.
Mirror the source path under `__tests__/` when adding new tests.

End-to-end tests (Playwright):

```sh
npm run test:e2e                # run all e2e specs
npm run test:e2e:interactive    # open the Playwright UI
npm run test:e2e:debug          # debug mode
npm run test:e2e:codegen        # generate tests by recording actions
```

E2E specs live in `tests/e2e/`. They run against a real AdGuard Home backend
that Playwright starts automatically (see the `webServer` config in
`playwright.config.ts`). The `./AdGuardHome` binary must be present in the
repository root for this to work. E2E tests are **not** part of
`npm run check`; run them explicitly.

### Translations

The base locale is `src/__locales/en.json`. After adding or editing
translation keys, verify consistency:

```sh
npm run translations:check
```

This audits source files for `intl.getMessage` / `intl.getPlural` usage and
reports missing, unused, and dynamic keys. Other locales are managed
externally via Twosky (configured in the root `.twosky.json`).

## Common Tasks

### Adding a Component

1. Create a PascalCase directory under `src/components/` (or
   `src/common/controls/` for primitives, `src/common/ui/` for higher-level
   UI).
2. Add an `index.tsx` and a co-located `*.module.pcss`.
3. Place sub-components in a `blocks/` subfolder.
4. Import styles as `s` and compose classes with `clsx` (imported as `cn`).
5. Use the `panel/*` path alias for all `src/` imports.
6. Add a co-located unit test under `src/__tests__/`.

See the [Code Quality](./AGENTS.md#code-quality) section of `AGENTS.md` for
the full set of conventions.

### Adding a Store

1. Create a `camelCase.ts` module under `src/stores/` with a module-scoped
   `createStore` singleton.
2. Async actions set a `processing*` flag, call `apiClient`, then `setState`.
3. Surface errors via `addErrorToast` from the toasts store.
4. Add a test under `src/__tests__/stores/`.

### Adding an API Method

Add a typed method to the `Api` class in `src/api/Api.ts`. All HTTP calls must
go through this singleton — do not call `fetch` directly from components or
stores.

### Adding a Translation Key

1. Add the key to `src/__locales/en.json`.
2. Use it via `intl.getMessage('key', values?)` or
   `intl.getPlural('key', number, values?)`.
3. Never build keys dynamically — always pass a string literal.
4. Run `npm run translations:check` to verify.

### Debugging in VS Code

- The dev server runs Webpack Dev Server with hot module replacement and
  source maps (`eval-source-map`).
- Set breakpoints directly in `src/` files; VS Code maps them to the bundled
  output automatically.
- For unit tests, use the Vitest VS Code extension or run
  `npm run test:watch` and attach a debugger to the Vitest process.

## Troubleshooting

### Dev server cannot find `../AdguardHome.yaml`

The dev server reads the backend host and port from the root
`AdguardHome.yaml`. Make sure the file exists at the repository root and
contains `bind_host` and `bind_port`. If it is missing, the server falls back
to `0.0.0.0:80` and the proxy will not reach the backend. Start the backend or
create the config file.

### API requests return 404 or connection refused

The dev server proxies `/control` to the backend at `bind_host:bind_port`.
Verify the AdGuard Home backend is running and that the port in
`AdguardHome.yaml` matches the running instance.

### E2e tests fail to start

E2e tests require the `./AdGuardHome` binary in the repository root. Build the
backend first (see [HACKING.md](../HACKING.md)). Playwright starts the backend
automatically; if a server is already running on port 3000, stop it first:

```sh
kill $(lsof -ti :3000)
```

### "Multiple instances of Solid" errors in tests

The Vitest config (see `vitest.config.ts`) forces a single `solid-js` instance
via `dedupe` and `ssr.noExternal`. If you add a dependency that pulls in its
own copy of `solid-js`, add it to the `dedupe` and `noExternal` lists.

### Port already in use

Stop any process occupying the dev server port or override it with
`DEV_SERVER_PORT`:

```sh
kill $(lsof -ti :11000)
DEV_SERVER_PORT=8080 npm run dev
```

### Type errors after adding imports

Use the `panel/*` alias (`panel/api/Api`, `panel/initialState`, etc.) instead
of deep relative paths. The alias is defined in `tsconfig.json` and consumed
by Webpack, Vitest, and ESLint.

## Additional Resources

- [AGENTS.md](./AGENTS.md) — code guidelines, architecture, and project
  structure
- [README.md](../README.md) — AdGuard Home project overview and user manual
- [HACKING.md](../HACKING.md) — AdGuard Home developer guidelines (Go backend)
- [CONTRIBUTING.md](../CONTRIBUTING.md) — contribution workflow
- [CHANGELOG.md](../CHANGELOG.md) — changelog
- [AdGuard Code Guidelines](https://github.com/AdguardTeam/CodeGuidelines) —
  general code style guidelines
