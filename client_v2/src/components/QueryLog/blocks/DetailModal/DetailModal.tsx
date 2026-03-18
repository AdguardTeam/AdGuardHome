import React from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import { Dialog } from 'panel/common/ui/Dialog';
import theme from 'panel/lib/theme';

import { formatElapsedMs } from 'panel/helpers/helpers';
import {
    getStatusClassName,
    getStatusLabel,
    getProtocolName,
    formatLogTimeDetailed,
    formatLogDate,
} from '../../helpers';
import { LogEntry, ResponseEntry } from '../../types';

import { DetailRow } from './DetailRow';
import s from './DetailModal.module.pcss';

type Props = {
    entry: LogEntry;
    onClose: () => void;
};

const formatResponses = (responses: ResponseEntry[] = []) => responses
    .map(({ type, value, ttl }) => {
        if (!value) {
            return '';
        }

        const entry = [type, value].filter(Boolean).join(': ');

        if (ttl || ttl === 0) {
            return `${entry} (ttl=${ttl})`;
        }

        return entry;
    })
    .filter(Boolean);

const getSourceNode = (trackerSource?: { url?: string; name?: string }) => {
    const sourceValue = trackerSource?.url ? trackerSource.url : trackerSource?.name;
    if (trackerSource?.url) {
        return (
            <a href={trackerSource.url} target="_blank" rel="noopener noreferrer" className={cn(s.link, s.value)}>
                {trackerSource.name}
            </a>
        );
    }
    if (sourceValue) {
        return <span className={s.value}>{trackerSource?.name}</span>;
    }
    return null;
};

export const DetailModal = ({ entry, onClose }: Props) => {
    const statusLabel = getStatusLabel(entry.reason, entry.originalResponse, false);
    const statusClassName = getStatusClassName(entry.reason);
    const clientName = entry.client_info?.name || '';
    const protocol = getProtocolName(entry.client_proto);
    const responseList = formatResponses(entry.response);
    const originalResponseList = formatResponses(entry.originalResponse);
    const trackerSource = entry.tracker?.sourceData;
    const country = entry.client_info?.whois?.country;
    const network = entry.client_info?.whois?.orgname;
    const serviceName = entry.serviceName || entry.service_name;
    const sourceNode = getSourceNode(trackerSource);

    return (
        <Dialog visible onClose={onClose} title={intl.getMessage('request_details')} className={s.dialog}>
            <div className={s.content}>
                <div className={s.section}>
                    {entry.answer_dnssec && (
                        <div className={cn(s.row, theme.text.t3, theme.text.semibold)}>
                            {intl.getMessage('validated_with_dnssec')}
                        </div>
                    )}
                    <DetailRow label={intl.getMessage('query_log_detail_time')} value={formatLogTimeDetailed(entry.time)} />
                    <DetailRow label={intl.getMessage('query_log_detail_date')} value={formatLogDate(entry.time)} />
                    <DetailRow label={intl.getMessage('query_log_detail_domain')} value={entry.unicodeName || entry.domain} />
                    <DetailRow label={intl.getMessage('query_log_detail_ecs')} value={entry.ecs} />
                    <DetailRow label={intl.getMessage('query_log_detail_type')} value={entry.type} />
                    <DetailRow label={intl.getMessage('query_log_detail_protocol')} value={protocol} />
                </div>

                {entry.tracker && (
                    <div className={s.section}>
                        <h3 className={cn(s.sectionTitle, theme.title.h6)}>{intl.getMessage('known_tracker')}</h3>
                        <DetailRow label={intl.getMessage('query_log_detail_name')} value={entry.tracker.name} />
                        <DetailRow label={intl.getMessage('query_log_detail_category')} value={entry.tracker.category} />
                        <DetailRow label={intl.getMessage('query_log_detail_source')} value={sourceNode} />
                    </div>
                )}

                <div className={s.section}>
                    <h3 className={cn(s.sectionTitle, theme.title.h6)}>{intl.getMessage('response_details')}</h3>
                    <DetailRow label={intl.getMessage('query_log_detail_served_from_cache')} value={entry.cached ? intl.getMessage('yes') : null} />
                    <DetailRow
                        label={intl.getMessage('query_log_detail_status')}
                        value={(
                            <span className={cn(s.value, s.statusValue, theme.text.semibold, statusClassName)}>
                                {statusLabel}
                            </span>
                        )}
                    />
                    <DetailRow label={intl.getMessage('query_log_detail_dns_server')} value={entry.upstream} />
                    <DetailRow label={intl.getMessage('query_log_detail_elapsed')} value={entry.elapsedMs ? formatElapsedMs(entry.elapsedMs, (key) => intl.getMessage(key)) : null} />
                    <DetailRow label={intl.getMessage('query_log_detail_response_code')} value={entry.status} />
                    <DetailRow
                        label={intl.getMessage('query_log_detail_response')}
                        value={responseList.length > 0 ? (
                            <div className={cn(s.value, s.responseList)}>
                                {responseList.map((response, index) => (
                                    <div
                                        key={index}
                                        className={cn(s.responseItem, theme.text.t3)}
                                    >
                                        {response}
                                    </div>
                                ))}
                            </div>
                        ) : null}
                    />
                    <DetailRow label={intl.getMessage('query_log_detail_service_name')} value={serviceName} />
                    <DetailRow
                        label={intl.getMessage('query_log_detail_rules')}
                        value={entry.rules?.length ? (
                            <div className={cn(s.value, s.responseList)}>
                                {entry.rules.map((rule, index) => (
                                    <div
                                        key={`${rule.text}-${index}`}
                                        className={cn(s.responseItem, theme.text.t3)}
                                    >
                                        {rule.text}
                                    </div>
                                ))}
                            </div>
                        ) : entry.rule}
                    />
                    <DetailRow
                        label={intl.getMessage('query_log_detail_original_response')}
                        value={originalResponseList.length > 0 ? (
                            <div className={cn(s.value, s.responseList)}>
                                {originalResponseList.map((response, index) => (
                                    <div
                                        key={index}
                                        className={cn(s.responseItem, theme.text.t3)}
                                    >
                                        {response}
                                    </div>
                                ))}
                            </div>
                        ) : null}
                    />
                </div>

                <div className={s.section}>
                    <h3 className={cn(s.sectionTitle, theme.title.h6)}>{intl.getMessage('client_details')}</h3>
                    <DetailRow label={intl.getMessage('query_log_detail_address')} value={entry.client} />
                    <DetailRow label={intl.getMessage('query_log_detail_name')} value={clientName || entry.client_id} />
                    <DetailRow label={intl.getMessage('query_log_detail_country')} value={country} />
                    <DetailRow label={intl.getMessage('query_log_detail_network')} value={network} />
                </div>

                <div className={s.footer}>
                    <Button
                        type="button"
                        variant="primary"
                        size="small"
                        className={s.footerButton}
                        onClick={onClose}
                    >
                        {intl.getMessage('close')}
                    </Button>
                </div>
            </div>
        </Dialog>
    );
};
