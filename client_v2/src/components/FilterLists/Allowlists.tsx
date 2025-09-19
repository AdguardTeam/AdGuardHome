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
import { FilterUpdateModal } from 'panel/components/FilterLists/blocks/FilterUpdateModal';
import { DeleteAllowlistModal } from './blocks/DeleteAllowlistModal/DeleteAllowlistModal';
import { ConfigureAllowlistModal } from './blocks/ConfigureAllowlistModal';
import { ListsTable, TABLE_IDS } from './blocks/ListsTable/ListsTable';
import s from './FilterLists.module.pcss';

export const Allowlists = () => {
    const dispatch = useDispatch();
    const { filtering } = useSelector((state: RootState) => state);
    const [currentFilter, setCurrentFilter] = useState<{ url: string; name: string; enabled?: boolean }>({
        url: '',
        name: '',
    });

    const { whitelistFilters, processingRefreshFilters, processingConfigFilter } = filtering;

    useEffect(() => {
        dispatch(getFilteringStatus());
    }, [dispatch]);

    const handleRefresh = () => {
        dispatch(refreshFilters({ whitelist: true }));
    };

    const toggleFilter = (url: string, data: { name: string; url: string; enabled: boolean }) => {
        dispatch(toggleFilterStatus(url, data, true));
    };

    const openFilterUpdateModal = () => {
        dispatch(openModal(MODAL_TYPE.FILTER_UPDATE));
    };

    const openAddAllowlistModal = () => {
        dispatch(openModal(MODAL_TYPE.ADD_ALLOWLIST));
    };

    const openEditAllowlistModal = (url: string, name: string, enabled: boolean) => {
        setCurrentFilter({ url, name, enabled });
        dispatch(openModal(MODAL_TYPE.EDIT_ALLOWLIST));
    };

    const openDeleteAllowlistModal = (url: string, name: string) => {
        setCurrentFilter({ url, name });
        dispatch(openModal(MODAL_TYPE.DELETE_ALLOWLIST));
    };

    return (
        <div className={theme.layout.container}>
            <div className={theme.layout.containerIn}>
                <div className={s.header}>
                    <h1 className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                        {intl.getMessage('allowlists_title')}
                    </h1>

                    <button
                        type="button"
                        onClick={handleRefresh}
                        disabled={processingRefreshFilters}
                        className={cn(s.button, s.button_checkUpdates)}
                    >
                        <Icon icon="refresh" color="green" />
                        {intl.getMessage('check_updates_btn')}
                    </button>

                    <button
                        type="button"
                        onClick={openFilterUpdateModal}
                        disabled={processingRefreshFilters}
                        className={cn(s.button, s.button_settings)}
                    >
                        <Icon icon="settings" color="green" />
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
                        deleteFilterList={openDeleteAllowlistModal}
                    />
                </div>

                <ConfigureAllowlistModal modalId={MODAL_TYPE.ADD_ALLOWLIST} />

                <ConfigureAllowlistModal modalId={MODAL_TYPE.EDIT_ALLOWLIST} filterToEdit={currentFilter} />

                <DeleteAllowlistModal filterToDelete={currentFilter} setFilterToDelete={setCurrentFilter} />

                <FilterUpdateModal />
            </div>
        </div>
    );
};
