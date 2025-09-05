import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { RootState } from 'panel/initialState';
import theme from 'panel/lib/theme';
import { getFilteringStatus, toggleFilterStatus, refreshFilters } from 'panel/actions/filtering';
import { Icon } from 'panel/common/ui/Icon';

import { openModal } from 'panel/reducers/modals';
import { DeleteAllowlistModal } from './blocks/DeleteAllowlistModal/DeleteAllowlistModal';
import { ConfigureAllowlistModal } from './blocks/ConfigureAllowlistModal';
import { ListsTable, TABLE_IDS } from './blocks/ListsTable/ListsTable';
import s from './Allowlists.module.pcss';

export const Allowlists = () => {
    const dispatch = useDispatch();
    const { filtering } = useSelector((state: RootState) => state);
    const [filterToDelete, setFilterToDelete] = useState<{ url: string; name: string }>({ url: '', name: '' });

    const { whitelistFilters, processingRefreshFilters, processingConfigFilter } = filtering;

    useEffect(() => {
        dispatch(getFilteringStatus());
    }, [dispatch]);

    const handleRefresh = () => {
        dispatch(refreshFilters({ whitelist: true }));
    };

    const toggleFilter = (url: string, data: { name: string; url: string; enabled: boolean }) => {
        dispatch(toggleFilterStatus(url, data));
    };

    const openAddAllowlistModal = () => {
        dispatch(openModal(MODAL_TYPE.ADD_ALLOWLIST));
    };

    const openEditAllowlistModal = () => {
        dispatch(openModal(MODAL_TYPE.EDIT_ALLOWLIST));
    };

    const openDeleteAllowFilterModal = (url: string, name: string) => {
        setFilterToDelete({ url, name });
        dispatch(openModal(MODAL_TYPE.DELETE_ALLOWLIST));
    };

    // const currentFilterData = getCurrentFilter(modalFilterUrl, filters);

    return (
        <div className={cn(theme.layout.container, theme.layout.container_wide)}>
            <div className={s.header}>
                <h1 className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                    {intl.getMessage('allowlists_title')}
                </h1>

                <button
                    type="button"
                    onClick={handleRefresh}
                    disabled={processingRefreshFilters}
                    className={cn(s.button, s.button_checkUpdates)}>
                    <Icon icon="refresh" color="green" />
                    {intl.getMessage('check_updates_btn')}
                </button>
            </div>

            <div className={s.desc}>{intl.getMessage('allowlists_desc')}</div>

            <div className={s.group}>
                <button type="button" className={cn(s.button, s.button_add)} onClick={openAddAllowlistModal}>
                    <Icon icon="plus" color="green" />
                    {intl.getMessage('add_allowlist')}
                </button>
            </div>

            <div className={cn(s.group, s.stretchSelf)}>
                <ListsTable
                    tableId={TABLE_IDS.ALLOWLISTS_TABLE}
                    filters={whitelistFilters}
                    processingConfigFilter={processingConfigFilter}
                    editFilterList={openEditAllowlistModal}
                    addFilterList={openAddAllowlistModal}
                    toggleFilterList={toggleFilter}
                    deleteFilterList={openDeleteAllowFilterModal}
                />
            </div>

            <ConfigureAllowlistModal modalId={MODAL_TYPE.ADD_ALLOWLIST} />

            <ConfigureAllowlistModal modalId={MODAL_TYPE.EDIT_ALLOWLIST} />

            <DeleteAllowlistModal filterToDelete={filterToDelete} setFilterToDelete={setFilterToDelete} />
        </div>
    );
};
