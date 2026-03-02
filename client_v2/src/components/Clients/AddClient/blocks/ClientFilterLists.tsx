import { createMemo, onMount, For, Show } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Switch } from 'panel/common/controls/Switch';
import { Button } from 'panel/common/ui/Button';
import { Breadcrumbs } from 'panel/common/ui/Breadcrumbs';
import { SwitchGroup } from 'panel/common/ui/SettingsGroup';
import { clientFormState, updateClientFormField } from 'panel/stores/clientForm';
import { filteringState, getFilteringStatus } from 'panel/stores/filtering';
import { RoutePath } from 'panel/components/Routes/Paths';
import type { Filter } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';

import { hydrateClientForm } from './hydrateClientForm';

import s from './ClientFilterLists.module.pcss';

/** Field of [ClientFormState] holding the IDs of a kind of filter list. */
type ListField = 'filter_list_ids' | 'allow_filter_list_ids';

export const ClientFilterLists = () => {
    const formReady = hydrateClientForm();

    onMount(() => {
        getFilteringStatus();
    });

    // Turning the switch on with no list loaded would save an own list policy
    // with no IDs, which drops every global list for the client.  The arrays are
    // empty both before the first fetch and after a failed one, so the controls
    // wait for a confirmed load.
    const ready = createMemo(() => filteringState.statusLoaded && formReady());

    const isEdit = createMemo(() => clientFormState.mode === 'edit');

    const clientPageLink = createMemo(() =>
        isEdit()
            ? {
                  path: RoutePath.ClientsEdit,
                  title: clientFormState.name || intl.getMessage('clients_edit'),
                  props: { clientName: encodeURIComponent(clientFormState.originalName) },
              }
            : {
                  path: RoutePath.ClientsAdd,
                  title: intl.getMessage('clients_add'),
              },
    );

    const breadcrumbs = createMemo(() => ({
        parentLinks: [
            { path: RoutePath.Clients, title: intl.getMessage('client_settings') },
            clientPageLink(),
        ],
        currentTitle: intl.getMessage('clients_filter_lists'),
    }));

    const useOwn = createMemo(() => clientFormState.use_own_filter_lists);

    const handleUseOwn = (e: Event) => {
        updateClientFormField({
            field: 'use_own_filter_lists',
            value: (e.target as HTMLInputElement).checked,
        });
    };

    const selected = (field: ListField) => new Set<number>(clientFormState[field] || []);

    const handleToggle = (field: ListField, id: number, checked: boolean) => {
        const ids = selected(field);
        if (checked) {
            ids.add(id);
        } else {
            ids.delete(id);
        }

        updateClientFormField(
            field,
            [...ids].sort((a, b) => a - b),
            true,
        );
    };

    const handleToggleAll = (field: ListField, lists: Filter[], checked: boolean) => {
        updateClientFormField(field, checked ? lists.map((f) => f.id) : [], true);
    };

    const section = (field: ListField, title: string, lists: () => Filter[]) => (
        <Show when={lists().length > 0}>
            <h2 class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {title}
            </h2>

            <div class={s.actions}>
                <Button
                    variant="secondary"
                    size="small"
                    disabled={!ready() || !useOwn()}
                    onClick={() => handleToggleAll(field, lists(), true)}
                    data-testid={`client-${field}-select-all`}
                >
                    {intl.getMessage('select_all')}
                </Button>
                <Button
                    variant="secondary"
                    size="small"
                    disabled={!ready() || !useOwn()}
                    onClick={() => handleToggleAll(field, lists(), false)}
                    data-testid={`client-${field}-deselect-all`}
                >
                    {intl.getMessage('deselect_all')}
                </Button>
            </div>

            <div class={s.list}>
                <For each={lists()}>
                    {(filter) => (
                        <div class={s.row}>
                            <div class={s.rowContent}>
                                <div class={cn(theme.text.t2, s.rowTitle)}>{filter.name}</div>
                                <Show when={!filter.enabled}>
                                    <div class={cn(theme.text.t3, s.rowDesc)}>
                                        {intl.getMessage('clients_filter_list_disabled')}
                                    </div>
                                </Show>
                            </div>
                            <Switch
                                id={`${field}_${filter.id}`}
                                ariaLabel={filter.name}
                                checked={selected(field).has(filter.id)}
                                disabled={!ready() || !useOwn()}
                                onChange={(e: Event) =>
                                    handleToggle(
                                        field,
                                        filter.id,
                                        (e.target as HTMLInputElement).checked,
                                    )
                                }
                            />
                        </div>
                    )}
                </For>
            </div>
        </Show>
    );

    return (
        <div class={cn(theme.layout.container, s.containerOverride)}>
            <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <div class={s.breadcrumbs}>
                    <Breadcrumbs
                        parentLinks={breadcrumbs().parentLinks}
                        currentTitle={breadcrumbs().currentTitle}
                    />
                </div>

                <h1 class={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                    {intl.getMessage('clients_filter_lists')}
                </h1>
                <p class={s.description}>{intl.getMessage('clients_filter_lists_desc')}</p>

                <SwitchGroup
                    id="use-own-filter-lists"
                    ariaLabel={intl.getMessage('clients_use_own_filter_lists')}
                    title={intl.getMessage('clients_use_own_filter_lists')}
                    description={intl.getMessage('clients_use_own_filter_lists_desc')}
                    checked={useOwn()}
                    onChange={handleUseOwn}
                    disabled={!ready()}
                />

                {section('filter_list_ids', intl.getMessage('blocklists_title'), () =>
                    (filteringState.filters || []).filter((f: Filter) => f.id > 0),
                )}

                {section('allow_filter_list_ids', intl.getMessage('allowlists_title'), () =>
                    (filteringState.whitelistFilters || []).filter((f: Filter) => f.id > 0),
                )}
            </div>
        </div>
    );
};
