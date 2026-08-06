import { test, expect, type Page } from '@playwright/test';

import { login } from '../helpers/login';
import { SEEDED_ALLOW_FILTERS, SEEDED_FILTERS } from './seededFilters.mjs';

const ENABLED_LIST = SEEDED_FILTERS[0].id;
const DISABLED_LIST = SEEDED_FILTERS[2].id;
const ALLOW_LIST = SEEDED_ALLOW_FILTERS[0].id;

const ALL_BLOCK_LISTS = SEEDED_FILTERS.map(({ id }) => id);

/**
 * A client of its own per test, since the project runs the file fully parallel
 * and a shared fixture would let one worker delete or overwrite the client
 * another one is editing.
 */
type ClientRef = {
    name: string;
    id: string;
};

const refs = {
    add: { name: 'flt-add', id: '192.0.2.71' },
    global: { name: 'flt-global', id: '192.0.2.72' },
    listing: { name: 'flt-listing', id: '192.0.2.73' },
    assign: { name: 'flt-assign', id: '192.0.2.74' },
    selectAll: { name: 'flt-select-all', id: '192.0.2.75' },
    hydrate: { name: 'flt-hydrate', id: '192.0.2.76' },
} satisfies Record<string, ClientRef>;

// The switches hide their checkbox behind a styled label, so the input is
// addressed by id and toggled through the label.
const switchInput = (page: Page, id: string) => page.locator(`input#${id}`);

const toggle = (page: Page, id: string) => page.locator(`label[for="${id}"]`).click();

const readClient = async (page: Page, ref: ClientRef) => {
    const res = await page.request.get('/control/clients');
    expect(res.status(), 'reading clients').toBe(200);

    const body = await res.json();

    return body.clients.find((c: { name: string }) => c.name === ref.name);
};

const removeClient = (page: Page, ref: ClientRef) =>
    page.request.post('/control/clients/delete', { data: { name: ref.name } });

/** Recreates ref on the global filter lists, whatever an earlier run left. */
const resetClient = async (page: Page, ref: ClientRef) => {
    await removeClient(page, ref);

    const res = await page.request.post('/control/clients/add', {
        data: {
            name: ref.name,
            ids: [ref.id],
            use_global_settings: true,
            use_global_blocked_services: true,
        },
    });
    expect(res.status(), 'creating the client').toBe(200);
};

test.describe('Per-client filter lists', () => {
    test.beforeEach(async ({ page }) => {
        await login(page);
    });

    const openClientForm = async (page: Page, ref: ClientRef) => {
        // Reload after the hash change, since Playwright's goto does not always
        // let the hash router react to a same document navigation.
        await page.goto(`/#/clients/edit/${encodeURIComponent(ref.name)}`);
        await page.reload();
        await expect(page.getByTestId('client-form')).toBeVisible();
        await expect(page.locator('input#client-name')).toHaveValue(ref.name);
    };

    const openFilterLists = async (page: Page, ref: ClientRef) => {
        await openClientForm(page, ref);
        await page.getByText('Filter lists', { exact: true }).click();
        await expect(switchInput(page, 'use-own-filter-lists')).toBeEnabled();
    };

    test('lists can be assigned while adding a client', async ({ page }) => {
        test.setTimeout(30000);

        const ref = refs.add;
        await removeClient(page, ref);

        await page.goto('/#/clients/add');
        await expect(page.getByTestId('client-form')).toBeVisible();
        await page.locator('input#client-name').fill(ref.name);
        await page.locator('input#client-identifier-0').fill(ref.id);

        await page.getByText('Filter lists', { exact: true }).click();
        await expect(switchInput(page, 'use-own-filter-lists')).toBeEnabled();

        await toggle(page, 'use-own-filter-lists');
        await toggle(page, `filter_list_ids_${ENABLED_LIST}`);

        await page.goBack();
        await expect(page.getByTestId('client-form')).toBeVisible();
        await page.getByTestId('client-form-save').click();

        await expect
            .poll(async () => {
                const client = await readClient(page, ref);

                return (
                    client && { own: client.use_own_filter_lists, block: client.filter_list_ids }
                );
            })
            .toEqual({ own: true, block: [ENABLED_LIST] });

        await removeClient(page, ref);
    });

    test('a new client uses the global filter lists', async ({ page }) => {
        const ref = refs.global;
        await resetClient(page, ref);

        expect((await readClient(page, ref)).use_own_filter_lists).toBe(false);

        await openFilterLists(page, ref);

        // Every list stays untouchable until the client opts out of the global
        // ones.
        await expect(switchInput(page, `filter_list_ids_${ENABLED_LIST}`)).toBeDisabled();
        await expect(page.getByTestId('client-filter_list_ids-select-all')).toBeDisabled();
    });

    test('the page lists the globally disabled blocklist too', async ({ page }) => {
        const ref = refs.listing;
        await resetClient(page, ref);

        await openFilterLists(page, ref);

        // Every control must be reachable by its visible name.
        await expect(
            page.getByRole('checkbox', { name: 'Use custom filter lists' }),
        ).toBeAttached();

        await Promise.all(
            SEEDED_FILTERS.map(({ name }) =>
                expect(page.getByRole('checkbox', { name })).toBeAttached(),
            ),
        );

        await expect(
            page.getByRole('checkbox', { name: SEEDED_ALLOW_FILTERS[0].name }),
        ).toBeAttached();

        await expect(switchInput(page, `allow_filter_list_ids_${ALLOW_LIST}`)).toBeAttached();
    });

    test('assigning lists reaches the API and survives a reload', async ({ page }) => {
        test.setTimeout(30000);

        const ref = refs.assign;
        await resetClient(page, ref);

        await openFilterLists(page, ref);

        await toggle(page, 'use-own-filter-lists');
        await expect(switchInput(page, `filter_list_ids_${DISABLED_LIST}`)).toBeEnabled();

        await toggle(page, `filter_list_ids_${DISABLED_LIST}`);
        await toggle(page, `allow_filter_list_ids_${ALLOW_LIST}`);

        // Go back through the router, since a reload would drop the form state.
        await page.goBack();
        await expect(page.getByTestId('client-form')).toBeVisible();
        await page.getByTestId('client-form-save').click();

        await expect
            .poll(async () => {
                const client = await readClient(page, ref);

                return {
                    own: client.use_own_filter_lists,
                    block: client.filter_list_ids,
                    allow: client.allow_filter_list_ids,
                };
            })
            .toEqual({ own: true, block: [DISABLED_LIST], allow: [ALLOW_LIST] });

        await openFilterLists(page, ref);
        await expect(switchInput(page, `filter_list_ids_${DISABLED_LIST}`)).toBeChecked();
        await expect(switchInput(page, `filter_list_ids_${ENABLED_LIST}`)).not.toBeChecked();
        await expect(switchInput(page, `allow_filter_list_ids_${ALLOW_LIST}`)).toBeChecked();
    });

    test('the page hydrates on a direct visit and a reload', async ({ page }) => {
        test.setTimeout(30000);

        const ref = refs.hydrate;
        await resetClient(page, ref);

        const res = await page.request.post('/control/clients/update', {
            data: {
                name: ref.name,
                data: {
                    name: ref.name,
                    ids: [ref.id],
                    use_global_settings: true,
                    use_own_filter_lists: true,
                    filter_list_ids: [DISABLED_LIST],
                },
            },
        });
        expect(res.status(), 'assigning the list').toBe(200);

        const url = `/#/clients/edit/${encodeURIComponent(ref.name)}/filter_lists`;

        // Landing straight on the nested route must show the saved policy rather
        // than the add-mode defaults.  Reload after the hash change, since
        // Playwright's goto does not always let the hash router react to a same
        // document navigation, and pasting the URL loads the document anyway.
        await page.goto(url);
        await page.reload();
        await expect(switchInput(page, 'use-own-filter-lists')).toBeChecked();
        await expect(switchInput(page, `filter_list_ids_${DISABLED_LIST}`)).toBeChecked();

        await page.reload();
        await expect(switchInput(page, 'use-own-filter-lists')).toBeChecked();
        await expect(switchInput(page, `filter_list_ids_${DISABLED_LIST}`)).toBeChecked();

        // A save from here must still target the client named by the URL.  Reach
        // the form through the breadcrumb, since the history entry before a
        // reload is not the client form.
        await toggle(page, `filter_list_ids_${ENABLED_LIST}`);
        await page.getByRole('link', { name: ref.name }).click();
        await expect(page.getByTestId('client-form')).toBeVisible();
        await expect(page.locator('input#client-name')).toHaveValue(ref.name);
        await page.getByTestId('client-form-save').click();

        await expect
            .poll(async () => {
                const client = await readClient(page, ref);

                return (
                    client && { own: client.use_own_filter_lists, block: client.filter_list_ids }
                );
            })
            .toEqual({ own: true, block: [ENABLED_LIST, DISABLED_LIST] });
    });

    test('select all and deselect all cover every blocklist', async ({ page }) => {
        test.setTimeout(20000);

        const ref = refs.selectAll;
        await resetClient(page, ref);

        await openFilterLists(page, ref);
        await toggle(page, 'use-own-filter-lists');

        await page.getByTestId('client-filter_list_ids-select-all').click();
        await Promise.all(
            ALL_BLOCK_LISTS.map((id) =>
                expect(switchInput(page, `filter_list_ids_${id}`)).toBeChecked(),
            ),
        );

        await page.getByTestId('client-filter_list_ids-deselect-all').click();
        await Promise.all(
            ALL_BLOCK_LISTS.map((id) =>
                expect(switchInput(page, `filter_list_ids_${id}`)).not.toBeChecked(),
            ),
        );
    });
});
