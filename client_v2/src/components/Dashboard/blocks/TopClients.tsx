import React, { useState, useRef, useEffect, useMemo } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { RootState } from 'panel/initialState';

import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { formatNumber, formatCompactNumber } from 'panel/helpers/helpers';
import { addErrorToast, addSuccessToast } from 'panel/actions/toasts';
import { apiClient } from 'panel/api/Api';
import { getAccessList } from 'panel/actions/access';
import theme from 'panel/lib/theme';
import cn from 'clsx';

import s from '../Dashboard.module.pcss';

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

type SortField = 'name' | 'count';
type SortDirection = 'asc' | 'desc';

export const TopClients = ({ topClients, numDnsQueries }: Props) => {
    const dispatch = useDispatch();
    const isMountedRef = useRef(true);
    const disallowedClientsStr = useSelector((state: RootState) => state.access?.disallowed_clients || '');
    const disallowedClientsList = disallowedClientsStr ? disallowedClientsStr.split('\n').filter(Boolean) : [];

    const [confirmDialog, setConfirmDialog] = useState<{
        open: boolean;
        client: string;
        action: 'block' | 'unblock';
    }>({ open: false, client: '', action: 'block' });
    const [openMenuClient, setOpenMenuClient] = useState<string | null>(null);
    const [sortField, setSortField] = useState<SortField>('count');
    const [sortDirection, setSortDirection] = useState<SortDirection>('desc');

    const isClientBlocked = (clientName: string) => disallowedClientsList.includes(clientName);

    useEffect(() => {
        return () => {
            isMountedRef.current = false;
        };
    }, []);

    const handleBlockClient = async (clientIp: string) => {
        try {
            const accessList = await apiClient.getAccessList();
            const disallowedClients = accessList.disallowed_clients || [];

            if (disallowedClients.includes(clientIp)) {
                dispatch(addErrorToast({ error: new Error(intl.getMessage('client_already_blocked', { ip: clientIp })) }));
                if (isMountedRef.current) {
                    setConfirmDialog({ open: false, client: '', action: 'block' });
                }
                return;
            }

            await apiClient.setAccessList({
                ...accessList,
                disallowed_clients: [...disallowedClients, clientIp],
            });

            dispatch(addSuccessToast(intl.getMessage('client_blocked_flash')));
            dispatch(getAccessList());
        } catch (error) {
            dispatch(addErrorToast({ error }));
        }
        if (isMountedRef.current) {
            setConfirmDialog({ open: false, client: '', action: 'block' });
        }
    };

    const handleUnblockClient = async (clientIp: string) => {
        try {
            const accessList = await apiClient.getAccessList();
            const disallowedClients = (accessList.disallowed_clients || []).filter(
                (c: string) => c !== clientIp
            );

            await apiClient.setAccessList({
                ...accessList,
                disallowed_clients: disallowedClients,
            });

            dispatch(addSuccessToast(intl.getMessage('client_unblocked_flash')));
            dispatch(getAccessList());
        } catch (error) {
            dispatch(addErrorToast({ error }));
        }
        if (isMountedRef.current) {
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
            <div className={s.protectionMenu}>
                {isBlocked ? (
                    <div
                        className={cn(theme.text.t2, theme.text.condenced, s.protectionMenuItem)}
                        onClick={() => openConfirmDialog(client.name, 'unblock')}
                    >
                        {intl.getMessage('unblock_client')}
                    </div>
                ) : (
                    <div
                        className={cn(
                            theme.text.t2,
                            theme.text.condenced,
                            s.protectionMenuItem,
                            s.protectionMenuItemRed
                        )}
                        onClick={() => openConfirmDialog(client.name, 'block')}
                    >
                        {intl.getMessage('block_client')}
                    </div>
                )}
            </div>
        );
    };

    const hasStats = topClients.length > 0;

    const sortedClients = useMemo(() => {
        return [...topClients].sort((a, b) => {
            const modifier = sortDirection === 'asc' ? 1 : -1;
            if (sortField === 'name') {
                return a.name.localeCompare(b.name) * modifier;
            }
            return (a.count - b.count) * modifier;
        });
    }, [topClients, sortField, sortDirection]);

    const handleSort = (field: SortField) => {
        if (sortField === field) {
            setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
        } else {
            setSortField(field);
            setSortDirection(field === 'name' ? 'asc' : 'desc');
        }
    };

    return (
        <div className={s.card}>
            <div className={s.cardHeader}>
                <div className={cn(theme.title.h5, s.cardTitle)}>{intl.getMessage('top_clients')}</div>

                {hasStats && (
                    <div className={cn(theme.text.t3, s.cardSubtitle)}>
                        {intl.getPlural('queries_total', formatCompactNumber(numDnsQueries))}
                    </div>
                )}
            </div>

            {hasStats && (
                <div className={cn(theme.text.t3, theme.text.semibold, s.tableHeader)}>
                    <span
                        className={s.sortableHeader}
                        onClick={() => handleSort('name')}
                    >
                        {intl.getMessage('table_client')}
                        {sortField === 'name' ? (
                            <Icon
                                icon="arrow_bottom"
                                className={cn(s.sortIcon, sortDirection === 'asc' && s.sortIconAsc)}
                            />
                        ) : (
                            <span className={s.sortDash}>—</span>
                        )}
                    </span>
                    <span
                        className={s.sortableHeader}
                        onClick={() => handleSort('count')}
                    >
                        {intl.getMessage('queries')}
                        {sortField === 'count' ? (
                            <Icon
                                icon="arrow_bottom"
                                className={cn(s.sortIcon, sortDirection === 'asc' && s.sortIconAsc)}
                            />
                        ) : (
                            <span className={s.sortDash}>—</span>
                        )}
                    </span>
                </div>
            )}

            <div className={s.tableRows}>
                {hasStats ? (
                    sortedClients.slice(0, 10).map((client) => {
                        const percent = numDnsQueries > 0 ? (client.count / numDnsQueries) * 100 : 0;
                        const isBlocked = isClientBlocked(client.name);

                        return (
                            <div key={client.name} className={s.clientRow}>
                                <div className={s.clientInfo}>
                                    <div className={cn(theme.text.t3, theme.text.condenced, s.clientIp)}>
                                        {client.info ? (
                                            <Icon icon="location" className={s.tableRowIcon} />
                                        ) : <div className={s.tableRowDot}></div>}

                                        {client.name}
                                    </div>
                                </div>

                                <div className={s.tableRowRight}>
                                    <Dropdown
                                        trigger="hover"
                                        position="top"
                                        noIcon
                                        disableAnimation
                                        overlayClassName={s.queryTooltipOverlay}
                                        menu={
                                            <div className={s.queryTooltip}>
                                                {formatNumber(client.count)} {intl.getMessage('queries').toLowerCase()}
                                            </div>
                                        }
                                    >
                                        <div className={cn(
                                            theme.text.t3,
                                            theme.text.condenced,
                                            s.queryCount,
                                            s.queryCountHover
                                        )}>
                                            {formatCompactNumber(client.count)}

                                            <div className={cn(theme.text.t3, theme.text.condenced, s.queryPercent)}>
                                                ({percent.toFixed(1)}%)
                                            </div>
                                        </div>
                                    </Dropdown>

                                    <div className={s.queryBar}>
                                        <div
                                            className={s.queryBarFill}
                                            style={{ width: `${percent}%` }}
                                        />
                                    </div>
                                    <Dropdown
                                        menu={getClientMenu(client)}
                                        trigger="click"
                                        position="bottomRight"
                                        noIcon
                                        open={openMenuClient === client.name}
                                        onOpenChange={(isOpen) => setOpenMenuClient(isOpen ? client.name : null)}
                                    >
                                        <button type="button" className={s.actionButton}>
                                            <Icon icon="bullets" />
                                        </button>
                                    </Dropdown>
                                </div>

                                <div className={s.tableRowInfo}>
                                    {client.info?.name && (
                                        <div className={cn(
                                            theme.text.t4,
                                            theme.text.condenced,
                                            s.clientName
                                        )}>
                                            {client.info.name}
                                        </div>
                                    )}
                                    {isBlocked && (
                                        <div className={cn(theme.text.t4, theme.text.condenced, s.clientBlocked)}>
                                            {intl.getMessage('blocked')}
                                        </div>
                                    )}
                                </div>
                            </div>
                        );
                    })
                ) : (
                    <div className={s.emptyState}>
                        <Icon icon="not_found_search" className={s.emptyStateIcon} />

                        <div className={s.emptyStateText}>{intl.getMessage('no_stats_yet')}</div>
                    </div>
                )}

                {confirmDialog.open && (
                    <ConfirmDialog
                        onClose={() => setConfirmDialog({ open: false, client: '', action: 'block' })}
                        title={intl.getMessage(
                            confirmDialog.action === 'block' ? 'confirm_client_block_title' : 'confirm_client_unblock_title',
                            { ip: confirmDialog.client }
                        )}
                        text={intl.getMessage(
                            confirmDialog.action === 'block' ? 'confirm_client_block_desc' : 'confirm_client_unblock_desc',
                            { ip: confirmDialog.client }
                        )}
                        buttonText={intl.getMessage(confirmDialog.action === 'block' ? 'block' : 'unblock')}
                        cancelText={intl.getMessage('cancel')}
                        buttonVariant={confirmDialog.action === 'block' ? 'danger' : 'primary'}
                        onConfirm={() => {
                            if (confirmDialog.action === 'block') {
                                handleBlockClient(confirmDialog.client);
                            } else {
                                handleUnblockClient(confirmDialog.client);
                            }
                        }}
                    />
                )}
            </div>
        </div>
    );
};
