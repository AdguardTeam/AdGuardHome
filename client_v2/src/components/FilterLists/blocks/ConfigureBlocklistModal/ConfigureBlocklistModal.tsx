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

    const [activeTab, setActiveTab] = createSignal<string>(TAB_TYPE.LIST);
    const [selectedFilterIds, setSelectedFilterIds] = createSignal<Record<string, boolean>>({});

    const [name, setName] = createSignal('');
    const [url, setUrl] = createSignal('');
    const [nameError, setNameError] = createSignal<string | undefined>();
    const [urlError, setUrlError] = createSignal<string | undefined>();

    createEffect(() => {
        setSelectedFilterIds(
            props.modalId === MODAL_TYPE.EDIT_BLOCKLIST && props.filterToEdit
                ? {}
                : selectedValues().selectedFilterIds,
        );
    });

    createEffect(() => {
        setName(props.filterToEdit?.name ?? '');
        setUrl(props.filterToEdit?.url ?? '');
        setNameError(undefined);
        setUrlError(undefined);
    });

    const validateNameField = () => {
        const err = validateRequiredValue(name());
        setNameError(err || undefined);
        return !err;
    };

    const validateUrlField = () => {
        const err = validateRequiredValue(url()) || validatePath(url());
        setUrlError(err || undefined);
        return !err;
    };

    const validateAndSetErrors = () => {
        const nameValid = validateNameField();
        const urlValid = validateUrlField();
        return nameValid && urlValid;
    };

    const handleNameChange = (value: string) => {
        setName(value);
        setNameError(undefined);
    };

    const handleUrlChange = (value: string) => {
        setUrl(value);
        setUrlError(undefined);
    };

    const resetForm = () => {
        setName('');
        setUrl('');
        setNameError(undefined);
        setUrlError(undefined);
    };

    const handleFormSubmit = async (e: Event) => {
        e.preventDefault();

        if (props.modalId === MODAL_TYPE.EDIT_BLOCKLIST) {
            const filterToEdit = props.filterToEdit;

            if (!filterToEdit || !validateAndSetErrors()) {
                return;
            }

            await editFilter(
                filterToEdit.url,
                {
                    name: name(),
                    url: url(),
                    enabled: filterToEdit.enabled ?? true,
                },
                false,
            );

            resetForm();
            closeModal();
            return;
        }

        if (activeTab() === TAB_TYPE.MANUAL) {
            if (!validateAndSetErrors()) {
                return;
            }

            await addFilter(url(), name(), false);

            resetForm();
            closeModal();
            return;
        }

        const existingFilterSources = new Set(
            filteringState.filters.map((filter: Filter) => filter.url),
        );

        const ids = selectedFilterIds();
        const changedValues = Object.entries(ids)?.reduce(
            (acc: Record<string, any>, [key, value]) => {
                if (value && key in filtersCatalog.filters) {
                    const filterSource =
                        filtersCatalog.filters[key as keyof typeof filtersCatalog.filters].source;
                    if (!existingFilterSources.has(filterSource)) {
                        acc[key] = value;
                    }
                }
                return acc;
            },
            {},
        );

        const filtersToAdd = Object.keys(changedValues).map((fieldName) => {
            const { source, name: filterName } =
                filtersCatalog.filters[fieldName as keyof typeof filtersCatalog.filters];
            return { url: source, name: filterName };
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

        for (const filterUrl of filtersToRemove) {
            await removeFilter(filterUrl, false);
        }

        closeModal();
    };

    const handleCancel = () => {
        resetForm();
        closeModal();
    };

    const manualForm = () => (
        <ManualFilterForm
            class={s.formGroup}
            name={name()}
            url={url()}
            nameError={nameError()}
            urlError={urlError()}
            onNameChange={handleNameChange}
            onUrlChange={handleUrlChange}
            onNameBlur={validateNameField}
            onUrlBlur={validateUrlField}
        />
    );

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
                            fallback={manualForm()}
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
                                        content: manualForm(),
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
                            onCard
                        >
                            {intl.getMessage('cancel')}
                        </Button>
                    </div>
                </form>
            </Dialog>
        </ModalWrapper>
    );
};
