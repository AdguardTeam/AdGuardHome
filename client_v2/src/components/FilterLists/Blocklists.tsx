import { createSignal, createMemo, createEffect, Show, onMount } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { MODAL_TYPE } from 'panel/helpers/constants';
import theme from 'panel/lib/theme';
import {
    getFilteringStatus,
    toggleFilterStatus,
    refreshFilters,
    filteringState,
} from 'panel/stores/filtering';
import { Icon } from 'panel/common/ui/Icon';
import { openModal } from 'panel/stores/modals';
import { DeleteBlocklistModal } from 'panel/components/FilterLists/blocks/DeleteBlocklistModal';
import { ConfigureBlocklistModal } from 'panel/components/FilterLists/blocks/ConfigureBlocklistModal';
import { PageLoader } from 'panel/common/ui/Loader';
import { PlusButton } from 'panel/common/ui/PlusButton';
import { ListsTable, TABLE_IDS } from './blocks/ListsTable/ListsTable';
import { FilterUpdateModal } from './blocks/FilterUpdateModal';

import s from './FilterLists.module.pcss';

export const Blocklists = () => {
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

    const toggleFilter = (url: string, data: { name: string; url: string; enabled: boolean }) => {
        toggleFilterStatus(url, data, false);
    };

    const handleRefresh = () => {
        refreshFilters({ whitelist: false });
    };

    const openFilterUpdateModal = () => {
        openModal(MODAL_TYPE.FILTER_UPDATE);
    };

    const openAddBlocklistModal = () => {
        openModal(MODAL_TYPE.ADD_BLOCKLIST);
    };

    const openEditBlocklistModal = (url: string, name: string, enabled: boolean) => {
        setCurrentFilter({ url, name, enabled });
        openModal(MODAL_TYPE.EDIT_BLOCKLIST);
    };

    const openDeleteBlocklistModal = (url: string, name: string) => {
        setCurrentFilter({ url, name });
        openModal(MODAL_TYPE.DELETE_BLOCKLIST);
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
                                    {intl.getMessage('blocklists_title')}
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

                            <div class={s.desc}>{intl.getMessage('blocklists_desc')}</div>

                            <div class={cn(s.group, s.buttonGroup)}>
                                <PlusButton onClick={openAddBlocklistModal}>
                                    {intl.getMessage('add_blocklist')}
                                </PlusButton>
                            </div>

                            <Show when={filteringState.filters.length > 0}>
                                <div class={cn(s.group, s.tableGroup)}>
                                    <ListsTable
                                        tableId={TABLE_IDS.BLOCKLISTS_TABLE}
                                        filters={filteringState.filters}
                                        processingConfigFilter={
                                            filteringState.processingConfigFilter
                                        }
                                        toggleFilterList={toggleFilter}
                                        addFilterList={openAddBlocklistModal}
                                        editFilterList={openEditBlocklistModal}
                                        deleteFilterList={openDeleteBlocklistModal}
                                    />
                                </div>
                            </Show>

                            <ConfigureBlocklistModal modalId={MODAL_TYPE.ADD_BLOCKLIST} />

                            <ConfigureBlocklistModal
                                modalId={MODAL_TYPE.EDIT_BLOCKLIST}
                                filterToEdit={currentFilter()}
                            />

                            <DeleteBlocklistModal
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
