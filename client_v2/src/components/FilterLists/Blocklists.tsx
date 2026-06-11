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
import { DeleteBlocklistModal } from 'panel/components/FilterLists/blocks/DeleteBlocklistModal';
import { ConfigureBlocklistModal } from 'panel/components/FilterLists/blocks/ConfigureBlocklistModal';
import { PageLoader } from 'panel/common/ui/Loader';
import { PlusButton } from 'panel/common/ui/PlusButton';
import { ListsTable, TABLE_IDS } from './blocks/ListsTable/ListsTable';
import { FilterUpdateModal } from './blocks/FilterUpdateModal';

import s from './FilterLists.module.pcss';

export const Blocklists = () => {
    const dispatch = useDispatch();
    const { filtering } = useSelector((state: RootState) => state);
    const [currentFilter, setCurrentFilter] = useState<{
        url: string;
        name: string;
        enabled?: boolean;
    }>({
        url: '',
        name: '',
    });

    const { filters, processingRefreshFilters, processingConfigFilter, processingFilters } =
        filtering;

    const [isInitialLoad, setIsInitialLoad] = useState(true);

    useEffect(() => {
        dispatch(getFilteringStatus());
    }, [dispatch]);

    useEffect(() => {
        if (!processingFilters && isInitialLoad) {
            setIsInitialLoad(false);
        }
    }, [processingFilters, isInitialLoad]);

    const toggleFilter = (url: string, data: { name: string; url: string; enabled: boolean }) => {
        dispatch(toggleFilterStatus(url, data));
    };

    const handleRefresh = () => {
        dispatch(refreshFilters({ whitelist: false }));
    };

    const openFilterUpdateModal = () => {
        dispatch(openModal(MODAL_TYPE.FILTER_UPDATE));
    };

    const openAddBlocklistModal = () => {
        dispatch(openModal(MODAL_TYPE.ADD_BLOCKLIST));
    };

    const openEditBlocklistModal = (url: string, name: string, enabled: boolean) => {
        setCurrentFilter({ url, name, enabled });
        dispatch(openModal(MODAL_TYPE.EDIT_BLOCKLIST));
    };

    const openDeleteBlocklistModal = (url: string, name: string) => {
        setCurrentFilter({ url, name });
        dispatch(openModal(MODAL_TYPE.DELETE_BLOCKLIST));
    };

    return (
        <div className={theme.layout.container}>
            <div className={theme.layout.containerIn}>
                {processingFilters && isInitialLoad ? (
                    <PageLoader />
                ) : (
                    <>
                        <div className={s.header}>
                            <h1
                                className={cn(
                                    theme.layout.title,
                                    theme.title.h4,
                                    theme.title.h3_tablet,
                                )}
                            >
                                {intl.getMessage('blocklists_title')}
                            </h1>

                            <button
                                type="button"
                                onClick={handleRefresh}
                                disabled={processingRefreshFilters}
                                className={cn(s.button, s.button_checkUpdates)}
                            >
                                <Icon icon="refresh" color="green" />
                                <span className={s.labelDesktop}>
                                    {intl.getMessage('check_updates_btn')}
                                </span>
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

                        <div className={s.desc}>{intl.getMessage('blocklists_desc')}</div>

                        <div className={s.group}>
                            <PlusButton onClick={openAddBlocklistModal}>
                                {intl.getMessage('add_blocklist')}
                            </PlusButton>
                        </div>

                        {filters.length > 0 && (
                            <div className={cn(s.group, s.tableGroup)}>
                                <ListsTable
                                    tableId={TABLE_IDS.BLOCKLISTS_TABLE}
                                    filters={filters}
                                    processingConfigFilter={processingConfigFilter}
                                    toggleFilterList={toggleFilter}
                                    addFilterList={openAddBlocklistModal}
                                    editFilterList={openEditBlocklistModal}
                                    deleteFilterList={openDeleteBlocklistModal}
                                />
                            </div>
                        )}

                        <ConfigureBlocklistModal modalId={MODAL_TYPE.ADD_BLOCKLIST} />

                        <ConfigureBlocklistModal
                            modalId={MODAL_TYPE.EDIT_BLOCKLIST}
                            filterToEdit={currentFilter}
                        />

                        <DeleteBlocklistModal
                            filterToDelete={currentFilter}
                            setFilterToDelete={setCurrentFilter}
                        />

                        <FilterUpdateModal />
                    </>
                )}
            </div>
        </div>
    );
};
