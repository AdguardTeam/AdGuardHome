import React from 'react';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';

import intl from 'panel/common/intl';
import { Filter } from 'panel/helpers/helpers';
import {
    formatLogDate,
    formatLogTime,
    getClientLocation,
    getProtocolName,
    getQueryReasonLabel,
    getQueryReasonDetails,
    getQueryReasonKey,
    getQueryStatusLabel,
    getQueryStatusKey,
    getStatusClassName,
    hasPersistentClient,
    isBlockedReason,
} from '../../helpers';
import { LogEntry, Service } from '../../types';
import { ActionsMenu } from '../ActionsMenu';

import s from './LogCard.module.pcss';

type Props = {
    entry: LogEntry;
    filters: Filter[];
    services: Service[];
    whitelistFilters: Filter[];
    onRowClick: (entry: LogEntry) => void;
    onBlock: (domain: string) => void;
    onUnblock: (domain: string) => void;
    onBlockClient: (domain: string, client: string) => void;
    onDisallowClient: (ip: string) => void;
    onAddPersistentClient: (clientId: string) => void;
    persistentClientIds: string[];
    persistentClientsLoaded: boolean;
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
    onAddPersistentClient,
    persistentClientIds,
    persistentClientsLoaded,
}: Props) => {
    const displayDomain = entry.unicodeName || entry.domain;
    const proto = getProtocolName(entry.client_proto);
    const clientDetails = entry.client_info?.name || entry.client_id;
    const clientLocation = getClientLocation(entry.client_info?.whois);
    const statusKey = getQueryStatusKey(entry.reason, entry.originalResponse ?? []);
    const reasonKey = getQueryReasonKey(entry.reason, entry.rules ?? []);
    const reasonDetails = getQueryReasonDetails({
        elapsedMs: entry.elapsedMs,
        filters,
        reason: entry.reason,
        rules: entry.rules ?? [],
        serviceName: entry.service_name || entry.serviceName,
        services,
        whitelistFilters,
    });
    const statusLabel = getQueryStatusLabel(statusKey);
    const reasonLabel = getQueryReasonLabel(reasonKey);

    return (
        <div className={s.card} onClick={() => onRowClick(entry)} data-testid="query-log-card">
            <div className={s.cardBody}>
                <div className={s.cardHeader}>
                    <div className={s.titleBlock}>
                        <div className={s.titleRow}>
                            <span
                                className={cn(
                                    s.domain,
                                    theme.text.t3,
                                    theme.text.condenced,
                                    theme.text.semibold,
                                )}
                                title={displayDomain}
                            >
                                {displayDomain}
                            </span>

                            <div className={s.iconsRow}>
                                <span className={s.iconWrapper} aria-hidden="true">
                                    <Icon
                                        icon="tracking"
                                        color={entry.tracker ? 'green' : 'gray'}
                                        className={s.icon}
                                    />
                                </span>

                                {entry.answer_dnssec && (
                                    <span className={s.iconWrapper} aria-hidden="true">
                                        <Icon icon="lock" color="green" className={s.icon} />
                                    </span>
                                )}
                            </div>
                        </div>

                        <span className={cn(s.typeLine, theme.text.t4, theme.text.condenced)}>
                            {intl.getMessage('type_value', { value: entry.type })}, {proto}
                        </span>
                    </div>

                    <div className={s.actions} onClick={(e) => e.stopPropagation()}>
                        <ActionsMenu
                            domain={entry.domain}
                            client={entry.client}
                            clientId={entry.client_id || entry.client}
                            onBlock={onBlock}
                            onUnblock={onUnblock}
                            onBlockClient={onBlockClient}
                            onDisallowClient={() => onDisallowClient(entry.client)}
                            onAddPersistentClient={onAddPersistentClient}
                            isBlocked={isBlockedReason(entry.reason)}
                            showAddPersistentClient={
                                persistentClientsLoaded &&
                                !hasPersistentClient(entry, persistentClientIds)
                            }
                            testIdPrefix="query-log-card"
                        />
                    </div>
                </div>

                <div className={s.fieldGrid}>
                    <span className={cn(s.fieldLabel, theme.text.t4, theme.text.condenced)}>
                        {intl.getMessage('time_table_header')}
                    </span>
                    <span className={cn(s.fieldValue, theme.text.t4, theme.text.condenced)}>
                        {formatLogDate(entry.time)}, {formatLogTime(entry.time)}
                    </span>

                    <span className={cn(s.fieldLabel, theme.text.t4, theme.text.condenced)}>
                        {intl.getMessage('status_table_header')}
                    </span>
                    <span
                        className={cn(
                            s.status,
                            theme.text.t4,
                            theme.text.condenced,
                            getStatusClassName(entry.reason),
                        )}
                    >
                        {statusLabel}
                    </span>

                    {reasonKey !== 'none' && (
                        <>
                            <span className={cn(s.fieldLabel, theme.text.t4, theme.text.condenced)}>
                                {intl.getMessage('reason_table_header')}
                            </span>
                            <span className={cn(s.fieldValue, theme.text.t4, theme.text.condenced)}>
                                {reasonLabel}
                                {reasonDetails ? ` / ${reasonDetails}` : ''}
                            </span>
                        </>
                    )}

                    <span className={cn(s.fieldLabel, theme.text.t4, theme.text.condenced)}>
                        {intl.getMessage('client_ip')}
                    </span>
                    <span className={cn(s.fieldValue, theme.text.t4, theme.text.condenced)}>
                        {entry.client}
                    </span>

                    {clientDetails && (
                        <>
                            <span className={cn(s.fieldLabel, theme.text.t4, theme.text.condenced)}>
                                {intl.getMessage('client_details')}
                            </span>
                            <span className={cn(s.fieldValue, theme.text.t4, theme.text.condenced)}>
                                {clientDetails}
                            </span>
                        </>
                    )}

                    {clientLocation && (
                        <>
                            <span className={cn(s.fieldLabel, theme.text.t4, theme.text.condenced)}>
                                {intl.getMessage('client_location')}
                            </span>
                            <span className={cn(s.fieldValue, theme.text.t4, theme.text.condenced)}>
                                {clientLocation}
                            </span>
                        </>
                    )}
                </div>
            </div>
        </div>
    );
};
