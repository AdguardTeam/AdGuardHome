import { createSignal, createMemo, Show, createEffect } from 'solid-js';

import intl from 'panel/common/intl';
import { Dialog } from 'panel/common/ui/Dialog/Dialog';
import { MODAL_TYPE, TAB_TYPE } from 'panel/helpers/constants';

import { ModalWrapper } from 'panel/common/ui/ModalWrapper';
import { closeModal } from 'panel/stores/modals';
import theme from 'panel/lib/theme';
import { Button } from 'panel/common/ui/Button';
import {
    addFilter,
    addFiltersBatch,
    editFilter,
    removeFilter,
    filteringState,
} from 'panel/stores/filtering';
import type { Filter } from 'panel/helpers/helpers';
import { validatePath, validateRequiredValue } from 'panel/helpers/validators';
import { ManualFilterForm } from 'panel/components/FilterLists/blocks/ConfigureBlocklistModal/blocks/ManualFilterForm';
import { Tabs } from 'panel/common/ui/Tabs';
import filtersCatalog from 'panel/helpers/filters/filters';
import { InlineLoader } from 'panel/common/ui/Loader/InlineLoader';
import { FiltersList } from './blocks/FiltersList';
import s from './ConfigureBlocklistModal.module.pcss';

type FormValues = {
    name: string;
    url: string;
    enabled?: boolean;
};

type ConfigureBlocklistModalIdType = 'ADD_BLOCKLIST' | 'EDIT_BLOCKLIST';

type Props = {
    modalId: ConfigureBlocklistModalIdType;
    filterToEdit?: FormValues;
};

type SelectedValues = {
    selectedFilterIds: Record<string, boolean>;
    selectedSources: Record<string, boolean>;
};

const getSelectedValues = (
    filters: Filter[],
    catalogSourcesToIdMap: Record<string, string>,
): SelectedValues =>
    filters.reduce(
        (acc: SelectedValues, { url }: Filter) => {
            if (Object.hasOwn(catalogSourcesToIdMap, url)) {
                const filterId = catalogSourcesToIdMap[url];
                acc.selectedFilterIds[filterId] = true;
                acc.selectedSources[url] = true;
            }
            return acc;
        },
        {
            selectedFilterIds: {} as Record<string, boolean>,
            selectedSources: {} as Record<string, boolean>,
        } as SelectedValues,
    );

const getTitle = (modalId: ConfigureBlocklistModalIdType) => {
    if (modalId === MODAL_TYPE.EDIT_BLOCKLIST) {
        return intl.getMessage('blocklist_edit');
    }

    return intl.getMessage('blocklists_add');
};

const getButtonText = (modalId: ConfigureBlocklistModalIdType) => {
    if (modalId === MODAL_TYPE.EDIT_BLOCKLIST) {
        return intl.getMessage('save');
    }

    return intl.getMessage('add');
};

export const ConfigureBlocklistModal = (props: Props) => {
    const catalogSourcesToIdMap = createMemo(() => {
        const map: Record<string, string> = {};
        Object.entries(filtersCatalog.filters).forEach(([filterId, filterData]) => {
            map[filterData.source] = filterId;
        });
        return map;
    });

    const selectedValues = createMemo(() =>
        getSelectedValues(filteringState.filters, catalogSourcesToIdMap()),
    );

    const [activeTab, setActiveTab] = createSignal(TAB_TYPE.LIST);
    const [selectedFilterIds, setSelectedFilterIds] = createSignal<Record<string, boolean>>({});

    createEffect(() => {
        setSelectedFilterIds(
            props.modalId === MODAL_TYPE.EDIT_BLOCKLIST && props.filterToEdit
                ? {}
                : selectedValues().selectedFilterIds,
        );
    });

    const handleFormSubmit = async (e: Event) => {
        e.preventDefault();

        const form = e.target as HTMLFormElement;
        const formData = new FormData(form);
        const values: FormValues = {
            name: (formData.get('name') as string) || '',
            url: (formData.get('url') as string) || '',
        };

        switch (props.modalId) {
            case MODAL_TYPE.ADD_BLOCKLIST: {
                if (values.url && values.name) {
                    const nameErr = validateRequiredValue(values.name);
                    const urlErr = validateRequiredValue(values.url) || validatePath(values.url);
                    if (nameErr || urlErr) {
                        return;
                    }
                    addFilter(values.url, values.name, false);
                } else {
                    const existingFilterSources = new Set(
                        filteringState.filters.map((filter: Filter) => filter.url),
                    );

                    const ids = selectedFilterIds();
                    const changedValues = Object.entries(ids)?.reduce(
                        (acc: Record<string, any>, [key, value]) => {
                            if (value && key in filtersCatalog.filters) {
                                const filterSource =
                                    filtersCatalog.filters[
                                        key as keyof typeof filtersCatalog.filters
                                    ].source;
                                if (!existingFilterSources.has(filterSource)) {
                                    acc[key] = value;
                                }
                            }
                            return acc;
                        },
                        {},
                    );

                    const filtersToAdd = Object.keys(changedValues).map((fieldName) => {
                        const { source, name } =
                            filtersCatalog.filters[
                                fieldName as keyof typeof filtersCatalog.filters
                            ];
                        return { url: source, name };
                    });
                    if (filtersToAdd.length > 0) {
                        await addFiltersBatch(filtersToAdd);
                    }

                    const initialSelected = selectedValues().selectedFilterIds;
                    const currentIds = selectedFilterIds();
                    const filtersToRemove = Object.entries(initialSelected)
                        .filter(([id, wasSelected]) => wasSelected && !currentIds[id])
                        .map(([id]) => {
                            const { source } =
                                filtersCatalog.filters[id as keyof typeof filtersCatalog.filters];
                            return source;
                        });

                    for (const url of filtersToRemove) {
                        await removeFilter(url, false);
                    }
                }
                break;
            }
            case MODAL_TYPE.EDIT_BLOCKLIST: {
                editFilter(props.filterToEdit!.url, values, false);
                break;
            }
            default: {
                break;
            }
        }

        closeModal();
    };

    const handleCancel = () => {
        closeModal();
    };

    return (
        <ModalWrapper id={props.modalId}>
            <Dialog visible onClose={handleCancel} title={getTitle(props.modalId)}>
                <form onSubmit={handleFormSubmit}>
                    <div>
                        <Show when={props.modalId !== MODAL_TYPE.EDIT_BLOCKLIST}>
                            <p class={s.desc}>{intl.getMessage('blocklists_add_desc')}</p>
                        </Show>
                        <Show
                            when={props.modalId === MODAL_TYPE.ADD_BLOCKLIST}
                            fallback={
                                <ManualFilterForm
                                    class={s.formGroup}
                                    initialName={props.filterToEdit?.name}
                                    initialUrl={props.filterToEdit?.url}
                                />
                            }
                        >
                            <Tabs
                                activeTab={activeTab()}
                                onTabChange={setActiveTab}
                                contentClass={s.content}
                                tabs={[
                                    {
                                        id: TAB_TYPE.LIST,
                                        label: intl.getMessage('blocklist_add_from_list'),
                                        content: (
                                            <FiltersList
                                                selectedSources={selectedValues().selectedSources}
                                                selectedIds={selectedFilterIds()}
                                                onChange={setSelectedFilterIds}
                                                disabled={filteringState.processingAddFilter}
                                            />
                                        ),
                                    },
                                    {
                                        id: TAB_TYPE.MANUAL,
                                        label: intl.getMessage('blocklist_add_manual'),
                                        content: (
                                            <ManualFilterForm
                                                class={s.formGroup}
                                                initialName={props.filterToEdit?.name}
                                                initialUrl={props.filterToEdit?.url}
                                            />
                                        ),
                                    },
                                ]}
                            />
                        </Show>
                    </div>

                    <div class={theme.dialog.footer}>
                        <Button
                            type="submit"
                            id="filters_save"
                            variant="primary"
                            size="small"
                            disabled={filteringState.processingAddFilter}
                            leftAddon={
                                filteringState.processingAddFilter ? <InlineLoader /> : undefined
                            }
                            class={theme.dialog.button}
                        >
                            {getButtonText(props.modalId)}
                        </Button>

                        <Button
                            type="button"
                            id="filters_cancel"
                            variant="secondary"
                            size="small"
                            onClick={handleCancel}
                            class={theme.dialog.button}
                        >
                            {intl.getMessage('cancel')}
                        </Button>
                    </div>
                </form>
            </Dialog>
        </ModalWrapper>
    );
};
