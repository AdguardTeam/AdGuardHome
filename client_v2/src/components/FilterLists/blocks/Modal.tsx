import React from 'react';

import intl from 'panel/common/intl';
import { Dialog } from 'panel/common/ui/Dialog/Dialog';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { Filter } from 'panel/helpers/helpers';
import filtersCatalog from 'panel/helpers/filters/filters';

import { Form, FormValues } from './Form';

const getTitle = (modalType: string) => {
    if (modalType === MODAL_TYPE.CHOOSE_FILTERING_LIST) {
        return intl.getMessage('blocklists_add_list');
    }
    if (modalType === MODAL_TYPE.EDIT_FILTERS) {
        return intl.getMessage('blocklist_edit');
    }
    return intl.getMessage('blocklists_add');
};

interface SelectedValues {
    selectedFilterIds: Record<string, boolean>;
    selectedSources: Record<string, boolean>;
}

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

type Props = {
    toggleFilteringModal: () => void;
    isOpen: boolean;
    addFilter: (url: string, name: string) => void;
    isFilterAdded: boolean;
    processingAddFilter: boolean;
    processingConfigFilter: boolean;
    handleSubmit: (values: FormValues) => void;
    modalType: string;
    currentFilterData: Partial<Filter>;
    filters: Filter[];
};

export const Modal = ({
    isOpen,
    processingAddFilter,
    processingConfigFilter,
    handleSubmit,
    modalType,
    currentFilterData,
    toggleFilteringModal,
    filters,
}: Props) => {
    const closeModal = () => {
        toggleFilteringModal();
    };

    let initialValues: Partial<FormValues> | undefined;

    const catalogSourcesToIdMap: Record<string, string> = {};
    Object.entries(filtersCatalog.filters).forEach(([filterId, filterData]) => {
        catalogSourcesToIdMap[filterData.source] = filterId;
    });

    const { selectedFilterIds, selectedSources } = getSelectedValues(filters, catalogSourcesToIdMap);

    switch (modalType) {
        case MODAL_TYPE.EDIT_FILTERS:
            initialValues = currentFilterData as Partial<FormValues>;
            break;
        case MODAL_TYPE.SELECT_MODAL_TYPE:
        case MODAL_TYPE.CHOOSE_FILTERING_LIST: {
            initialValues = selectedFilterIds as Partial<FormValues>;
            break;
        }
        default:
            break;
    }

    const title = getTitle(modalType);

    return (
        <Dialog visible={isOpen} onClose={closeModal} title={title}>
            <Form
                selectedSources={selectedSources}
                initialValues={initialValues}
                modalType={modalType}
                onSubmit={handleSubmit}
                processingAddFilter={processingAddFilter}
                processingConfigFilter={processingConfigFilter}
                closeModal={closeModal}
                toggleFilteringModal={toggleFilteringModal}
            />
        </Dialog>
    );
};
