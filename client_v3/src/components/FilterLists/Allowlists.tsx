import { createSignal, createMemo, createEffect, Show, onMount } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { PageLoader } from 'panel/common/ui/Loader';
import theme from 'panel/lib/theme';
import {
    getFilteringStatus,
    toggleFilterStatus,
    refreshFilters,
    filteringState,
} from 'panel/stores/filtering';
import { Icon } from 'panel/common/ui/Icon';

import { openModal } from 'panel/stores/modals';
import { PlusButton } from 'panel/common/ui/PlusButton';
import { FilterUpdateModal } from 'panel/components/FilterLists/blocks/FilterUpdateModal';
import { DeleteAllowlistModal } from './blocks/DeleteAllowlistModal/DeleteAllowlistModal';
import { ConfigureAllowlistModal } from './blocks/ConfigureAllowlistModal';
import { ListsTable, TABLE_IDS } from './blocks/ListsTable/ListsTable';
import s from './FilterLists.module.pcss';

export const Allowlists = () => {
    const [currentFilter, setCurrentFilter] = createSignal<{
        url: string;
        name: string;
        enabled?: boolean;
    }>({
        url: '',
        name: '',
    });

    const [isInitialLoad, setIsInitialLoad] = createSignal(true);

    onMount(() => {
        getFilteringStatus();
    });

    createEffect(() => {
        if (!filteringState.processingFilters && isInitialLoad()) {
            setIsInitialLoad(false);
        }
    });

    const isDataReady = createMemo(() => filteringState.processingFilters && isInitialLoad());

    const handleRefresh = () => {
        refreshFilters({ whitelist: true });
    };

    const toggleFilter = (url: string, data: { name: string; url: string; enabled: boolean }) => {
        toggleFilterStatus(url, data, true);
    };

    const openFilterUpdateModal = () => {
        openModal(MODAL_TYPE.FILTER_UPDATE);
    };

    const openAddAllowlistModal = () => {
        openModal(MODAL_TYPE.ADD_ALLOWLIST);
    };

    const openEditAllowlistModal = (url: string, name: string, enabled: boolean) => {
        setCurrentFilter({ url, name, enabled });
        openModal(MODAL_TYPE.EDIT_ALLOWLIST);
    };

    const openDeleteAllowlistModal = (url: string, name: string) => {
        setCurrentFilter({ url, name });
        openModal(MODAL_TYPE.DELETE_ALLOWLIST);
    };

    return (
        <div class={theme.layout.container}>
            <div class={theme.layout.containerIn}>
                <Show
                    when={isDataReady()}
                    fallback={
                        <>
                            <div class={s.header}>
                                <h1
                                    class={cn(
                                        theme.layout.title,
                                        theme.title.h4,
                                        theme.title.h3_tablet,
                                    )}
                                >
                                    {intl.getMessage('allowlists_title')}
                                </h1>

                                <button
                                    type="button"
                                    onClick={handleRefresh}
                                    disabled={filteringState.processingRefreshFilters}
                                    class={cn(s.button, s.button_checkUpdates)}
                                >
                                    <Icon icon="refresh" color="green" />
                                    <span class={s.labelDesktop}>
                                        {intl.getMessage('check_updates_btn')}
                                    </span>
                                </button>

                                <button
                                    type="button"
                                    onClick={openFilterUpdateModal}
                                    disabled={filteringState.processingRefreshFilters}
                                    class={cn(s.button, s.button_settings)}
                                >
                                    <Icon icon="settings" color="green" />
                                </button>
                            </div>

                            <div class={s.desc}>{intl.getMessage('allowlists_desc')}</div>

                            <div class={cn(s.group, s.buttonGroup)}>
                                <PlusButton onClick={openAddAllowlistModal}>
                                    {intl.getMessage('add_allowlist')}
                                </PlusButton>
                            </div>

                            <Show when={filteringState.whitelistFilters.length > 0}>
                                <div class={cn(s.group, s.tableGroup)}>
                                    <ListsTable
                                        tableId={TABLE_IDS.ALLOWLISTS_TABLE}
                                        filters={filteringState.whitelistFilters}
                                        processingConfigFilter={
                                            filteringState.processingConfigFilter
                                        }
                                        editFilterList={openEditAllowlistModal}
                                        addFilterList={openAddAllowlistModal}
                                        toggleFilterList={toggleFilter}
                                        deleteFilterList={openDeleteAllowlistModal}
                                    />
                                </div>
                            </Show>

                            <ConfigureAllowlistModal modalId={MODAL_TYPE.ADD_ALLOWLIST} />

                            <ConfigureAllowlistModal
                                modalId={MODAL_TYPE.EDIT_ALLOWLIST}
                                filterToEdit={currentFilter()}
                            />

                            <DeleteAllowlistModal
                                filterToDelete={currentFilter()}
                                setFilterToDelete={setCurrentFilter}
                            />

                            <FilterUpdateModal />
                        </>
                    }
                >
                    <PageLoader />
                </Show>
            </div>
        </div>
    );
};
