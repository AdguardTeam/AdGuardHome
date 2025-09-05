import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { getCurrentFilter } from 'panel/helpers/helpers';
import filtersCatalog from 'panel/helpers/filters/filters';
import { RootState } from 'panel/initialState';
import theme from 'panel/lib/theme';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import {
    getFilteringStatus,
    removeFilter,
    toggleFilterStatus,
    addFilter,
    toggleFilteringModal,
    refreshFilters,
    editFilter,
} from 'panel/actions/filtering';
import { Icon } from 'panel/common/ui/Icon';
import { Modal } from './blocks/Modal';
import { ListsTable, TABLE_IDS } from './blocks/ListsTable/ListsTable';
import { FilterUpdateModal } from './blocks/FilterUpdateModal';

import s from './Blocklists.module.pcss';

export const Blocklists = () => {
    const dispatch = useDispatch();
    const { filtering } = useSelector((state: RootState) => state);
    const [openConfirmDelete, setOpenConfirmDelete] = useState(false);
    const [filterToDelete, setFilterToDelete] = useState<{ url: string; name: string }>({ url: '', name: '' });
    const [openUpdateModal, setOpenUpdateModal] = useState(false);

    const {
        filters,
        isModalOpen,
        isFilterAdded,
        processingRefreshFilters,
        processingAddFilter,
        processingConfigFilter,
        modalType,
        modalFilterUrl,
    } = filtering;

    useEffect(() => {
        dispatch(getFilteringStatus());
    }, [dispatch]);

    const handleSubmit = (values: Record<string, any>) => {
        switch (modalType) {
            case MODAL_TYPE.EDIT_FILTERS:
                dispatch(editFilter(modalFilterUrl, values));
                break;
            case MODAL_TYPE.ADD_FILTERS: {
                dispatch(addFilter(values.url, values.name));
                break;
            }
            case MODAL_TYPE.SELECT_MODAL_TYPE:
            case MODAL_TYPE.CHOOSE_FILTERING_LIST: {
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
            default:
                break;
        }
    };

    const handleAddFilter = (url: string, name: string) => {
        dispatch(addFilter(url, name));
    };

    const handleToggleFilteringModal = (payload?: { type: string; url?: string }) => {
        dispatch(toggleFilteringModal(payload));
    };

    const handleDeleteOpen = (url: string, name: string) => {
        setFilterToDelete({ url, name });
        setOpenConfirmDelete(true);
    };

    const handleDeleteClose = () => {
        setOpenConfirmDelete(false);
        setFilterToDelete({ url: '', name: '' });
    };

    const handleDeleteConfirm = () => {
        if (filterToDelete.url) {
            dispatch(removeFilter(filterToDelete.url));
        }
        handleDeleteClose();
    };

    const toggleFilter = (url: string, data: { name: string; url: string; enabled: boolean }) => {
        dispatch(toggleFilterStatus(url, data));
    };

    const handleRefresh = () => {
        dispatch(refreshFilters({ whitelist: false }));
    };

    const openSelectTypeModal = () => {
        dispatch(toggleFilteringModal({ type: MODAL_TYPE.SELECT_MODAL_TYPE }));
    };

    const currentFilterData = getCurrentFilter(modalFilterUrl, filters);

    return (
        <div className={cn(theme.layout.container, theme.layout.container_wide)}>
            <div className={s.header}>
                <h1 className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                    {intl.getMessage('blocklists_title')}
                </h1>

                <button
                    type="button"
                    onClick={handleRefresh}
                    disabled={processingRefreshFilters}
                    className={cn(s.button, s.button_checkUpdates)}>
                    <Icon icon="refresh" color="green" />
                    {intl.getMessage('check_updates_btn')}
                </button>

                <button
                    type="button"
                    onClick={() => setOpenUpdateModal(true)}
                    disabled={processingRefreshFilters}
                    className={cn(s.button, s.button_settings)}>
                    <Icon icon="settings" color="green" />
                </button>
            </div>

            <div className={s.desc}>{intl.getMessage('blocklists_desc')}</div>

            <div className={s.group}>
                <button type="button" className={cn(s.button, s.button_add)} onClick={openSelectTypeModal}>
                    <Icon icon="plus" color="green" />
                    {intl.getMessage('add_blocklist')}
                </button>
            </div>

            <Modal
                isOpen={isModalOpen}
                toggleFilteringModal={handleToggleFilteringModal}
                addFilter={handleAddFilter}
                isFilterAdded={isFilterAdded}
                processingAddFilter={processingAddFilter}
                processingConfigFilter={processingConfigFilter}
                handleSubmit={handleSubmit}
                modalType={modalType}
                currentFilterData={currentFilterData}
                filters={filters}
            />

            <div className={s.group}>
                <ListsTable
                    tableId={TABLE_IDS.BLOCKLISTS_TABLE}
                    filters={filters}
                    processingConfigFilter={processingConfigFilter}
                    toggleFilterList={toggleFilter}
                    addFilterList={openSelectTypeModal}
                    editFilterList={handleToggleFilteringModal}
                    deleteFilterList={handleDeleteOpen}
                />
            </div>

            {openConfirmDelete && (
                <ConfirmDialog
                    onClose={handleDeleteClose}
                    onConfirm={handleDeleteConfirm}
                    buttonText={intl.getMessage('remove')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('blocklist_remove')}
                    text={intl.getMessage('blocklist_remove_desc', {
                        value: filterToDelete.name || filterToDelete.url,
                    })}
                    buttonVariant="danger"
                />
            )}

            {openUpdateModal && <FilterUpdateModal onClose={() => setOpenUpdateModal(false)} />}
        </div>
    );
};
