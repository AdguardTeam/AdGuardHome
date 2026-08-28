import { Show, createMemo, createSignal, onMount } from 'solid-js';
import cn from 'clsx';
import { useNavigate, useSearchParams } from '@solidjs/router';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import type { TableColumn } from 'panel/common/ui/Table';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { Icon } from 'panel/common/ui/Icon';
import { statsState, getStats } from 'panel/stores/stats';
import { accessState, getAccessList, toggleClientBlock } from 'panel/stores/access';
import { addErrorToast } from 'panel/stores/toasts';
import { initClientForm } from 'panel/stores/clientForm';
import { formatCompactNumber } from 'panel/helpers/helpers';
import { computePercent, resolveStatsPeriod } from 'panel/helpers/statistics';
import { LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import { PlusButton } from 'panel/common/ui/PlusButton';
import { Paths } from 'panel/components/Routes/Paths';
import { StatsPage } from './StatsPage';
import { StatMobileCard } from './blocks/StatMobileCard/StatMobileCard';

import s from './TopClientsPage.module.pcss';

type ClientStat = {
    name: string;
    count: number;
    info?: { name?: string; whois_info?: Record<string, string> };
};

export const TopClientsPage = () => {
    const navigate = useNavigate();
    const [searchParams] = useSearchParams<{ period?: string }>();

    onMount(() => {
        getStats(resolveStatsPeriod(searchParams, statsState.interval));
        getAccessList();
    });

    const handleAddClient = () => {
        initClientForm(null);
        navigate(Paths.ClientsAdd);
    };

    const handleRefresh = () => {
        getStats(resolveStatsPeriod(searchParams, statsState.interval));
        getAccessList();
    };

    const disallowed = createMemo(() => {
        const str = accessState.disallowed_clients || '';
        return str ? str.split('\n').filter(Boolean) : [];
    });

    const isBlocked = (clientIp: string) => disallowed().includes(clientIp);

    // Pre-sorted by identifier ascending: Table's stable sort keeps this order
    // as the tie-break when counts are equal.
    const rows = createMemo<ClientStat[]>(() =>
        [...statsState.topClients].toSorted((a, b) =>
            (a.info?.name ?? a.name).localeCompare(b.info?.name ?? b.name),
        ),
    );

    const [confirmDialog, setConfirmDialog] = createSignal<{
        open: boolean;
        client: string;
        action: 'block' | 'unblock';
    }>({ open: false, client: '', action: 'block' });

    const openConfirmDialog = (client: string, action: 'block' | 'unblock') =>
        setConfirmDialog({ open: true, client, action });

    const closeConfirmDialog = () => setConfirmDialog({ open: false, client: '', action: 'block' });

    const handleBlock = async () => {
        const { client } = confirmDialog();
        if (isBlocked(client)) {
            addErrorToast({
                error: new Error(intl.getMessage('client_already_blocked', { ip: client })),
            });
        } else {
            await toggleClientBlock(client, false, '');
        }
        closeConfirmDialog();
    };

    const handleUnblock = async () => {
        const { client } = confirmDialog();
        await toggleClientBlock(client, true, client);
        closeConfirmDialog();
    };

    const whoisCountry = (row: ClientStat) => row.info?.whois_info?.country || '';
    const whoisOrg = (row: ClientStat) =>
        row.info?.whois_info?.orgname || row.info?.whois_info?.org || '';

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
            sortable: true,
            render: (_v, row) => (
                <span class={cn(theme.text.t3, isBlocked(row.name) && s.blockedStatus)}>
                    {intl.getMessage(isBlocked(row.name) ? 'blocked' : 'unblocked')}
                </span>
            ),
        },
        {
            key: 'queries',
            header: { text: intl.getMessage('queries') },
            accessor: (row) => row.count,
            sortable: true,
            sortFn: (a: number, b: number) => a - b,
            render: (_v, row) => (
                <span class={cn(theme.text.t3, theme.text.condenced, s.countCell)}>
                    {formatCompactNumber(row.count)}
                    <span class={s.percent} data-testid="stats-percent">
                        ({computePercent(row.count, statsState.numDnsQueries).toFixed(1)}%)
                    </span>
                </span>
            ),
        },
        {
            key: 'ip',
            header: { text: intl.getMessage('ip_address') },
            accessor: (row) => row.name,
            sortable: true,
            render: (_v, row) => (
                <span
                    class={cn(theme.text.t3, theme.text.condenced, s.ipCell)}
                    data-testid="client-ip-cell"
                >
                    {row.name}
                </span>
            ),
        },
        {
            key: 'whois',
            header: { text: intl.getMessage('whois') },
            accessor: (row) => `${whoisOrg(row)} ${whoisCountry(row)}`,
            render: (_v, row) => (
                <span class={cn(theme.text.t3, s.whoisCell)}>
                    <div>{whoisOrg(row) || '—'}</div>
                    <div>{whoisCountry(row)}</div>
                </span>
            ),
        },
        {
            key: 'actions',
            header: { text: intl.getMessage('actions') },
            fitContent: true,
            render: (_v, row) => (
                <Dropdown
                    wrapClass={s.actionsDropdown}
                    position="bottomRight"
                    noIcon
                    menu={
                        <div class={s.actionMenu}>
                            <Show
                                when={isBlocked(row.name)}
                                fallback={
                                    <div
                                        class={cn(
                                            theme.text.t2,
                                            theme.text.condenced,
                                            s.menuItem,
                                            s.menuItemRed,
                                        )}
                                        data-testid="client-block-menu-item"
                                        onClick={() => openConfirmDialog(row.name, 'block')}
                                    >
                                        {intl.getMessage('block_client')}
                                    </div>
                                }
                            >
                                <div
                                    class={cn(theme.text.t2, theme.text.condenced, s.menuItem)}
                                    data-testid="client-unblock-menu-item"
                                    onClick={() => openConfirmDialog(row.name, 'unblock')}
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
                        aria-label={`${intl.getMessage('actions')}: ${row.info?.name || row.name}`}
                    >
                        <Icon icon="bullets" />
                    </button>
                </Dropdown>
            ),
        },
    ];

    const mobileCard = (row: ClientStat) => (
        <StatMobileCard
            items={[
                {
                    label: intl.getMessage('name_table_header'),
                    value: <span>{row.info?.name || '—'}</span>,
                },
                {
                    label: intl.getMessage('status_table_header'),
                    value: (
                        <span class={cn(isBlocked(row.name) && s.blockedStatus)}>
                            {isBlocked(row.name)
                                ? intl.getMessage('blocked')
                                : intl.getMessage('unblocked')}
                        </span>
                    ),
                },
                {
                    label: intl.getMessage('queries'),
                    value: (
                        <span class={s.countCell}>
                            {formatCompactNumber(row.count)}
                            <span class={s.percent} data-testid="stats-percent">
                                ({computePercent(row.count, statsState.numDnsQueries).toFixed(1)}
                                %)
                            </span>
                        </span>
                    ),
                },
                {
                    label: intl.getMessage('ip_address'),
                    value: <span>{row.name}</span>,
                },
                {
                    label: intl.getMessage('whois'),
                    value: (
                        <span class={s.whoisCell}>
                            <div>{whoisOrg(row) || '—'}</div>
                            <div>{whoisCountry(row)}</div>
                        </span>
                    ),
                },
            ]}
            actions={
                <button
                    type="button"
                    class={cn(theme.link.link, isBlocked(row.name) ? s.unblockLink : s.blockLink)}
                    onClick={() =>
                        openConfirmDialog(row.name, isBlocked(row.name) ? 'unblock' : 'block')
                    }
                >
                    {intl.getMessage(isBlocked(row.name) ? 'unblock_client' : 'block_client')}
                </button>
            }
        />
    );

    return (
        <>
            <StatsPage<ClientStat>
                title={intl.getMessage('top_clients')}
                rows={rows()}
                columns={columns()}
                getRowId={(row) => row.name}
                defaultSort={{ key: 'queries', direction: 'desc' }}
                loading={statsState.processingStats || accessState.processing}
                emptyText={intl.getMessage('stats_table_empty')}
                onRefresh={handleRefresh}
                searchTextForRow={(row) => `${row.name} ${row.info?.name ?? ''}`}
                pageSizeKey={LOCAL_STORAGE_KEYS.TOP_CLIENTS_PAGE_SIZE}
                sortStorageKey={LOCAL_STORAGE_KEYS.TOP_CLIENTS_SORT}
                renderMobileCard={(row) => mobileCard(row)}
            >
                <div class={s.addClientRow}>
                    <PlusButton
                        class={s.addClientButton}
                        onClick={() => handleAddClient()}
                        testId="stats-add-client-button"
                    >
                        {intl.getMessage('clients_add')}
                    </PlusButton>
                </div>
            </StatsPage>

            <Show when={confirmDialog().open}>
                <ConfirmDialog
                    title={intl.getMessage(
                        confirmDialog().action === 'block' ? 'block_client' : 'unblock_client',
                    )}
                    text={intl.getMessage(
                        confirmDialog().action === 'block' ? 'block_client' : 'unblock_client',
                    )}
                    buttonText={intl.getMessage(
                        confirmDialog().action === 'block' ? 'block_client' : 'unblock_client',
                    )}
                    cancelText={intl.getMessage('cancel_btn')}
                    submitTestId="confirm-dialog-submit"
                    onClose={closeConfirmDialog}
                    onConfirm={() =>
                        confirmDialog().action === 'block' ? handleBlock() : handleUnblock()
                    }
                />
            </Show>
        </>
    );
};
