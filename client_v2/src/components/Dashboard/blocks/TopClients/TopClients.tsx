import { Show, For, createSignal, createMemo } from 'solid-js';
import { useIsDesktop } from 'panel/hooks/useMediaQuery';

import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Dropdown } from 'panel/common/ui/Dropdown';
import {
    ClientBlockConfirmDialog,
    ClientBlockMenuItem,
    useClientBlockConfirm,
} from 'panel/common/ui/ClientBlockConfirm';
import { Link } from 'panel/common/ui/Link';
import { RoutePath } from 'panel/components/Routes/Paths';
import { formatCompactNumber } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import cn from 'clsx';
import { useSortedData, TOP_CLIENTS_VISIBLE_ITEMS } from '../../hooks/useSortedData';
import { TableHeader } from '../TableHeader';
import { EmptyState } from '../EmptyState';
import { CardFooter } from '../CardFooter';
import { ClientTooltip } from '../ClientTooltip';
import { RowTooltip } from '../RowTooltip';

import s from './TopClients.module.pcss';

import type { ClientFindSubEntry } from 'panel/api/model/clientFindSubEntry';
import type { ClientBlockAction } from 'panel/common/ui/ClientBlockConfirm';

type ClientInfo = {
    name: string;
    count: number;
    info?: ClientFindSubEntry;
};

type Props = {
    topClients: ClientInfo[];
    numDnsQueries: number;
    period?: number;
};

/** The row is a Query Log link, so the block/unblock menu must not follow it. */
const preventRowLink = (e: MouseEvent) => e.preventDefault();

export const TopClients = (props: Props) => {
    const {
        confirmState: confirmDialog,
        isClientBlocked,
        openConfirmDialog,
        closeConfirmDialog,
        handleConfirm,
    } = useClientBlockConfirm();

    const [openMenuClient, setOpenMenuClient] = createSignal<string | null>(null);

    const isDesktop = useIsDesktop();
    const { sortedData: sortedClients, hasMore } = useSortedData(
        () => props.topClients,
        TOP_CLIENTS_VISIBLE_ITEMS,
    );

    const openClientConfirmDialog = (client: string, action: ClientBlockAction) => {
        setOpenMenuClient(null);
        openConfirmDialog(client, action);
    };

    const getClientMenu = (client: ClientInfo) => {
        // A memo, not a plain read: the row mapper is untracked, so the menu
        // must track the access list itself to follow a block/unblock.
        const blocked = createMemo(() => isClientBlocked(client.name));

        return (
            <div class={s.protectionMenu}>
                <ClientBlockMenuItem
                    action={blocked() ? 'unblock' : 'block'}
                    class={cn(
                        s.tableRowMenuItem,
                        blocked() ? s.tableRowMenuItemGreen : s.tableRowMenuItemRed,
                    )}
                    onClick={(action) => openClientConfirmDialog(client.name, action)}
                />
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
                <TableHeader
                    nameLabel={intl.getMessage('table_client')}
                    countLabel={intl.getMessage('queries')}
                />
            </Show>

            <div class={s.tableRows}>
                <Show when={hasStats()} fallback={<EmptyState />}>
                    <For each={sortedClients()}>
                        {(client) => {
                            const percent = createMemo(() =>
                                props.numDnsQueries > 0
                                    ? (client.count / props.numDnsQueries) * 100
                                    : 0,
                            );

                            return (
                                <RowTooltip
                                    content={
                                        <ClientTooltip
                                            address={client.name}
                                            whoisInfo={client.info?.whois_info}
                                            blocked={isClientBlocked(client.name)}
                                        />
                                    }
                                >
                                    <Link
                                        to={RoutePath.QueryLog}
                                        query={{ search: `"${client.name}"` }}
                                        class={cn(s.clientRow, s.clientRowLink)}
                                        data-testid="top-client-row"
                                    >
                                        <div class={s.clientInfo}>
                                            <div
                                                class={cn(
                                                    theme.text.t3,
                                                    theme.text.condenced,
                                                    s.clientIp,
                                                )}
                                            >
                                                <Show
                                                    when={isClientBlocked(client.name)}
                                                    fallback={
                                                        <Icon icon="wifi" class={s.tableRowIcon} />
                                                    }
                                                >
                                                    <Icon
                                                        icon="wifi_protect"
                                                        class={cn(
                                                            s.tableRowIcon,
                                                            s.tableRowIconDanger,
                                                        )}
                                                    />
                                                </Show>

                                                <span class={s.clientIpText}>{client.name}</span>
                                            </div>
                                        </div>

                                        <div class={s.tableRowRight}>
                                            <Show when={isDesktop()}>
                                                <div class={s.dropdownWrapper}>
                                                    <div
                                                        class={cn(
                                                            theme.text.t3,
                                                            theme.text.condenced,
                                                            s.queryCount,
                                                        )}
                                                    >
                                                        <span
                                                            class={cn(
                                                                theme.text.t3,
                                                                theme.text.condenced,
                                                                s.queryCountLink,
                                                            )}
                                                        >
                                                            {formatCompactNumber(client.count)}
                                                        </span>

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

                                            <div class={s.dropdownWrapper} onClick={preventRowLink}>
                                                <Dropdown
                                                    wrapClass={s.clientActionsDropdown}
                                                    menu={getClientMenu(client)}
                                                    position="bottomRight"
                                                    noIcon
                                                    open={openMenuClient() === client.name}
                                                    onOpenChange={(isOpen: boolean) =>
                                                        setOpenMenuClient(
                                                            isOpen ? client.name : null,
                                                        )
                                                    }
                                                >
                                                    <button type="button" class={s.actionButton}>
                                                        <Icon icon="bullets" />
                                                    </button>
                                                </Dropdown>
                                            </div>
                                        </div>

                                        <div class={s.tableRowInfo}>
                                            <Show
                                                when={client.info?.name}
                                                fallback={
                                                    <div
                                                        data-testid="top-client-name"
                                                        class={cn(
                                                            theme.text.t4,
                                                            theme.text.condenced,
                                                            s.clientName,
                                                        )}
                                                    >
                                                        {intl.getMessage('not_available')}
                                                    </div>
                                                }
                                            >
                                                <div
                                                    data-testid="top-client-name"
                                                    class={cn(
                                                        theme.text.t4,
                                                        theme.text.condenced,
                                                        s.clientName,
                                                    )}
                                                >
                                                    {client.info.name}
                                                </div>
                                            </Show>
                                            <div class={s.tableRowQueriesInfo}>
                                                <div
                                                    class={cn(
                                                        theme.text.t3,
                                                        theme.text.condenced,
                                                        s.queryCount,
                                                    )}
                                                >
                                                    <span
                                                        class={cn(
                                                            theme.text.t3,
                                                            theme.text.condenced,
                                                            s.queryCountLink,
                                                        )}
                                                    >
                                                        {formatCompactNumber(client.count)}
                                                    </span>

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

                                            <div class={s.tableRowActions} onClick={preventRowLink}>
                                                {getClientMenu(client)}
                                            </div>
                                        </div>
                                    </Link>
                                </RowTooltip>
                            );
                        }}
                    </For>
                </Show>
            </div>

            {/* Outside the list so its dialog is not the list's last child,
                which would give the last row a divider back while it is open. */}
            <ClientBlockConfirmDialog
                state={confirmDialog()}
                onClose={closeConfirmDialog}
                onConfirm={handleConfirm}
            />

            <Show when={hasMore()}>
                <CardFooter
                    to={RoutePath.TopClients}
                    testId="show-more-top-clients"
                    query={props.period ? { period: props.period } : undefined}
                />
            </Show>
        </div>
    );
};
