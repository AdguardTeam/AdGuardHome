import React from 'react';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';

import intl from 'panel/common/intl';
import { Filter } from 'panel/helpers/helpers';
import {
    formatLogDate,
    formatLogTime,
    getProtocolName,
    getResponseDetails,
    getStatusClassName,
    getStatusLabel,
    isBlockedReason,
} from '../../helpers';
import { LogEntry, Service } from '../../types';
import { ActionsMenu } from '../ActionsMenu';
import { WhoisInfo } from '../WhoisInfo';

import s from './LogCard.module.pcss';

type Props = {
    entry: LogEntry;
    filters: Filter[];
    services: Service[];
    whitelistFilters: Filter[];
    onRowClick: (entry: LogEntry) => void;
    onBlock: (type: string, domain: string) => void;
    onUnblock: (type: string, domain: string) => void;
    onBlockClient: (type: string, domain: string, client: string) => void;
    onDisallowClient: (ip: string) => void;
};

export const LogCard = ({
    entry,
    filters,
    services,
    whitelistFilters,
    onRowClick,
    onBlock,
    onUnblock,
    onBlockClient,
    onDisallowClient,
}: Props) => {
    const statusLabel = getStatusLabel(entry.reason, entry.originalResponse ?? [], false);
    const statusClassName = getStatusClassName(s, entry.reason);
    const clientName = entry.client_info?.name || '';
    const proto = getProtocolName(entry.client_proto);

    const responseDetails = getResponseDetails({
        elapsedMs: entry.elapsedMs,
        filters,
        reason: entry.reason,
        rules: entry.rules,
        serviceName: entry.service_name,
        services,
        whitelistFilters,
    });

    return (
        <div className={s.card} onClick={() => onRowClick(entry)}>
            <div className={s.row}>
                {entry.answer_dnssec && (
                    <Icon icon="lock" color="green" className={s.icon} />
                )}
                {entry.tracker && (
                    <Icon icon="tracking" color="green" className={s.icon} />
                )}

                <span className={cn(s.domain, theme.text.t2, theme.text.medium)} title={entry.domain}>
                    {entry.domain}
                </span>

                <div className={s.actions} onClick={(e) => e.stopPropagation()}>
                    <ActionsMenu
                        domain={entry.domain}
                        client={entry.client}
                        onBlock={onBlock}
                        onUnblock={onUnblock}
                        onBlockClient={onBlockClient}
                        onDisallowClient={() => onDisallowClient(entry.client)}
                        isBlocked={isBlockedReason(entry.reason)}
                    />
                </div>
            </div>

            <div className={s.row}>
                <span className={theme.text.t3}>
                    {intl.getMessage('type_value', { value: entry.type })}, {proto}
                </span>
            </div>

            <div className={s.row}>
                <span className={theme.text.t3}>
                    {formatLogTime(entry.time)}
                </span>
                <span className={cn(theme.text.t4, s.secondary)}>
                    {formatLogDate(entry.time)}
                </span>
            </div>

            <div className={s.row}>
                <span
                    className={cn(
                        s.status,
                        statusClassName,
                        theme.text.t3,
                    )}
                >
                    {statusLabel}
                </span>
                <span className={cn(s.secondary, theme.text.t4)}>{responseDetails}</span>
            </div>

            <div className={cn(s.row, s.row_client)} onClick={(e) => e.stopPropagation()}>
                <span className={cn(s.clientIp, theme.text.t3)}>
                    {entry.client} {clientName && `(${clientName})`}
                </span>
                <span className={cn(s.secondary, theme.text.t4)}>
                    <WhoisInfo whois={entry.client_info?.whois} className={s.secondary} />
                </span>
            </div>
        </div>
    );
};
