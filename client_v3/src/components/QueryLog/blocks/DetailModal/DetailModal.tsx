import { Show, For } from 'solid-js';
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

const hasValue = (value: any) =>
    value !== undefined && value !== null && value !== '' && value !== false;

export const DetailModal = (props: Props) => {
    const statusKey = () =>
        getQueryStatusKey(props.entry.reason, props.entry.originalResponse ?? []);
    const reasonKey = () => getQueryReasonKey(props.entry.reason, props.entry.rules ?? []);
    const isBlocked = () => isBlockedReason(props.entry.reason);
    const isBlockedService = () => checkBlockedService(props.entry.reason);
    const isSafeSearch = () => props.entry.reason === FILTERED_STATUS.FILTERED_SAFE_SEARCH;
    const isRewrite = () =>
        props.entry.reason === FILTERED_STATUS.REWRITE ||
        props.entry.reason === FILTERED_STATUS.REWRITE_HOSTS ||
        props.entry.reason === FILTERED_STATUS.REWRITE_RULE;
    const showBlock = () => !isBlocked() && !isRewrite() && !isSafeSearch();
    const showAllowlist = () => isBlocked() || isSafeSearch();
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
    const statusClassName = () => getStatusClassName(props.entry.reason);
    const clientName = () => props.entry.client_info?.name || '';
    const protocol = () => getProtocolName(props.entry.client_proto);
    const responseList = () => formatResponses(props.entry.response);
    const originalResponseList = () => formatResponses(props.entry.originalResponse);
    const trackerSource = () => props.entry.tracker?.sourceData;
    const country = () => props.entry.client_info?.whois?.country;
    const network = () => props.entry.client_info?.whois?.orgname;
    const serviceId = () => props.entry.serviceName || props.entry.service_name;
    const serviceName = () =>
        serviceId() ? getServiceName(props.services, serviceId()!) || serviceId() : '';
    const reasonValue = () =>
        reasonDetails()
            ? `${getQueryReasonLabel(reasonKey())} / ${reasonDetails()}`
            : getQueryReasonLabel(reasonKey());
    const rowClassName = () => cn(s.row, theme.text.t3);
    const labelClassName = () => cn(s.label, theme.text.semibold);
    const renderValue = (content: any) => <span class={s.value}>{content}</span>;
    const renderListValue = (content: any) => (
        <div class={cn(s.value, s.responseList)}>{content}</div>
    );
    const renderStatusValue = (content: any) => (
        <span class={cn(s.value, s.statusValue, theme.text.semibold, statusClassName())}>
            {content}
        </span>
    );

    const handleBlock = () => {
        props.onBlock(props.entry.domain);
        props.onClose();
    };

    const handleAddToAllowlist = () => {
        props.onAddToAllowlist(props.entry.domain);
        props.onClose();
    };

    const handleAllowService = () => {
        if (!serviceId()) {
            return;
        }
        props.onAllowService(serviceId()!);
        props.onClose();
    };

    return (
        <Dialog
            visible
            onClose={props.onClose}
            title={
                <span data-testid="query-log-detail-title">{intl.getMessage('query_details')}</span>
            }
            class={s.dialog}
            wrapClass={s.wrap}
        >
            <div class={s.content} data-testid="query-log-detail-modal">
                <div class={s.scrollArea} data-testid="query-log-detail-scroll-area">
                    <div class={s.section}>
                        <Show when={props.entry.answer_dnssec}>
                            <div class={cn(s.row, theme.text.t3, theme.text.semibold)}>
                                {intl.getMessage('validated_with_dnssec')}
                            </div>
                        </Show>
                        <div
                            class={rowClassName()}
                            data-testid="query-log-detail-time"
                            data-field="time"
                        >
                            {intl.getMessage('query_log_detail_time', {
                                value: formatLogTimeDetailed(props.entry.time),
                                span: renderValue,
                            })}
                        </div>
                        <div
                            class={rowClassName()}
                            data-testid="query-log-detail-date"
                            data-field="date"
                        >
                            {intl.getMessage('query_log_detail_date', {
                                value: formatLogDate(props.entry.time),
                                span: renderValue,
                            })}
                        </div>
                        <div
                            class={rowClassName()}
                            data-testid="query-log-detail-domain"
                            data-field="domain"
                        >
                            {intl.getMessage('query_log_detail_domain', {
                                value: props.entry.unicodeName || props.entry.domain,
                                span: renderValue,
                            })}
                        </div>
                        <Show when={hasValue(props.entry.ecs)}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-ecs"
                                data-field="ecs"
                            >
                                {intl.getMessage('query_log_detail_ecs', {
                                    value: props.entry.ecs,
                                    span: renderValue,
                                })}
                            </div>
                        </Show>
                        <div
                            class={rowClassName()}
                            data-testid="query-log-detail-type"
                            data-field="type"
                        >
                            {intl.getMessage('query_log_detail_type', {
                                value: props.entry.type,
                                span: renderValue,
                            })}
                        </div>
                        <div
                            class={rowClassName()}
                            data-testid="query-log-detail-protocol"
                            data-field="protocol"
                        >
                            {intl.getMessage('query_log_detail_protocol', {
                                value: protocol(),
                                span: renderValue,
                            })}
                        </div>
                    </div>

                    <Show when={props.entry.tracker}>
                        <div class={s.section}>
                            <h3 class={cn(s.sectionTitle, theme.title.h6)}>
                                {intl.getMessage('known_tracker')}
                            </h3>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-tracker-name"
                                data-field="tracker-name"
                            >
                                {intl.getMessage('query_log_detail_name', {
                                    value: props.entry.tracker!.name,
                                    span: renderValue,
                                })}
                            </div>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-tracker-category"
                                data-field="tracker-category"
                            >
                                {intl.getMessage('query_log_detail_category', {
                                    value: props.entry.tracker!.category,
                                    span: renderValue,
                                })}
                            </div>
                            <Show when={trackerSource()?.name}>
                                <div
                                    class={rowClassName()}
                                    data-testid="query-log-detail-tracker-source"
                                    data-field="tracker-source"
                                >
                                    {intl.getMessage('query_log_detail_source', {
                                        value: trackerSource()!.name,
                                        span: (content: any) =>
                                            trackerSource()!.url ? (
                                                <a
                                                    href={trackerSource()!.url}
                                                    target="_blank"
                                                    rel="noopener noreferrer"
                                                    class={cn(s.link, s.value)}
                                                >
                                                    {content}
                                                </a>
                                            ) : (
                                                renderValue(content)
                                            ),
                                    })}
                                </div>
                            </Show>
                        </div>
                    </Show>

                    <div class={s.section}>
                        <h3 class={cn(s.sectionTitle, theme.title.h6)}>
                            {intl.getMessage('response_details')}
                        </h3>
                        <div
                            class={rowClassName()}
                            data-testid="query-log-detail-status"
                            data-field="status"
                        >
                            {intl.getMessage('query_log_detail_status', {
                                value: getQueryStatusLabel(statusKey()),
                                span: renderStatusValue,
                            })}
                        </div>
                        <Show when={reasonKey() !== 'none'}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-reason"
                                data-field="reason"
                            >
                                <span class={labelClassName()}>
                                    {intl.getMessage('query_log_detail_reason')}
                                </span>{' '}
                                {renderValue(reasonValue())}
                            </div>
                        </Show>
                        <div
                            class={rowClassName()}
                            data-testid="query-log-detail-cached"
                            data-field="cached"
                        >
                            {intl.getMessage('query_log_detail_served_from_cache', {
                                value: props.entry.cached
                                    ? intl.getMessage('yes')
                                    : intl.getMessage('no'),
                                span: renderValue,
                            })}
                        </div>
                        <Show when={hasValue(props.entry.upstream)}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-dns-server"
                                data-field="dns-server"
                            >
                                {intl.getMessage('query_log_detail_dns_server', {
                                    value: props.entry.upstream,
                                    span: renderValue,
                                })}
                            </div>
                        </Show>
                        <Show when={hasValue(props.entry.elapsedMs)}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-elapsed"
                                data-field="elapsed"
                            >
                                {intl.getMessage('query_log_detail_elapsed', {
                                    value: formatElapsedMs(
                                        props.entry.elapsedMs,
                                        intl.getMessage('milliseconds_abbreviation'),
                                    ),
                                    span: renderValue,
                                })}
                            </div>
                        </Show>
                        <Show when={hasValue(props.entry.status)}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-response-code"
                                data-field="response-code"
                            >
                                {intl.getMessage('query_log_detail_response_code', {
                                    value: props.entry.status,
                                    span: renderValue,
                                })}
                            </div>
                        </Show>
                        <Show when={responseList().length > 0}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-response"
                                data-field="response"
                            >
                                <span class={labelClassName()}>
                                    {intl.getMessage('query_log_detail_response')}
                                </span>{' '}
                                {renderListValue(
                                    <For each={responseList()}>
                                        {(response) => <div class={theme.text.t3}>{response}</div>}
                                    </For>,
                                )}
                            </div>
                        </Show>
                        <Show when={hasValue(serviceName())}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-service-name"
                                data-field="service-name"
                            >
                                {intl.getMessage('query_log_detail_service_name', {
                                    value: serviceName(),
                                    span: renderValue,
                                })}
                            </div>
                        </Show>
                        <Show when={props.entry.rules?.length || hasValue(props.entry.rule)}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-rules"
                                data-field="rules"
                            >
                                <span class={labelClassName()}>
                                    {intl.getMessage('query_log_detail_rules')}
                                </span>{' '}
                                {props.entry.rules?.length
                                    ? renderListValue(
                                          <For each={props.entry.rules}>
                                              {(rule) => (
                                                  <div class={theme.text.t3}>{rule.text}</div>
                                              )}
                                          </For>,
                                      )
                                    : renderValue(props.entry.rule)}
                            </div>
                        </Show>
                        <Show when={originalResponseList().length > 0}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-original-response"
                                data-field="original-response"
                            >
                                <span class={labelClassName()}>
                                    {intl.getMessage('query_log_detail_original_response')}
                                </span>{' '}
                                {renderListValue(
                                    <For each={originalResponseList()}>
                                        {(response) => <div class={theme.text.t3}>{response}</div>}
                                    </For>,
                                )}
                            </div>
                        </Show>
                    </div>

                    <div class={s.section}>
                        <h3 class={cn(s.sectionTitle, theme.title.h6)}>
                            {intl.getMessage('client_details')}
                        </h3>
                        <Show when={hasValue(props.entry.client)}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-client-address"
                                data-field="client-address"
                            >
                                {intl.getMessage('query_log_detail_address', {
                                    value: props.entry.client,
                                    span: renderValue,
                                })}
                            </div>
                        </Show>
                        <Show when={hasValue(clientName() || props.entry.client_id)}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-client-name"
                                data-field="client-name"
                            >
                                {intl.getMessage('query_log_detail_name', {
                                    value: clientName() || props.entry.client_id,
                                    span: renderValue,
                                })}
                            </div>
                        </Show>
                        <Show when={hasValue(country())}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-client-country"
                                data-field="client-country"
                            >
                                {intl.getMessage('query_log_detail_country', {
                                    value: country(),
                                    span: renderValue,
                                })}
                            </div>
                        </Show>
                        <Show when={hasValue(network())}>
                            <div
                                class={rowClassName()}
                                data-testid="query-log-detail-client-network"
                                data-field="client-network"
                            >
                                {intl.getMessage('query_log_detail_network', {
                                    value: network(),
                                    span: renderValue,
                                })}
                            </div>
                        </Show>
                    </div>
                </div>

                <div class={s.actionFooter} data-testid="query-log-detail-action-footer">
                    <Show when={showBlock()}>
                        <Button
                            data-testid="query-log-detail-action-block"
                            data-action="block"
                            type="button"
                            variant="danger"
                            size="small"
                            class={s.actionButton}
                            onClick={handleBlock}
                        >
                            {intl.getMessage('block')}
                        </Button>
                    </Show>

                    <Show when={showAllowlist()}>
                        <Button
                            data-testid="query-log-detail-action-allowlist"
                            data-action="allowlist"
                            type="button"
                            variant="primary"
                            size="small"
                            class={s.actionButton}
                            onClick={handleAddToAllowlist}
                        >
                            {intl.getMessage('add_to_allowlist')}
                        </Button>
                    </Show>

                    <Show when={isBlockedService() && serviceId()}>
                        <Button
                            data-testid="query-log-detail-action-allow-service"
                            data-action="allow-service"
                            type="button"
                            variant="secondary"
                            size="small"
                            class={s.actionButton}
                            onClick={handleAllowService}
                        >
                            {intl.getMessage('allow_service')}
                        </Button>
                    </Show>

                    <Show when={!showBlock() && !showAllowlist()}>
                        <Button
                            data-testid="query-log-detail-action-close"
                            data-action="close"
                            type="button"
                            variant="primary"
                            size="small"
                            class={s.actionButton}
                            onClick={props.onClose}
                        >
                            {intl.getMessage('close')}
                        </Button>
                    </Show>
                </div>
            </div>
        </Dialog>
    );
};
