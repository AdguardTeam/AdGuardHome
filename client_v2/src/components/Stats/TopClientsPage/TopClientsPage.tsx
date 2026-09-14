import { Show, createMemo, createSignal, onMount } from 'solid-js';
import cn from 'clsx';
import { useNavigate } from '@solidjs/router';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import type { TableColumn } from 'panel/common/ui/Table';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Link } from 'panel/common/ui/Link';
import {
    ClientBlockConfirmDialog,
    type ClientBlockAction,
    useClientBlockConfirm,
} from 'panel/common/ui/ClientBlockConfirm';
import { Icon } from 'panel/common/ui/Icon';
import { statsState } from 'panel/stores/stats';
import { accessState, getAccessList } from 'panel/stores/access';
import { initClientForm } from 'panel/stores/clientForm';
import { LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import { computePercent } from 'panel/helpers/statistics';
import { splitByNewLine } from 'panel/helpers/helpers';
import type { IOption } from 'panel/lib/helpers/utils';
import { PlusButton } from 'panel/common/ui/PlusButton';
import { Paths, RoutePath } from 'panel/components/Routes/Paths';
import { StatsPage } from '../StatsPage';
import { StatMobileCard, type StatCardItem } from '../blocks/StatMobileCard';
import { CountWithPercent } from '../blocks/CountWithPercent';
import { useStatsRefresh } from '../hooks/useStatsRefresh';

import s from './TopClientsPage.module.pcss';

type ClientStat = {
    name: string;
    count: number;
    info?: { name?: string; whois_info?: Record<string, string> };
};

export const TopClientsPage = () => {
    const navigate = useNavigate();
    const refreshStats = useStatsRefresh();

    onMount(() => {
        refreshStats();
        getAccessList();
    });

    const handleAddClient = () => {
        initClientForm(null);
        navigate(Paths.ClientsAdd);
    };

    const handleRefresh = () => {
        refreshStats();
        getAccessList();
    };

    const {
        confirmState: confirmDialog,
        isClientBlocked: isBlocked,
        openConfirmDialog,
        closeConfirmDialog,
        handleConfirm,
    } = useClientBlockConfirm();

    const [openMenuClient, setOpenMenuClient] = createSignal<string | null>(null);

    const openClientConfirmDialog = (client: string, action: ClientBlockAction) => {
        setOpenMenuClient(null);
        openConfirmDialog(client, action);
    };

    const [showOnlyBlocked, setShowOnlyBlocked] = createSignal(false);

    // Track the access list: <For> mappers run untracked, so rows must be
    // re-created (cloned) to re-render block state.
    const blockedSet = createMemo(() => new Set(splitByNewLine(accessState.disallowed_clients)));

    const rows = createMemo<ClientStat[]>(() => {
        void blockedSet();
        return [...statsState.topClients]
            .toSorted((a, b) => (a.info?.name ?? a.name).localeCompare(b.info?.name ?? b.name))
            .map((row) => ({ ...row }));
    });

    const visibleRows = createMemo<ClientStat[]>(() =>
        showOnlyBlocked() ? rows().filter((row) => blockedSet().has(row.name)) : rows(),
    );

    const whoisCountry = (row: ClientStat) => row.info?.whois_info?.country || '';
    const whoisCity = (row: ClientStat) => row.info?.whois_info?.city || '';
    const whoisOrg = (row: ClientStat) =>
        row.info?.whois_info?.orgname || row.info?.whois_info?.org || '';

    const hasWhoisInfo = (row: ClientStat) =>
        Boolean(whoisOrg(row) || whoisCountry(row) || whoisCity(row));

    const whoisCardItem = (row: ClientStat): StatCardItem => ({
        label: intl.getMessage('whois'),
        layout: 'column',
        value: (
            <span class={s.whoisCell}>
                <Show when={whoisOrg(row)}>
                    <div>{whoisOrg(row)}</div>
                </Show>
                <Show when={whoisCountry(row) || whoisCity(row)}>
                    <div class={s.whoisSub}>
                        <span>{whoisCountry(row)}</span>
                        <Show when={whoisCountry(row) && whoisCity(row)}>
                            <span class={s.whoisDivider} />
                        </Show>
                        <span>{whoisCity(row)}</span>
                    </div>
                </Show>
            </span>
        ),
    });

    const columns = (): TableColumn<ClientStat>[] => [
        {
            key: 'name',
            header: { text: intl.getMessage('name_table_header') },
            accessor: (row) => row.info?.name || row.name,
            sortable: true,
            render: (_v, row) => (
                <span
                    class={cn(theme.text.t3, theme.text.condenced, s.nameCell)}
                    title={row.info?.name}
                    data-testid="client-name-cell"
                >
                    {row.info?.name || '—'}
                </span>
            ),
        },
        {
            key: 'status',
            header: { text: intl.getMessage('status_table_header') },
            accessor: (row) => (isBlocked(row.name) ? 'blocked' : 'unblocked'),
            sortable: false,
            width: 93,
            render: (_v, row) => (
                <Show when={isBlocked(row.name)}>
                    <span class={cn(theme.text.t3, s.blockedStatus)}>
                        {intl.getMessage('blocked')}
                    </span>
                </Show>
            ),
        },
        {
            key: 'queries',
            header: { text: intl.getMessage('queries') },
            accessor: (row) => row.count,
            sortable: true,
            sortFn: (a: number, b: number) => a - b,
            render: (_v, row) => (
                <CountWithPercent
                    count={row.count}
                    total={statsState.numDnsQueries}
                    queryLogSearch={row.name}
                />
            ),
        },
        {
            key: 'ip',
            header: { text: intl.getMessage('ip_address') },
            accessor: (row) => row.name,
            sortable: true,
            render: (_v, row) => (
                <Link
                    to={RoutePath.QueryLog}
                    query={{ search: `"${row.name}"` }}
                    class={cn(theme.text.t3, theme.text.condenced, s.ipCellLink)}
                    title={row.name}
                    data-testid="client-ip-cell"
                >
                    {row.name}
                </Link>
            ),
        },
        {
            key: 'whois',
            header: { text: intl.getMessage('whois') },
            accessor: (row) => `${whoisOrg(row)} ${whoisCountry(row)}`,
            width: 219,
            render: (_v, row) => (
                <span class={cn(theme.text.t3, s.whoisCell)}>
                    <div>{whoisOrg(row) || '—'}</div>
                    <Show when={whoisCountry(row) || whoisCity(row)}>
                        <div class={s.whoisSub}>
                            <span>{whoisCountry(row)}</span>
                            <Show when={whoisCountry(row) && whoisCity(row)}>
                                <span class={s.whoisDivider} />
                            </Show>
                            <span>{whoisCity(row)}</span>
                        </div>
                    </Show>
                </span>
            ),
        },
        {
            key: 'actions',
            header: { text: '' },
            width: 48,
            class: s.actionsCell,
            render: (_v, row) => (
                <Dropdown
                    wrapClass={s.actionsDropdown}
                    position="bottomRight"
                    noIcon
                    open={openMenuClient() === row.name}
                    onOpenChange={(isOpen: boolean) => setOpenMenuClient(isOpen ? row.name : null)}
                    menu={
                        <div class={s.protectionMenu}>
                            <Show
                                when={isBlocked(row.name)}
                                fallback={
                                    <div
                                        class={cn(
                                            theme.text.t2,
                                            theme.text.condenced,
                                            s.protectionMenuItem,
                                            s.protectionMenuItemRed,
                                        )}
                                        data-testid="client-block-menu-item"
                                        onClick={() => openClientConfirmDialog(row.name, 'block')}
                                    >
                                        {intl.getMessage('block_client')}
                                    </div>
                                }
                            >
                                <div
                                    class={cn(
                                        theme.text.t2,
                                        theme.text.condenced,
                                        theme.dropdown.item,
                                        s.protectionMenuItem,
                                    )}
                                    data-testid="client-unblock-menu-item"
                                    onClick={() => openClientConfirmDialog(row.name, 'unblock')}
                                >
                                    {intl.getMessage('unblock_client')}
                                </div>
                            </Show>
                        </div>
                    }
                >
                    <button
                        type="button"
                        class={s.actionButton}
                        data-testid="client-action-button"
                        aria-label={`${intl.getMessage('aria_actions')}: ${row.info?.name || row.name}`}
                    >
                        <Icon icon="bullets" />
                    </button>
                </Dropdown>
            ),
        },
    ];

    const mobileSortOptions = (): IOption<string>[] => [
        { value: 'name:asc', label: intl.getMessage('sort_name_asc') },
        { value: 'name:desc', label: intl.getMessage('sort_name_desc') },
        { value: 'queries:desc', label: intl.getMessage('sort_queries_desc') },
        { value: 'queries:asc', label: intl.getMessage('sort_queries_asc') },
        { value: 'ip:desc', label: intl.getMessage('sort_ip_desc') },
        { value: 'ip:asc', label: intl.getMessage('sort_ip_asc') },
    ];

    const mobileCard = (row: ClientStat) => {
        const blocked = isBlocked(row.name);
        return (
            <StatMobileCard
                title={
                    <span class={s.mobileTitle} data-testid="client-name-cell">
                        {row.info?.name || row.name}
                    </span>
                }
                status={
                    <Show when={blocked}>
                        <span
                            class={cn(theme.text.t3, s.mobileBlocked)}
                            data-testid="client-blocked-status"
                        >
                            {intl.getMessage('blocked')}
                        </span>
                    </Show>
                }
                items={[
                    {
                        label: intl.getMessage('queries'),
                        value: (
                            <CountWithPercent
                                count={row.count}
                                total={statsState.numDnsQueries}
                                queryLogSearch={row.name}
                            />
                        ),
                        progress: computePercent(row.count, statsState.numDnsQueries),
                    },
                    {
                        label: intl.getMessage('ip_address'),
                        value: (
                            <Link
                                to={RoutePath.QueryLog}
                                query={{ search: `"${row.name}"` }}
                                class={cn(theme.text.t3, theme.text.condenced, s.ipCellLink)}
                                title={row.name}
                            >
                                {row.name}
                            </Link>
                        ),
                    },
                    ...(hasWhoisInfo(row) ? [whoisCardItem(row)] : []),
                ]}
                actions={
                    <button
                        type="button"
                        class={cn(
                            s.mobileActionButton,
                            blocked ? s.mobileActionButtonGreen : s.mobileActionButtonRed,
                        )}
                        data-testid={blocked ? 'client-unblock-button' : 'client-block-button'}
                        onClick={() => openConfirmDialog(row.name, blocked ? 'unblock' : 'block')}
                    >
                        {blocked
                            ? intl.getMessage('unblock_client')
                            : intl.getMessage('block_client')}
                    </button>
                }
            />
        );
    };

    const filterPill = (
        <button
            type="button"
            class={cn(s.filterPill, showOnlyBlocked() && s.filterPillActive)}
            data-testid="stats-show-blocked-button"
            aria-pressed={showOnlyBlocked()}
            onClick={() => setShowOnlyBlocked((value) => !value)}
        >
            {intl.getMessage('blocked_only')}
            <Show when={showOnlyBlocked()}>
                <Icon icon="cross" class={s.filterPillCross} />
            </Show>
        </button>
    );

    return (
        <>
            <StatsPage<ClientStat>
                title={intl.getMessage('top_clients')}
                rows={visibleRows()}
                columns={columns()}
                getRowId={(row) => row.name}
                defaultSort={{ key: 'queries', direction: 'desc' }}
                loading={statsState.processingStats || accessState.processing}
                emptyText={intl.getMessage('nothing_found')}
                onRefresh={handleRefresh}
                searchTextForRow={(row) => `${row.name} ${row.info?.name ?? ''}`}
                pageSizeKey={LOCAL_STORAGE_KEYS.TOP_CLIENTS_PAGE_SIZE}
                sortStorageKey={LOCAL_STORAGE_KEYS.TOP_CLIENTS_SORT}
                mobileSortOptions={mobileSortOptions()}
                renderMobileCard={(row) => mobileCard(row)}
                controlsBefore={filterPill}
                controlsUnderTitle
                controlsAfter={
                    <PlusButton onClick={() => handleAddClient()} testId="stats-add-client-button">
                        {intl.getMessage('clients_add')}
                    </PlusButton>
                }
            />

            <ClientBlockConfirmDialog
                state={confirmDialog()}
                onClose={closeConfirmDialog}
                onConfirm={handleConfirm}
            />
        </>
    );
};
