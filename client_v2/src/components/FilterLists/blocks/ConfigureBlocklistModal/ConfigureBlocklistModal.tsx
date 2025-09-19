import React, { Dispatch, SetStateAction, useEffect, useMemo, useState } from 'react';

import intl from 'panel/common/intl';
import { Dialog } from 'panel/common/ui/Dialog/Dialog';
import { MODAL_TYPE, TAB_TYPE } from 'panel/helpers/constants';

import { ModalWrapper } from 'panel/common/ui/ModalWrapper';
import { useDispatch, useSelector } from 'react-redux';
import { closeModal } from 'panel/reducers/modals';
import theme from 'panel/lib/theme';
import { Button } from 'panel/common/ui/Button';
import { FormProvider, useForm } from 'react-hook-form';
import { RootState } from 'panel/initialState';
import { addFilter, editFilter } from 'panel/actions/filtering';
import { Filter } from 'panel/helpers/helpers';
import { ManualFilterForm } from 'panel/components/FilterLists/blocks/ConfigureBlocklistModal/blocks/ManualFilterForm';
import { Tabs } from 'panel/common/ui/Tabs';
import filtersCatalog from 'panel/helpers/filters/filters';
import { FiltersList } from './blocks/FiltersList';

type FormValues = {
    name: string;
    url: string;
    enabled?: boolean;
};

const defaultValues: FormValues = {
    name: '',
    url: '',
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

const getSelectedValues = (filters: Filter[], catalogSourcesToIdMap: Record<string, string>): SelectedValues =>
    filters.reduce(
        (acc: SelectedValues, { url }: Filter) => {
            if (Object.prototype.hasOwnProperty.call(catalogSourcesToIdMap, url)) {
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

const getFormContent = ({
    modalId,
    activeTab,
    onTabChange,
    selectedSources,
}: {
    modalId: ConfigureBlocklistModalIdType;
    activeTab: string;
    onTabChange: Dispatch<SetStateAction<string>>;
    selectedSources?: Record<string, boolean>;
}) => {
    switch (modalId) {
        case MODAL_TYPE.ADD_BLOCKLIST: {
            return (
                <Tabs
                    activeTab={activeTab}
                    onTabChange={onTabChange}
                    tabs={[
                        {
                            id: TAB_TYPE.LIST,
                            label: intl.getMessage('blocklist_add_from_list'),
                            content: <FiltersList selectedSources={selectedSources} />,
                        },
                        {
                            id: TAB_TYPE.MANUAL,
                            label: intl.getMessage('blocklist_add_manual'),
                            content: <ManualFilterForm />,
                        },
                    ]}
                />
            );
        }
        case MODAL_TYPE.EDIT_BLOCKLIST: {
            return <ManualFilterForm />;
        }
        default: {
            return null;
        }
    }
};

export const ConfigureBlocklistModal = ({ modalId, filterToEdit }: Props) => {
    const dispatch = useDispatch();
    const { filtering } = useSelector((state: RootState) => state);
    const { processingAddFilter, filters } = filtering;

    const catalogSourcesToIdMap = useMemo(() => {
        const map: Record<string, string> = {};
        Object.entries(filtersCatalog.filters).forEach(([filterId, filterData]) => {
            map[filterData.source] = filterId;
        });
        return map;
    }, []);

    const { selectedFilterIds, selectedSources } = useMemo(
        () => getSelectedValues(filters, catalogSourcesToIdMap),
        [filters, catalogSourcesToIdMap],
    );

    const initialValues = useMemo(() => {
        if (modalId === MODAL_TYPE.EDIT_BLOCKLIST && filterToEdit) {
            return { ...defaultValues, ...filterToEdit };
        }
        if (modalId === MODAL_TYPE.ADD_BLOCKLIST) {
            return { ...defaultValues, ...selectedFilterIds };
        }
        return defaultValues;
    }, [modalId, filterToEdit, selectedFilterIds]);

    const methods = useForm({
        defaultValues: {
            ...initialValues,
        },
        mode: 'onBlur',
    });
    const { handleSubmit, reset } = methods;

    const [activeTab, setActiveTab] = useState(TAB_TYPE.LIST);

    useEffect(() => {
        reset(initialValues);
    }, [initialValues, reset]);

    const handleFormSubmit = async (values: FormValues) => {
        switch (modalId) {
            case MODAL_TYPE.ADD_BLOCKLIST: {
                if (values.url && values.name) {
                    // Manual filter form submission
                    dispatch(addFilter(values.url, values.name));
                } else {
                    // Filter list selection submission
                    const existingFilterSources = new Set(filters.map((filter) => filter.url));

                    const changedValues = Object.entries(values)?.reduce((acc: Record<string, any>, [key, value]) => {
                        if (value && key in filtersCatalog.filters) {
                            const filterSource = filtersCatalog.filters[key].source;
                            // Only include if not already added
                            if (!existingFilterSources.has(filterSource)) {
                                acc[key] = value;
                            }
                        }
                        return acc;
                    }, {});

                    Object.keys(changedValues).forEach((fieldName) => {
                        const { source, name } = filtersCatalog.filters[fieldName];
                        dispatch(addFilter(source, name));
                    });
                }
                break;
            }
            case MODAL_TYPE.EDIT_BLOCKLIST: {
                dispatch(editFilter(values.url, values));
                dispatch(closeModal());
                break;
            }
            default: {
                break;
            }
        }
    };

    const handleCancel = () => {
        reset(initialValues);
        dispatch(closeModal());
    };

    return (
        <ModalWrapper id={modalId}>
            <Dialog visible onClose={handleCancel} title={getTitle(modalId)}>
                <FormProvider {...methods}>
                    <form onSubmit={handleSubmit(handleFormSubmit)}>
                        <div>
                            {getFormContent({
                                modalId,
                                activeTab,
                                onTabChange: setActiveTab,
                                selectedSources,
                            })}
                        </div>

                        <div className={theme.dialog.footer}>
                            <Button
                                type="submit"
                                id="filters_save"
                                variant="primary"
                                size="small"
                                disabled={processingAddFilter}
                                className={theme.dialog.button}
                            >
                                {getButtonText(modalId)}
                            </Button>

                            <Button
                                type="button"
                                id="filters_cancel"
                                variant="secondary"
                                size="small"
                                onClick={handleCancel}
                                className={theme.dialog.button}
                            >
                                {intl.getMessage('cancel')}
                            </Button>
                        </div>
                    </form>
                </FormProvider>
            </Dialog>
        </ModalWrapper>
    );
};
