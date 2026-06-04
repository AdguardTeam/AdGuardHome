# Quick Specification: Clients Page & Client Add/Edit Form E2E Tests

**Status**: Implemented
**Implemented by**: Deepseek V4 Pro Max Thinking

## Problem Analysis

**Problem**: The clients page (`Clients.tsx`) and the client add/edit form
(`AddClient.tsx`) have no end-to-end browser tests. Critical flows — adding a
new persistent client, editing an existing client, and deleting a client —
are only exercised through unit tests that mock Redux and routing. The query
log already has a comprehensive Playwright spec (`tests/e2e/query-log.spec.ts`)
that demonstrates the project's testing patterns with `page.route` API mocking,
so the approach is well understood.

**Type**: Missing test coverage (new tests).

## Research

### Existing patterns to follow

The primary reference is `tests/e2e/query-log.spec.ts` (1200 lines). Key
patterns:

1. **API mocking with `page.route`**: tests intercept `**/control/*` endpoints
   and return mock JSON. Stateful mocks update in-memory state so subsequent
   GET and POST handlers stay consistent.
2. **Login helper**: a `login(page: Page)` function navigates to `/login.html`,
   fills credentials, and waits for the dashboard URL. The `query-log.spec.ts`
   version retries up to 3 times.
3. **Test structure**: `test.describe('suite name', () => { ... })` with
   independent `test(...)` blocks.
4. **Constants**: shared in `tests/constants.ts` — `ADMIN_USERNAME`,
   `ADMIN_PASSWORD`, `PORT`, `CONFIG_FILE_PATH`, `WORK_DIR_PATH`.
5. **TypeScript type definitions**: test files define inline types matching
   the API response shapes, not imported from production code.

### API endpoints used by the clients page

| Endpoint                        | Method | Purpose                                             |
| ------------------------------- | ------ | --------------------------------------------------- |
| `/control/clients`              | GET    | Fetch persistent + runtime clients + supported tags |
| `/control/clients/add`          | POST   | Create a new persistent client                      |
| `/control/clients/update`       | POST   | Update an existing persistent client                |
| `/control/clients/delete`       | POST   | Delete a persistent client                          |
| `/control/stats`                | GET    | Fetch statistics (for top clients data)             |
| `/control/blocked_services/all` | GET    | Fetch blocked services list (for service icons)     |

The `GET /control/clients` response shape:

```ts
type ClientsResponse = {
    clients: Client[];
    auto_clients: AutoClient[];
    supported_tags: string[];
};
```

### Source files involved

| File                                                                              | Role                                         |
| --------------------------------------------------------------------------------- | -------------------------------------------- |
| `src/components/Clients/Clients.tsx`                                              | Clients list page                            |
| `src/components/Clients/AddClient/AddClient.tsx`                                  | Add/edit client form                         |
| `src/components/Clients/blocks/PersistentClientsTable/PersistentClientsTable.tsx` | Persistent clients table                     |
| `src/components/Clients/blocks/RuntimeClientsTable/RuntimeClientsTable.tsx`       | Runtime clients table                        |
| `src/components/Clients/AddClient/blocks/Identifiers/Identifiers.tsx`             | Identifier input fields                      |
| `src/actions/clients.ts`                                                          | Client CRUD actions (old API)                |
| `src/actions/clientForm.ts`                                                       | Form actions (new API with `saveClient`)     |
| `src/initialState.ts`                                                             | `Client`, `ClientFormState` type definitions |
| `tests/constants.ts`                                                              | E2E shared constants                         |
| `tests/e2e/query-log.spec.ts`                                                     | Reference test pattern                       |
| `tests/e2e/login.spec.ts`                                                         | Simpler login test pattern                   |

### Forms elements (by `id` and accessible name)

The AddClient form uses `id` attributes on inputs and title/text on buttons.

| Element                    | Selector                                                              |
| -------------------------- | --------------------------------------------------------------------- |
| Client name input          | `#client-name`                                                        |
| Identifier inputs          | `#client-identifier-0`, `#client-identifier-1`, etc.                  |
| Add identifier button      | `getByRole('button', { name: 'Add identifier' })` or `getByText(...)` |
| Tags select                | `getByText('Tags')` (followed by clicking `.select__control`)         |
| Use global settings switch | `#use-global-settings`                                                |
| Upstreams textarea         | `#client-upstreams`                                                   |
| DNS cache switch           | `#use-dns-cache`                                                      |
| DNS cache size input       | `#dns-cache-size`                                                     |
| Save button                | `getByRole('button', { name: 'Save' })`                               |
| Cancel button              | `getByRole('button', { name: 'Cancel' })`                             |
| Add Client (page header)   | `getByRole('button', { name: 'Add Client' })`                         |
| Edit row action            | `getByRole('button', { name: 'Edit' })` (button title attribute)      |
| Delete row action          | `getByRole('button', { name: 'Delete' })` (button title attribute)    |
| Confirm delete button      | `getByRole('button', { name: 'Remove' })`                             |

No `data-testid` attributes exist on Clients components; selectors use `id`,
`role`, and accessible names.

### Edge cases and constraints

- The clients page relies on `useEffect` to fetch data on mount. Tests must
  mock the API **before** navigation.
- The add/edit form URL depends on hash routing:
    - Add: `/#clients/add`
    - Edit: `/#clients/edit/:clientName`
    - Deletion uses a `ConfirmDialog` component (modal).
- Form validation: name is required, identifiers must not be empty.
- The edit form auto-initializes from URL params when the page is reloaded
  directly; the test must navigate via the UI (click edit on a table row) or
  pre-populate the Redux form state — the simpler approach is to use the UI
  flow since Playwright can click edit buttons.
- The PersistentClientsTable and RuntimeClientsTable use internal pagination
  (page sizes: 5/10/20/50/100). With only 1-2 mock clients, pagination won't
  be tested.

### Complexity assessment

- **Multiple unrelated components**: No — everything is within the clients
  domain.
- **New entities or data models**: No — existing `Client` type is reused.
- **API contract changes**: No — tests consume the existing API.
- **New user-facing features**: No — this is test coverage only.
- **Cross-cutting concerns**: No.

**Recommendation**: Quick spec is appropriate (no full SDD needed).

## Clarifications Needed

Before proceeding with implementation, I need to clarify one point:

1. **Scope of "add client" test**: The add client form has sub-pages for
   protection settings and blocked services (navigated via nav links). Should
   the e2e test cover:
    - (A) Only the main form (name, identifiers, tags, upstreams, save) —
      minimal critical path
    - (B) Also navigate to protection and blocked services sub-pages

    The user said "critical functionality only" and referenced query-log tests
    as examples. The query-log tests are comprehensive but focused. I'll
    assume option (A) — the main form fields — unless the user specifies
    otherwise.

## Solution Design

### Approach

Create `tests/e2e/clients.spec.ts` following the query-log pattern:

1. Use `page.route` to intercept all `/control/*` API calls and return mock
   data.
2. Provide a `login` helper.
3. Write independent test cases for each critical flow.
4. Mock `GET /control/clients`, `POST /control/clients/add`,
   `POST /control/clients/update`, `POST /control/clients/delete`,
   `GET /control/stats`, and `GET /control/blocked_services/all`.

### Test cases

1. **Clients page renders sections**: Navigate to `/#clients`, verify
   "Persistent clients" and "Runtime clients" headings are visible, verify
   mock client data appears in the table.

2. **Add a new client**: Click "Add Client", fill in name and identifier,
   click Save, verify the POST payload was sent with correct data, verify
   navigation back to clients page.

3. **Edit an existing client**: Click edit on a persistent client row, verify
   form is pre-filled, change the name, click Save, verify the POST update
   payload.

4. **Delete a client**: Click delete on a persistent client row, verify
   confirmation dialog, confirm deletion, verify the DELETE API call.

5. **Add client form validation**: Leave name empty, click Save, verify
   validation error is shown.

### Files to create or modify

| File                                                                              | Action | Purpose                                                            |
| --------------------------------------------------------------------------------- | ------ | ------------------------------------------------------------------ |
| `src/components/Clients/Clients.tsx`                                              | Edit   | Add `data-testid` to heading and "Add Client" button               |
| `src/components/Clients/blocks/PersistentClientsTable/PersistentClientsTable.tsx` | Edit   | Add `data-testid` to edit/delete action buttons                    |
| `src/components/Clients/AddClient/AddClient.tsx`                                  | Edit   | Add `data-testid` to form wrapper, name input, save/cancel buttons |
| `src/components/Clients/AddClient/blocks/Identifiers/Identifiers.tsx`             | Edit   | Add `data-testid` to add-identifier button                         |
| `tests/e2e/clients.spec.ts`                                                       | Create | E2E tests for clients page and add/edit form                       |

### Verification

```sh
cd client_v2 && npm run build-prod && npm run test:e2e -- tests/e2e/clients.spec.ts
```

## Implementation Plan

### File Structure

```
client_v2/
  src/
    components/
      Clients/
        Clients.tsx                                          # Edit: add data-testid
        blocks/
          PersistentClientsTable/
            PersistentClientsTable.tsx                       # Edit: add data-testid
        AddClient/
          AddClient.tsx                                      # Edit: add data-testid
          blocks/
            Identifiers/
              Identifiers.tsx                                # Edit: add data-testid
  tests/
    e2e/
      clients.spec.ts                                        # New: all clients e2e tests
```

### Task 1: Add `data-testid` to the Clients page

**File**: `src/components/Clients/Clients.tsx`

Add `data-testid` attributes to the page heading and "Add Client" button so
e2e tests can target them reliably.

Find:

```tsx
<h1 className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
    {intl.getMessage('client_settings')}
</h1>
```

Replace with:

```tsx
<h1
    className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}
    data-testid="clients-title"
>
    {intl.getMessage('client_settings')}
</h1>
```

And find:

```tsx
<button
    type="button"
    onClick={handleAddClient}
    className={cn(s.button, s.button_add)}
>
```

Replace with:

```tsx
<button
    type="button"
    onClick={handleAddClient}
    className={cn(s.button, s.button_add)}
    data-testid="clients-add-button"
>
```

### Task 2: Add `data-testid` to the PersistentClientsTable action buttons

**File**: `src/components/Clients/blocks/PersistentClientsTable/PersistentClientsTable.tsx`

Add `data-testid` attributes to the edit and delete buttons inside the
"actions" column render function.

Find the edit button (inside the `render` of the `actions` column):

```tsx
<button
    type="button"
    onClick={() => onEdit(row)}
    disabled={editDisabled}
    className={s.action}
    title={intl.getMessage('edit_table_action')}
>
```

Replace with:

```tsx
<button
    type="button"
    onClick={() => onEdit(row)}
    disabled={editDisabled}
    className={s.action}
    title={intl.getMessage('edit_table_action')}
    data-testid="clients-edit-button"
>
```

Find the delete button:

```tsx
<button
    type="button"
    onClick={() => onDelete(row.name)}
    disabled={deleteDisabled}
    className={cn(s.action, s.action_danger)}
    title={intl.getMessage('delete_table_action')}
>
```

Replace with:

```tsx
<button
    type="button"
    onClick={() => onDelete(row.name)}
    disabled={deleteDisabled}
    className={cn(s.action, s.action_danger)}
    title={intl.getMessage('delete_table_action')}
    data-testid="clients-delete-button"
>
```

### Task 3: Add `data-testid` to the AddClient form

**File**: `src/components/Clients/AddClient/AddClient.tsx`

Add `data-testid` attributes to the form wrapper, client name input, and
save/cancel buttons.

Find the form wrapper div (the outermost div after the container):

```tsx
<div className={cn(theme.layout.container, s.containerOverride)}>
```

Replace with:

```tsx
<div className={cn(theme.layout.container, s.containerOverride)} data-testid="client-form">
```

Find the name `Input`:

```tsx
<Input
    id="client-name"
    type="text"
    value={form.name}
```

Replace with:

```tsx
<Input
    id="client-name"
    data-testid="client-form-name"
    type="text"
    value={form.name}
```

Find the Save `Button`:

```tsx
<Button variant="primary" size="small" onClick={handleSave} disabled={form.processingSave}>
    {intl.getMessage('save_btn')}
</Button>
```

Replace with:

```tsx
<Button
    variant="primary"
    size="small"
    onClick={handleSave}
    disabled={form.processingSave}
    data-testid="client-form-save"
>
    {intl.getMessage('save_btn')}
</Button>
```

Find the Cancel `Button`:

```tsx
<Button variant="secondary" size="small" onClick={handleCancel}>
    {intl.getMessage('cancel_btn')}
</Button>
```

Replace with:

```tsx
<Button variant="secondary" size="small" onClick={handleCancel} data-testid="client-form-cancel">
    {intl.getMessage('cancel_btn')}
</Button>
```

### Task 4: Add `data-testid` to the Identifiers "Add identifier" button

**File**: `src/components/Clients/AddClient/blocks/Identifiers/Identifiers.tsx`

Find the "Add identifier" button:

```tsx
<button type="button" className={s.addButton} onClick={handleAdd}>
```

Replace with:

```tsx
<button type="button" className={s.addButton} onClick={handleAdd} data-testid="client-form-add-identifier">
```

### Task 5: Write the e2e test file with mock types, constants, and login helper

**File**: `tests/e2e/clients.spec.ts`

Create the file with all imports, mock types, mock data, login helper, and
API mock setup.

```ts
import { test, expect, type Page } from '@playwright/test';

import { ADMIN_USERNAME, ADMIN_PASSWORD } from '../constants';

// ---- Mock data types ----

type Client = {
    name: string;
    ids: string[];
    tags: string[];
    use_global_settings: boolean;
    filtering_enabled: boolean;
    safebrowsing_enabled: boolean;
    parental_enabled: boolean;
    safe_search: Record<string, boolean>;
    ignore_querylog: boolean;
    ignore_statistics: boolean;
    blocked_services: string[];
    use_global_blocked_services: boolean;
    blocked_services_schedule: { time_zone: string };
    upstreams: string[];
    upstreams_cache_enabled: boolean;
    upstreams_cache_size: number;
};

type AutoClient = {
    ip: string;
    name: string;
    source: string;
    whois_info: Record<string, unknown>;
};

type ClientsResponse = {
    clients: Client[];
    auto_clients: AutoClient[];
    supported_tags: string[];
};

type AddClientPayload = {
    name: string;
    ids: string[];
    tags: string[];
    use_global_settings: boolean;
    filtering_enabled: boolean;
    safebrowsing_enabled: boolean;
    parental_enabled: boolean;
    safe_search: Record<string, boolean>;
    ignore_querylog: boolean;
    ignore_statistics: boolean;
    blocked_services: string[];
    use_global_blocked_services: boolean;
    blocked_services_schedule: { time_zone: string };
    upstreams: string[];
    upstreams_cache_enabled: boolean;
    upstreams_cache_size: number;
};

type UpdateClientPayload = {
    name: string;
    data: AddClientPayload;
};

type DeleteClientPayload = {
    name: string;
};

// ---- Mock data ----

const MOCK_CLIENT_1: Client = {
    name: 'Office Desktop',
    ids: ['192.168.0.100'],
    tags: ['work'],
    use_global_settings: true,
    filtering_enabled: true,
    safebrowsing_enabled: true,
    parental_enabled: false,
    safe_search: {
        enabled: false,
        google: false,
        youtube: false,
        bing: false,
        duckduckgo: false,
        yandex: false,
        pixabay: false,
    },
    ignore_querylog: false,
    ignore_statistics: false,
    blocked_services: [],
    use_global_blocked_services: true,
    blocked_services_schedule: { time_zone: 'UTC' },
    upstreams: [],
    upstreams_cache_enabled: false,
    upstreams_cache_size: 0,
};

const MOCK_CLIENT_2: Client = {
    name: 'Living Room TV',
    ids: ['AA:BB:CC:DD:EE:FF'],
    tags: [],
    use_global_settings: false,
    filtering_enabled: true,
    safebrowsing_enabled: false,
    parental_enabled: true,
    safe_search: {
        enabled: true,
        google: true,
        youtube: true,
        bing: false,
        duckduckgo: false,
        yandex: false,
        pixabay: false,
    },
    ignore_querylog: false,
    ignore_statistics: false,
    blocked_services: ['youtube'],
    use_global_blocked_services: false,
    blocked_services_schedule: { time_zone: 'Europe/London' },
    upstreams: ['tls://1.1.1.1'],
    upstreams_cache_enabled: true,
    upstreams_cache_size: 4194304,
};

const MOCK_AUTO_CLIENT: AutoClient = {
    ip: '192.168.0.200',
    name: 'android-phone.local',
    source: 'rDNS',
    whois_info: {},
};

const DEFAULT_CLIENTS_RESPONSE: ClientsResponse = {
    clients: [MOCK_CLIENT_1, MOCK_CLIENT_2],
    auto_clients: [MOCK_AUTO_CLIENT],
    supported_tags: ['work', 'home', 'guest'],
};

// ---- Helpers ----

async function login(page: Page) {
    let lastError: unknown;

    for (let attempt = 0; attempt < 3; attempt += 1) {
        await page.goto('/login.html', { waitUntil: 'domcontentloaded' });

        try {
            await page.locator('#username').waitFor({ state: 'visible', timeout: 5000 });
            await page.locator('#username').fill(ADMIN_USERNAME);
            await page.locator('#password').fill(ADMIN_PASSWORD);
            await page.locator('#sign_in').click();
            await page.waitForURL((url) => !url.href.endsWith('/login.html'));

            return;
        } catch (error) {
            lastError = error;
        }
    }

    throw lastError;
}

type ClientsMocksResult = {
    addClientPayloads: AddClientPayload[];
    updateClientPayloads: UpdateClientPayload[];
    deleteClientPayloads: DeleteClientPayload[];
};

async function setupClientsMocks(
    page: Page,
    { clientsResponse = DEFAULT_CLIENTS_RESPONSE }: { clientsResponse?: ClientsResponse } = {},
): Promise<ClientsMocksResult> {
    const addClientPayloads: AddClientPayload[] = [];
    const updateClientPayloads: UpdateClientPayload[] = [];
    const deleteClientPayloads: DeleteClientPayload[] = [];
    let clientsState = JSON.parse(JSON.stringify(clientsResponse)) as ClientsResponse;

    await page.route('**/control/clients', (route) => {
        if (route.request().method() === 'GET') {
            route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify(clientsState),
            });
        } else {
            route.continue();
        }
    });

    await page.route('**/control/clients/add', (route) => {
        const payload = route.request().postDataJSON() as AddClientPayload;
        addClientPayloads.push(payload);
        const newClient: Client = { ...payload, upstreams: payload.upstreams || [] };
        clientsState = {
            ...clientsState,
            clients: [...clientsState.clients, newClient],
        };
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
    });

    await page.route('**/control/clients/update', (route) => {
        const payload = route.request().postDataJSON() as UpdateClientPayload;
        updateClientPayloads.push(payload);
        clientsState = {
            ...clientsState,
            clients: clientsState.clients.map((c) =>
                c.name === payload.name
                    ? { ...payload.data, name: payload.data.name || payload.name }
                    : c,
            ),
        };
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
    });

    await page.route('**/control/clients/delete', (route) => {
        const payload = route.request().postDataJSON() as DeleteClientPayload;
        deleteClientPayloads.push(payload);
        clientsState = {
            ...clientsState,
            clients: clientsState.clients.filter((c) => c.name !== payload.name),
        };
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
    });

    await page.route('**/control/stats', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
                dns_queries: [],
                blocked_filtering: [],
                avg_processing_time: 0,
                top_clients: [],
                top_queried_domains: [],
                top_blocked_domains: [],
                stats_period: {},
                num_dns_queries: 0,
                num_blocked_filtering: 0,
                num_replaced_safebrowsing: 0,
                num_replaced_safesearch: 0,
                num_replaced_parental: 0,
            }),
        });
    });

    await page.route('**/control/blocked_services/all', (route) => {
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
                blocked_services: [{ id: 'youtube', name: 'YouTube' }],
                groups: [],
            }),
        });
    });

    return { addClientPayloads, updateClientPayloads, deleteClientPayloads };
}
```

### Task 6: Test — Clients page renders with persistent and runtime clients

**File**: `tests/e2e/clients.spec.ts` (append)

```ts
test.describe('Clients', () => {
    test('renders the clients page with persistent and runtime clients', async ({ page }) => {
        await setupClientsMocks(page);
        await login(page);
        await page.goto('/#clients');

        // Page heading
        await expect(page.getByTestId('clients-title')).toBeVisible();
        await expect(page.getByTestId('clients-add-button')).toBeVisible();

        // Persistent clients section heading
        await expect(page.getByText('Persistent clients')).toBeVisible();

        // Mock clients should appear in the table
        await expect(page.getByText('Office Desktop')).toBeVisible({ timeout: 10_000 });
        await expect(page.getByText('192.168.0.100')).toBeVisible();
        await expect(page.getByText('Living Room TV')).toBeVisible();

        // Runtime clients section heading
        await expect(page.getByText('Runtime clients')).toBeVisible();
        await expect(page.getByText('192.168.0.200')).toBeVisible();
    });
```

### Task 7: Test — Add a new persistent client

**File**: `tests/e2e/clients.spec.ts` (append)

```ts
test('adds a new persistent client', async ({ page }) => {
    const { addClientPayloads } = await setupClientsMocks(page);
    await login(page);
    await page.goto('/#clients');

    // Click "Add Client" in the page header
    await page.getByTestId('clients-add-button').click();
    await expect(page).toHaveURL(/#clients\/add$/);
    await expect(page.getByTestId('client-form')).toBeVisible();

    // Fill in name and identifier
    await page.getByTestId('client-form-name').fill('Test Client');
    await page.locator('#client-identifier-0').fill('192.168.0.50');

    // Save
    await page.getByTestId('client-form-save').click();
    await expect(page).toHaveURL(/#clients$/);

    // Verify API payload
    await expect.poll(() => addClientPayloads.length).toBe(1);
    expect(addClientPayloads[0].name).toBe('Test Client');
    expect(addClientPayloads[0].ids).toEqual(['192.168.0.50']);
});
```

### Task 8: Test — Edit an existing persistent client

**File**: `tests/e2e/clients.spec.ts` (append)

```ts
test('edits an existing persistent client', async ({ page }) => {
    const { updateClientPayloads } = await setupClientsMocks(page);
    await login(page);
    await page.goto('/#clients');

    // Find the row containing "Office Desktop" and click its edit button
    const officeRow = page.locator('tr', { hasText: 'Office Desktop' });
    await expect(officeRow).toBeVisible();
    await officeRow.getByTestId('clients-edit-button').click();

    await expect(page).toHaveURL(/#clients\/edit\/Office%20Desktop$/);
    await expect(page.getByTestId('client-form')).toBeVisible();

    // Verify pre-filled data
    await expect(page.getByTestId('client-form-name')).toHaveValue('Office Desktop');
    await expect(page.locator('#client-identifier-0')).toHaveValue('192.168.0.100');

    // Change name and save
    await page.getByTestId('client-form-name').fill('Office Desktop Updated');
    await page.getByTestId('client-form-save').click();

    await expect(page).toHaveURL(/#clients$/);

    // Verify update payload
    await expect.poll(() => updateClientPayloads.length).toBe(1);
    expect(updateClientPayloads[0].name).toBe('Office Desktop');
    expect(updateClientPayloads[0].data.name).toBe('Office Desktop Updated');
});
```

### Task 9: Test — Delete a client with confirmation dialog

**File**: `tests/e2e/clients.spec.ts` (append)

```ts
test('deletes a persistent client with confirmation', async ({ page }) => {
    const { deleteClientPayloads } = await setupClientsMocks(page);
    await login(page);
    await page.goto('/#clients');

    // Find the row for "Living Room TV" and click delete
    const tvRow = page.locator('tr', { hasText: 'Living Room TV' });
    await expect(tvRow).toBeVisible();
    await tvRow.getByTestId('clients-delete-button').click();

    // ConfirmDialog should appear
    await expect(page.getByText(/Are you sure you want to delete client/)).toBeVisible();

    // Click confirm "Remove"
    await page.getByRole('button', { name: 'Remove' }).click();

    // Verify DELETE API call
    await expect.poll(() => deleteClientPayloads.length).toBe(1);
    expect(deleteClientPayloads[0].name).toBe('Living Room TV');
});
```

### Task 10: Test — Form validation on empty name

**File**: `tests/e2e/clients.spec.ts` (append)

```ts
    test('shows validation error when client name is empty', async ({ page }) => {
        const { addClientPayloads } = await setupClientsMocks(page);
        await login(page);
        await page.goto('/#clients/add');

        await expect(page.getByTestId('client-form')).toBeVisible();

        // Fill identifier but leave name empty
        await page.locator('#client-identifier-0').fill('192.168.0.99');

        // Click Save
        await page.getByTestId('client-form-save').click();

        // Should show validation error
        await expect(page.locator('#client-name')).toHaveAttribute('aria-invalid', 'true');

        // No API call
        expect(addClientPayloads).toHaveLength(0);
    });
});
```

### Task 11: Build and run all clients e2e tests

```sh
cd client_v2 && npm run build-prod && npx playwright test tests/e2e/clients.spec.ts
```

All 5 tests should pass. If any fail, debug with:

```sh
npx playwright test tests/e2e/clients.spec.ts --debug
```

### Task 12: Verify formatting, linting, and typecheck

```sh
cd client_v2 && npx prettier --check tests/e2e/clients.spec.ts
cd client_v2 && npm run lint
cd client_v2 && npm run typecheck
```

## Self-Review

### Spec coverage

The plan covers all critical flows:

- [x] Add `data-testid` attributes to source components (Tasks 1–4)
- [x] Clients list page rendering with mock data (Task 6)
- [x] Add a new client (Task 7)
- [x] Edit an existing client (Task 8)
- [x] Delete a client with confirmation (Task 9)
- [x] Form validation (Task 10)

### Placeholder scan

- [x] No "TBD", "TODO", or "implement later"
- [x] No vague "add appropriate error handling"
- [x] Every code step has concrete code
- [x] No "Similar to Task N" without repeating the code
- [x] All types referenced are defined in the spec (Client, AutoClient, etc.)

### Type consistency

- [x] `AddClientPayload` used consistently in mock setup and assertions
- [x] `UpdateClientPayload` structure matches `{ name, data }` pattern from the API
- [x] `DeleteClientPayload` matches `{ name }` shape
- [x] `MOCK_CLIENT_1` and `MOCK_CLIENT_2` conform to the `Client` type

### `data-testid` patterns

- [x] Follows existing project conventions (QueryLog uses `query-log-*`, DNS
      Rewrites uses `rewrite-*`)
- [x] Uses `clients-*` prefix for the list page and `client-form-*` prefix
      for the add/edit form
- [x] Identifier inputs keep their existing `id` attributes; no additional
      `data-testid` needed
- [x] `Input` component passes through `data-testid` to the inner `<input>`
      element

## Implementation Notes

### Deviations from spec

1. **Table row selectors**: The `Table` component uses `<div>` elements with CSS
   grid (not `<table>/<tr>/<td>`), so `page.locator('tr', { hasText: ... })`
   does not work. Changed to `page.getByTestId('clients-edit-button').nth(1)`
   and `page.getByTestId('clients-delete-button').first()`.

2. **Validation error check**: The `Input` component does not set `aria-invalid`
   attribute on error; it renders an error message text instead. Changed
   assertion from `toHaveAttribute('aria-invalid', 'true')` to checking for
   visible error text `getByText('Please fill out this field')`.

3. **Additional API mock**: The page calls `POST /control/clients/search` which
   was not in the original spec. Added a mock returning an empty array to
   prevent the real server response from causing a client-side `TypeError`.

### Test results

All 5 tests pass (2.7s):

- ✓ renders the clients page with persistent and runtime clients
- ✓ adds a new persistent client
- ✓ edits an existing persistent client
- ✓ deletes a persistent client with confirmation
- ✓ shows validation error when client name is empty
