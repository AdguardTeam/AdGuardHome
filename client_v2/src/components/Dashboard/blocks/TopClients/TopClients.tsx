import { Show, For, createSignal, createMemo, onCleanup } from 'solid-js';
import { useIsDesktop } from 'panel/helpers/useMediaQuery';
import { MOBILE_TABLE_MAX_ROWS } from 'panel/helpers/constants';

import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Tooltip } from 'panel/common/ui/Tooltip';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { formatNumber, formatCompactNumber } from 'panel/helpers/helpers';
import { addErrorToast } from 'panel/stores/toasts';
import { accessState, toggleClientBlock } from 'panel/stores/access';
import theme from 'panel/lib/theme';
import cn from 'clsx';
import { useSortedData } from '../../hooks/useSortedData';
import { SortableTableHeader } from '../SortableTableHeader';
import { EmptyState } from '../EmptyState';

import s from './TopClients.module.pcss';

type ClientInfo = {
    name: string;
    count: number;
    info?: {
        name?: string;
        whois_info?: {
            orgname?: string;
            country?: string;
        };
        disallowed?: boolean;
    };
};

type Props = {
    topClients: ClientInfo[];
    numDnsQueries: number;
};

export const TopClients = (props: Props) => {
    let isMounted = true;
    onCleanup(() => {
        isMounted = false;
    });

    const disallowedClientsList = createMemo(() => {
        const str = accessState.disallowed_clients || '';
        return str ? str.split('\n').filter(Boolean) : [];
    });

    const [confirmDialog, setConfirmDialog] = createSignal<{
        open: boolean;
        client: string;
        action: 'block' | 'unblock';
    }>({ open: false, client: '', action: 'block' });
    const [openMenuClient, setOpenMenuClient] = createSignal<string | null>(null);

    const isDesktop = useIsDesktop();
    const {
        sortedData: sortedClients,
        sortField,
        sortDirection,
        handleSort,
    } = useSortedData(() => props.topClients);
    const visibleClients = createMemo(() =>
        isDesktop() ? sortedClients() : sortedClients().slice(0, MOBILE_TABLE_MAX_ROWS),
    );

    const isClientBlocked = (clientName: string) => disallowedClientsList().includes(clientName);

    const handleBlockClient = async (clientIp: string) => {
        const disallowedList = accessState.disallowed_clients
            ? accessState.disallowed_clients.split('\n').filter(Boolean)
            : [];
        const isDisallowed = disallowedList.includes(clientIp);
        if (isDisallowed) {
            addErrorToast({
                error: new Error(intl.getMessage('client_already_blocked', { ip: clientIp })),
            });
            if (isMounted) {
                setConfirmDialog({ open: false, client: '', action: 'block' });
            }
            return;
        }
        await toggleClientBlock(clientIp, false, '');
        if (isMounted) {
            setConfirmDialog({ open: false, client: '', action: 'block' });
        }
    };

    const handleUnblockClient = async (clientIp: string) => {
        const disallowedList = accessState.disallowed_clients
            ? accessState.disallowed_clients.split('\n').filter(Boolean)
            : [];
        const isDisallowed = disallowedList.includes(clientIp);
        await toggleClientBlock(clientIp, isDisallowed, isDisallowed ? clientIp : '');
        if (isMounted) {
            setConfirmDialog({ open: false, client: '', action: 'unblock' });
        }
    };

    const openConfirmDialog = (client: string, action: 'block' | 'unblock') => {
        setOpenMenuClient(null);
        setConfirmDialog({ open: true, client, action });
    };

    const getClientMenu = (client: ClientInfo) => {
        const isBlocked = isClientBlocked(client.name);

        return (
            <div class={s.protectionMenu}>
                <Show
                    when={isBlocked}
                    fallback={
                        <div
                            class={cn(
                                theme.text.t2,
                                theme.text.condenced,
                                s.protectionMenuItem,
                                s.protectionMenuItemRed,
                            )}
                            onClick={() => openConfirmDialog(client.name, 'block')}
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
                        onClick={() => openConfirmDialog(client.name, 'unblock')}
                    >
                        {intl.getMessage('unblock_client')}
                    </div>
                </Show>
            </div>
        );
    };

    const hasStats = createMemo(() => props.topClients.length > 0);

    return (
        <div class={s.card}>
            <div class={s.cardHeader}>
                <div class={cn(theme.title.h5, s.cardTitle)}>{intl.getMessage('top_clients')}</div>
            </div>

            <Show when={hasStats()}>
                <SortableTableHeader
                    nameLabel={intl.getMessage('table_client')}
                    countLabel={intl.getMessage('queries')}
                    sortField={sortField()}
                    sortDirection={sortDirection()}
                    onSort={handleSort}
                />
            </Show>

            <div class={s.tableRows}>
                <Show when={hasStats()} fallback={<EmptyState />}>
                    <For each={visibleClients()}>
                        {(client) => {
                            const percent = createMemo(() =>
                                props.numDnsQueries > 0
                                    ? (client.count / props.numDnsQueries) * 100
                                    : 0,
                            );
                            const isBlocked = isClientBlocked(client.name);

                            return (
                                <div class={s.clientRow}>
                                    <div class={s.clientInfo}>
                                        <div
                                            class={cn(
                                                theme.text.t3,
                                                theme.text.condenced,
                                                s.clientIp,
                                            )}
                                        >
                                            <Show
                                                when={client.info}
                                                fallback={<div class={s.tableRowDot} />}
                                            >
                                                <Icon icon="location" class={s.tableRowIcon} />
                                            </Show>

                                            {client.name}
                                        </div>
                                    </div>

                                    <div class={s.tableRowRight}>
                                        <Show when={isDesktop()}>
                                            <div class={s.dropdowWrapper}>
                                                <Tooltip
                                                    position="top"
                                                    overlayClass={s.queryTooltipOverlay}
                                                    content={
                                                        <div class={s.queryTooltip}>
                                                            {intl.getMessage('queries_tooltip', {
                                                                value: formatNumber(client.count),
                                                            })}
                                                        </div>
                                                    }
                                                >
                                                    <div
                                                        class={cn(
                                                            theme.text.t3,
                                                            theme.text.condenced,
                                                            s.queryCount,
                                                            s.queryCountHover,
                                                        )}
                                                    >
                                                        {formatCompactNumber(client.count)}

                                                        <div
                                                            class={cn(
                                                                theme.text.t3,
                                                                theme.text.condenced,
                                                                s.queryPercent,
                                                            )}
                                                        >
                                                            ({percent().toFixed(1)}%)
                                                        </div>
                                                    </div>
                                                </Tooltip>
                                            </div>
                                        </Show>

                                        <Show when={isDesktop()}>
                                            <div class={s.queryBar}>
                                                <div
                                                    class={s.queryBarFill}
                                                    style={{ width: `${percent()}%` }}
                                                />
                                            </div>
                                        </Show>

                                        <div class={s.dropdownWrapper}>
                                            <Dropdown
                                                wrapClass={s.clientActionsDropdown}
                                                menu={getClientMenu(client)}
                                                position="bottomRight"
                                                noIcon
                                                open={openMenuClient() === client.name}
                                                onOpenChange={(isOpen: boolean) =>
                                                    setOpenMenuClient(isOpen ? client.name : null)
                                                }
                                            >
                                                <button type="button" class={s.actionButton}>
                                                    <Icon icon="bullets" />
                                                </button>
                                            </Dropdown>
                                        </div>

                                        <Show when={isBlocked}>
                                            <div
                                                class={cn(
                                                    theme.text.t4,
                                                    theme.text.condenced,
                                                    s.clientBlocked,
                                                )}
                                            >
                                                {intl.getMessage('blocked')}
                                            </div>
                                        </Show>
                                    </div>

                                    <div class={s.tableRowInfo}>
                                        <Show when={client.info?.name}>
                                            <div
                                                class={cn(
                                                    theme.text.t4,
                                                    theme.text.condenced,
                                                    s.clientName,
                                                )}
                                            >
                                                {client.info.name}
                                            </div>
                                        </Show>
                                        <Show when={isBlocked}>
                                            <div
                                                class={cn(
                                                    theme.text.t4,
                                                    theme.text.condenced,
                                                    s.clientBlocked,
                                                )}
                                            >
                                                {intl.getMessage('blocked')}
                                            </div>
                                        </Show>
                                        <div class={s.tableRowQueriesInfo}>
                                            <div
                                                class={cn(
                                                    theme.text.t3,
                                                    theme.text.condenced,
                                                    s.queryCount,
                                                    s.queryCountHover,
                                                )}
                                            >
                                                {formatCompactNumber(client.count)}

                                                <div
                                                    class={cn(
                                                        theme.text.t3,
                                                        theme.text.condenced,
                                                        s.queryPercent,
                                                    )}
                                                >
                                                    ({percent().toFixed(1)}%)
                                                </div>
                                            </div>

                                            <div class={s.queryBar}>
                                                <div
                                                    class={s.queryBarFill}
                                                    style={{ width: `${percent()}%` }}
                                                />
                                            </div>
                                        </div>

                                        <div class={s.tableRowActions}>{getClientMenu(client)}</div>
                                    </div>
                                </div>
                            );
                        }}
                    </For>
                </Show>

                <Show when={confirmDialog().open}>
                    {(() => {
                        const dialog = confirmDialog();
                        const isBlock = dialog.action === 'block';

                        return (
                            <ConfirmDialog
                                onClose={() =>
                                    setConfirmDialog({ open: false, client: '', action: 'block' })
                                }
                                title={
                                    isBlock
                                        ? intl.getMessage('confirm_client_block_title', {
                                              ip: dialog.client,
                                          })
                                        : intl.getMessage('confirm_client_unblock_title', {
                                              ip: dialog.client,
                                          })
                                }
                                text={
                                    isBlock
                                        ? intl.getMessage('confirm_client_block_desc', {
                                              ip: dialog.client,
                                          })
                                        : intl.getMessage('confirm_client_unblock_desc', {
                                              ip: dialog.client,
                                          })
                                }
                                buttonText={
                                    isBlock ? intl.getMessage('block') : intl.getMessage('unblock')
                                }
                                cancelText={intl.getMessage('cancel')}
                                buttonVariant={isBlock ? 'danger' : 'primary'}
                                onConfirm={() => {
                                    if (isBlock) {
                                        handleBlockClient(dialog.client);
                                    } else {
                                        handleUnblockClient(dialog.client);
                                    }
                                }}
                            />
                        );
                    })()}
                </Show>
            </div>
        </div>
    );
};
