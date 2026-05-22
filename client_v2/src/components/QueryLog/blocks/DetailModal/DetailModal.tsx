import React from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import { Dialog } from 'panel/common/ui/Dialog';
import theme from 'panel/lib/theme';

import {
    checkBlockedService,
    formatElapsedMs,
    getServiceName,
    type Filter,
} from 'panel/helpers/helpers';
import { FILTERED_STATUS } from 'panel/helpers/constants';
import {
    getQueryReasonDetails,
    getQueryReasonLabel,
    getQueryReasonKey,
    getQueryStatusLabel,
    getQueryStatusKey,
    getStatusClassName,
    getProtocolName,
    isBlockedReason,
    formatLogTimeDetailed,
    formatLogDate,
} from '../../helpers';
import { LogEntry, ResponseEntry, Service } from '../../types';

import s from './DetailModal.module.pcss';

type Props = {
    entry: LogEntry;
    filters: Filter[];
    services: Service[];
    whitelistFilters: Filter[];
    onClose: () => void;
    onBlock: (domain: string) => void;
    onAddToAllowlist: (domain: string) => void;
    onAllowService: (serviceId: string) => void;
};

const formatResponses = (responses: ResponseEntry[] = []) =>
    responses
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

const hasValue = (value: React.ReactNode) =>
    value !== undefined && value !== null && value !== '' && value !== false;

export const DetailModal = ({
    entry,
    filters,
    services,
    whitelistFilters,
    onClose,
    onBlock,
    onAddToAllowlist,
    onAllowService,
}: Props) => {
    const statusKey = getQueryStatusKey(entry.reason, entry.originalResponse ?? []);
    const reasonKey = getQueryReasonKey(entry.reason, entry.rules ?? []);
    const isBlocked = isBlockedReason(entry.reason);
    const isBlockedService = checkBlockedService(entry.reason);
    const isSafeSearch = entry.reason === FILTERED_STATUS.FILTERED_SAFE_SEARCH;
    const isRewrite =
        entry.reason === FILTERED_STATUS.REWRITE ||
        entry.reason === FILTERED_STATUS.REWRITE_HOSTS ||
        entry.reason === FILTERED_STATUS.REWRITE_RULE;
    const showBlock = !isBlocked && !isRewrite && !isSafeSearch;
    const showAllowlist = isBlocked || isSafeSearch;
    const reasonDetails = getQueryReasonDetails({
        elapsedMs: entry.elapsedMs,
        filters,
        reason: entry.reason,
        rules: entry.rules ?? [],
        serviceName: entry.service_name || entry.serviceName,
        services,
        whitelistFilters,
    });
    const statusClassName = getStatusClassName(entry.reason);
    const clientName = entry.client_info?.name || '';
    const protocol = getProtocolName(entry.client_proto);
    const responseList = formatResponses(entry.response);
    const originalResponseList = formatResponses(entry.originalResponse);
    const trackerSource = entry.tracker?.sourceData;
    const country = entry.client_info?.whois?.country;
    const network = entry.client_info?.whois?.orgname;
    const serviceId = entry.serviceName || entry.service_name;
    const serviceName = serviceId ? getServiceName(services, serviceId) || serviceId : '';
    const reasonValue = reasonDetails
        ? `${getQueryReasonLabel(reasonKey)} / ${reasonDetails}`
        : getQueryReasonLabel(reasonKey);
    const rowClassName = cn(s.row, theme.text.t3);
    const labelClassName = cn(s.label, theme.text.semibold);
    const renderValue = (content: React.ReactNode) => <span className={s.value}>{content}</span>;
    const renderListValue = (content: React.ReactNode) => (
        <div className={cn(s.value, s.responseList)}>{content}</div>
    );
    const renderStatusValue = (content: React.ReactNode) => (
        <span className={cn(s.value, s.statusValue, theme.text.semibold, statusClassName)}>
            {content}
        </span>
    );

    const handleBlock = () => {
        onBlock(entry.domain);
        onClose();
    };

    const handleAddToAllowlist = () => {
        onAddToAllowlist(entry.domain);
        onClose();
    };

    const handleAllowService = () => {
        if (!serviceId) {
            return;
        }

        onAllowService(serviceId);
        onClose();
    };

    return (
        <Dialog
            visible
            onClose={onClose}
            title={
                <span data-testid="query-log-detail-title">{intl.getMessage('query_details')}</span>
            }
            className={s.dialog}
            wrapClassName={s.wrap}
        >
            <div className={s.content} data-testid="query-log-detail-modal">
                <div className={s.scrollArea} data-testid="query-log-detail-scroll-area">
                    <div className={s.section}>
                        {entry.answer_dnssec && (
                            <div className={cn(s.row, theme.text.t3, theme.text.semibold)}>
                                {intl.getMessage('validated_with_dnssec')}
                            </div>
                        )}
                        <div
                            className={rowClassName}
                            data-testid="query-log-detail-time"
                            data-field="time"
                        >
                            {intl.getMessage('query_log_detail_time', {
                                value: formatLogTimeDetailed(entry.time),
                                span: renderValue,
                            })}
                        </div>
                        <div
                            className={rowClassName}
                            data-testid="query-log-detail-date"
                            data-field="date"
                        >
                            {intl.getMessage('query_log_detail_date', {
                                value: formatLogDate(entry.time),
                                span: renderValue,
                            })}
                        </div>
                        <div
                            className={rowClassName}
                            data-testid="query-log-detail-domain"
                            data-field="domain"
                        >
                            {intl.getMessage('query_log_detail_domain', {
                                value: entry.unicodeName || entry.domain,
                                span: renderValue,
                            })}
                        </div>
                        {hasValue(entry.ecs) && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-ecs"
                                data-field="ecs"
                            >
                                {intl.getMessage('query_log_detail_ecs', {
                                    value: entry.ecs,
                                    span: renderValue,
                                })}
                            </div>
                        )}
                        <div
                            className={rowClassName}
                            data-testid="query-log-detail-type"
                            data-field="type"
                        >
                            {intl.getMessage('query_log_detail_type', {
                                value: entry.type,
                                span: renderValue,
                            })}
                        </div>
                        <div
                            className={rowClassName}
                            data-testid="query-log-detail-protocol"
                            data-field="protocol"
                        >
                            {intl.getMessage('query_log_detail_protocol', {
                                value: protocol,
                                span: renderValue,
                            })}
                        </div>
                    </div>

                    {entry.tracker && (
                        <div className={s.section}>
                            <h3 className={cn(s.sectionTitle, theme.title.h6)}>
                                {intl.getMessage('known_tracker')}
                            </h3>
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-tracker-name"
                                data-field="tracker-name"
                            >
                                {intl.getMessage('query_log_detail_name', {
                                    value: entry.tracker.name,
                                    span: renderValue,
                                })}
                            </div>
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-tracker-category"
                                data-field="tracker-category"
                            >
                                {intl.getMessage('query_log_detail_category', {
                                    value: entry.tracker.category,
                                    span: renderValue,
                                })}
                            </div>
                            {trackerSource?.name && (
                                <div
                                    className={rowClassName}
                                    data-testid="query-log-detail-tracker-source"
                                    data-field="tracker-source"
                                >
                                    {intl.getMessage('query_log_detail_source', {
                                        value: trackerSource.name,
                                        span: (content: React.ReactNode) =>
                                            trackerSource.url ? (
                                                <a
                                                    href={trackerSource.url}
                                                    target="_blank"
                                                    rel="noopener noreferrer"
                                                    className={cn(s.link, s.value)}
                                                >
                                                    {content}
                                                </a>
                                            ) : (
                                                renderValue(content)
                                            ),
                                    })}
                                </div>
                            )}
                        </div>
                    )}

                    <div className={s.section}>
                        <h3 className={cn(s.sectionTitle, theme.title.h6)}>
                            {intl.getMessage('response_details')}
                        </h3>
                        <div
                            className={rowClassName}
                            data-testid="query-log-detail-status"
                            data-field="status"
                        >
                            {intl.getMessage('query_log_detail_status', {
                                value: getQueryStatusLabel(statusKey),
                                span: renderStatusValue,
                            })}
                        </div>
                        {reasonKey !== 'none' && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-reason"
                                data-field="reason"
                            >
                                <span className={labelClassName}>
                                    {intl.getMessage('query_log_detail_reason')}
                                </span>{' '}
                                {renderValue(reasonValue)}
                            </div>
                        )}
                        <div
                            className={rowClassName}
                            data-testid="query-log-detail-cached"
                            data-field="cached"
                        >
                            {intl.getMessage('query_log_detail_served_from_cache', {
                                value: entry.cached
                                    ? intl.getMessage('yes')
                                    : intl.getMessage('no'),
                                span: renderValue,
                            })}
                        </div>
                        {hasValue(entry.upstream) && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-dns-server"
                                data-field="dns-server"
                            >
                                {intl.getMessage('query_log_detail_dns_server', {
                                    value: entry.upstream,
                                    span: renderValue,
                                })}
                            </div>
                        )}
                        {hasValue(entry.elapsedMs) && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-elapsed"
                                data-field="elapsed"
                            >
                                {intl.getMessage('query_log_detail_elapsed', {
                                    value: formatElapsedMs(
                                        entry.elapsedMs,
                                        intl.getMessage('milliseconds_abbreviation'),
                                    ),
                                    span: renderValue,
                                })}
                            </div>
                        )}
                        {hasValue(entry.status) && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-response-code"
                                data-field="response-code"
                            >
                                {intl.getMessage('query_log_detail_response_code', {
                                    value: entry.status,
                                    span: renderValue,
                                })}
                            </div>
                        )}
                        {responseList.length > 0 && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-response"
                                data-field="response"
                            >
                                <span className={labelClassName}>
                                    {intl.getMessage('query_log_detail_response')}
                                </span>{' '}
                                {renderListValue(
                                    responseList.map((response, index) => (
                                        <div key={index} className={theme.text.t3}>
                                            {response}
                                        </div>
                                    )),
                                )}
                            </div>
                        )}
                        {hasValue(serviceName) && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-service-name"
                                data-field="service-name"
                            >
                                {intl.getMessage('query_log_detail_service_name', {
                                    value: serviceName,
                                    span: renderValue,
                                })}
                            </div>
                        )}
                        {(entry.rules?.length || hasValue(entry.rule)) && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-rules"
                                data-field="rules"
                            >
                                <span className={labelClassName}>
                                    {intl.getMessage('query_log_detail_rules')}
                                </span>{' '}
                                {entry.rules?.length
                                    ? renderListValue(
                                          entry.rules.map((rule, index) => (
                                              <div
                                                  key={`${rule.text}-${index}`}
                                                  className={theme.text.t3}
                                              >
                                                  {rule.text}
                                              </div>
                                          )),
                                      )
                                    : renderValue(entry.rule)}
                            </div>
                        )}
                        {originalResponseList.length > 0 && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-original-response"
                                data-field="original-response"
                            >
                                <span className={labelClassName}>
                                    {intl.getMessage('query_log_detail_original_response')}
                                </span>{' '}
                                {renderListValue(
                                    originalResponseList.map((response, index) => (
                                        <div key={index} className={theme.text.t3}>
                                            {response}
                                        </div>
                                    )),
                                )}
                            </div>
                        )}
                    </div>

                    <div className={s.section}>
                        <h3 className={cn(s.sectionTitle, theme.title.h6)}>
                            {intl.getMessage('client_details')}
                        </h3>
                        {hasValue(entry.client) && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-client-address"
                                data-field="client-address"
                            >
                                {intl.getMessage('query_log_detail_address', {
                                    value: entry.client,
                                    span: renderValue,
                                })}
                            </div>
                        )}
                        {hasValue(clientName || entry.client_id) && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-client-name"
                                data-field="client-name"
                            >
                                {intl.getMessage('query_log_detail_name', {
                                    value: clientName || entry.client_id,
                                    span: renderValue,
                                })}
                            </div>
                        )}
                        {hasValue(country) && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-client-country"
                                data-field="client-country"
                            >
                                {intl.getMessage('query_log_detail_country', {
                                    value: country,
                                    span: renderValue,
                                })}
                            </div>
                        )}
                        {hasValue(network) && (
                            <div
                                className={rowClassName}
                                data-testid="query-log-detail-client-network"
                                data-field="client-network"
                            >
                                {intl.getMessage('query_log_detail_network', {
                                    value: network,
                                    span: renderValue,
                                })}
                            </div>
                        )}
                    </div>
                </div>

                <div className={s.actionFooter} data-testid="query-log-detail-action-footer">
                    {showBlock && (
                        <Button
                            data-testid="query-log-detail-action-block"
                            data-action="block"
                            type="button"
                            variant="danger"
                            size="small"
                            className={s.actionButton}
                            onClick={handleBlock}
                        >
                            {intl.getMessage('block')}
                        </Button>
                    )}

                    {showAllowlist && (
                        <Button
                            data-testid="query-log-detail-action-allowlist"
                            data-action="allowlist"
                            type="button"
                            variant="primary"
                            size="small"
                            className={s.actionButton}
                            onClick={handleAddToAllowlist}
                        >
                            {intl.getMessage('add_to_allowlist')}
                        </Button>
                    )}

                    {isBlockedService && serviceId && (
                        <Button
                            data-testid="query-log-detail-action-allow-service"
                            data-action="allow-service"
                            type="button"
                            variant="secondary"
                            size="small"
                            className={s.actionButton}
                            onClick={handleAllowService}
                        >
                            {intl.getMessage('allow_service')}
                        </Button>
                    )}

                    {!showBlock && !showAllowlist && (
                        <Button
                            data-testid="query-log-detail-action-close"
                            data-action="close"
                            type="button"
                            variant="primary"
                            size="small"
                            className={s.actionButton}
                            onClick={onClose}
                        >
                            {intl.getMessage('close')}
                        </Button>
                    )}
                </div>
            </div>
        </Dialog>
    );
};
