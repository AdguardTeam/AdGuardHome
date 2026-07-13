import { Show } from 'solid-js';
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

export const LogCard = (props: Props) => {
    const displayDomain = () => props.entry.unicodeName || props.entry.domain;
    const proto = () => getProtocolName(props.entry.client_proto);
    const clientDetails = () => props.entry.client_info?.name || props.entry.client_id;
    const clientLocation = () => getClientLocation(props.entry.client_info?.whois);
    const statusKey = () =>
        getQueryStatusKey(props.entry.reason, props.entry.originalResponse ?? []);
    const reasonKey = () => getQueryReasonKey(props.entry.reason, props.entry.rules ?? []);
    const reasonDetails = () =>
        getQueryReasonDetails({
            elapsedMs: props.entry.elapsedMs,
            filters: props.filters,
            reason: props.entry.reason,
            rules: props.entry.rules ?? [],
            serviceName: props.entry.service_name || props.entry.serviceName,
            services: props.services,
            whitelistFilters: props.whitelistFilters,
        });
    const statusLabel = () => getQueryStatusLabel(statusKey());
    const reasonLabel = () => getQueryReasonLabel(reasonKey());

    return (
        <div
            class={s.card}
            onClick={() => props.onRowClick(props.entry)}
            data-testid="query-log-card"
        >
            <div class={s.cardBody}>
                <div class={s.cardHeader}>
                    <div class={s.titleBlock}>
                        <div class={s.titleRow}>
                            <span
                                class={cn(
                                    s.domain,
                                    theme.text.t3,
                                    theme.text.condenced,
                                    theme.text.semibold,
                                )}
                                title={displayDomain()}
                            >
                                {displayDomain()}
                            </span>

                            <div class={s.iconsRow}>
                                <span class={s.iconWrapper} aria-hidden="true">
                                    <Icon
                                        icon="tracking"
                                        color={props.entry.tracker ? 'green' : 'gray'}
                                        class={s.icon}
                                    />
                                </span>

                                <Show when={props.entry.answer_dnssec}>
                                    <span class={s.iconWrapper} aria-hidden="true">
                                        <Icon icon="lock" color="green" class={s.icon} />
                                    </span>
                                </Show>
                            </div>
                        </div>

                        <span class={cn(s.typeLine, theme.text.t4, theme.text.condenced)}>
                            {intl.getMessage('type_value', { value: props.entry.type })}, {proto()}
                        </span>
                    </div>

                    <div class={s.actions} onClick={(e) => e.stopPropagation()}>
                        <ActionsMenu
                            domain={props.entry.domain}
                            client={props.entry.client}
                            clientId={props.entry.client_id || props.entry.client}
                            onBlock={props.onBlock}
                            onUnblock={props.onUnblock}
                            onBlockClient={props.onBlockClient}
                            onDisallowClient={() => props.onDisallowClient(props.entry.client)}
                            onAddPersistentClient={props.onAddPersistentClient}
                            isBlocked={isBlockedReason(props.entry.reason)}
                            showAddPersistentClient={
                                props.persistentClientsLoaded &&
                                !hasPersistentClient(props.entry, props.persistentClientIds)
                            }
                            testIdPrefix="query-log-card"
                        />
                    </div>
                </div>

                <div class={s.fieldGrid}>
                    <span class={cn(s.fieldLabel, theme.text.t4, theme.text.condenced)}>
                        {intl.getMessage('time_table_header')}
                    </span>
                    <span class={cn(s.fieldValue, theme.text.t4, theme.text.condenced)}>
                        {formatLogDate(props.entry.time)}, {formatLogTime(props.entry.time)}
                    </span>

                    <span class={cn(s.fieldLabel, theme.text.t4, theme.text.condenced)}>
                        {intl.getMessage('status_table_header')}
                    </span>
                    <span
                        class={cn(
                            s.status,
                            theme.text.t4,
                            theme.text.condenced,
                            getStatusClassName(props.entry.reason),
                        )}
                    >
                        {statusLabel()}
                    </span>

                    <Show when={reasonKey() !== 'none'}>
                        <span class={cn(s.fieldLabel, theme.text.t4, theme.text.condenced)}>
                            {intl.getMessage('reason_table_header')}
                        </span>
                        <span class={cn(s.fieldValue, theme.text.t4, theme.text.condenced)}>
                            {reasonLabel()}
                            {reasonDetails() ? ` / ${reasonDetails()}` : ''}
                        </span>
                    </Show>

                    <span class={cn(s.fieldLabel, theme.text.t4, theme.text.condenced)}>
                        {intl.getMessage('client_ip')}
                    </span>
                    <span class={cn(s.fieldValue, theme.text.t4, theme.text.condenced)}>
                        {props.entry.client}
                    </span>

                    <Show when={clientDetails()}>
                        <span class={cn(s.fieldLabel, theme.text.t4, theme.text.condenced)}>
                            {intl.getMessage('client_details')}
                        </span>
                        <span class={cn(s.fieldValue, theme.text.t4, theme.text.condenced)}>
                            {clientDetails()}
                        </span>
                    </Show>

                    <Show when={clientLocation()}>
                        <span class={cn(s.fieldLabel, theme.text.t4, theme.text.condenced)}>
                            {intl.getMessage('client_location')}
                        </span>
                        <span class={cn(s.fieldValue, theme.text.t4, theme.text.condenced)}>
                            {clientLocation()}
                        </span>
                    </Show>
                </div>
            </div>
        </div>
    );
};
