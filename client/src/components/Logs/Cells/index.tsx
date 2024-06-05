import React, { Dispatch, memo, SetStateAction } from 'react';
import classNames from 'classnames';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';

import {
    captitalizeWords,
    checkFiltered,
    getRulesToFilterList,
    formatDateTime,
    formatElapsedMs,
    formatTime,
    getBlockingClientName,
    getServiceName,
    processContent,
} from '../../../helpers/helpers';
import {
    BLOCK_ACTIONS,
    DEFAULT_SHORT_DATE_FORMAT_OPTIONS,
    FILTERED_STATUS,
    FILTERED_STATUS_TO_META_MAP,
    LONG_TIME_FORMAT,
    QUERY_STATUS_COLORS,
    SCHEME_TO_PROTOCOL_MAP,
} from '../../../helpers/constants';
import { getSourceData } from '../../../helpers/trackers/trackers';

import { toggleBlocking, toggleBlockingForClient } from '../../../actions';

import DateCell from './DateCell';

import DomainCell from './DomainCell';

import ResponseCell from './ResponseCell';

import ClientCell from './ClientCell';
import { toggleClientBlock } from '../../../actions/access';
import { getBlockClientInfo, BUTTON_PREFIX } from './helpers';
import { updateLogs } from '../../../actions/queryLogs';

import '../Logs.css';
import { RootState } from '../../../initialState';

interface RowProps {
    style?: object;
    rowProps: {
        reason: string;
        answer_dnssec: boolean;
        client: string;
        domain: string;
        elapsedMs: string;
        response: unknown[];
        time: string;
        tracker?: {
            name: string;
            category: string;
        };
        upstream: string;
        cached: boolean;
        type: string;
        client_proto: string;
        client_id?: string;
        ecs?: string;
        client_info?: {
            name: string;
            whois: {
                country?: string;
                city?: string;
                orgname?: string;
            };
            disallowed: boolean;
            disallowed_rule: string;
        };
        rules?: {
            text: string;
            filter_list_id: number;
        }[];
        originalResponse?: unknown[];
        status: string;
        service_name?: string;
    };
    isSmallScreen: boolean;
    setDetailedDataCurrent: Dispatch<SetStateAction<any>>;
    setButtonType: (...args: unknown[]) => unknown;
    setModalOpened: (...args: unknown[]) => unknown;
}

const Row = memo(
    ({
        style,
        rowProps,
        rowProps: { reason },
        isSmallScreen,
        setDetailedDataCurrent,
        setButtonType,
        setModalOpened,
    }: RowProps) => {
        const dispatch = useDispatch();
        const { t } = useTranslation();

        const dnssec_enabled = useSelector((state: RootState) => state.dnsConfig.dnssec_enabled);

        const filters = useSelector((state: RootState) => state.filtering.filters, shallowEqual);

        const whitelistFilters = useSelector((state: RootState) => state.filtering.whitelistFilters, shallowEqual);

        const autoClients = useSelector((state: RootState) => state.dashboard.autoClients, shallowEqual);

        const processingSet = useSelector((state: RootState) => state.access.processingSet);

        const allowedClients = useSelector((state: RootState) => state.access.allowed_clients, shallowEqual);

        const services = useSelector((state: RootState) => state?.services);

        const clients = useSelector((state: RootState) => state.dashboard.clients);

        const onClick = () => {
            if (!isSmallScreen) {
                return;
            }
            const {
                answer_dnssec,
                client,
                domain,
                elapsedMs,
                client_info,
                response,
                time,
                tracker,
                upstream,
                type,
                client_proto,
                client_id,
                rules,
                originalResponse,
                status,
                service_name,
                cached,
            } = rowProps;

            const hasTracker = !!tracker;

            const autoClient = autoClients.find((autoClient: any) => autoClient.name === client);

            const source = autoClient?.source;

            const formattedElapsedMs = formatElapsedMs(elapsedMs, t);
            const isFiltered = checkFiltered(reason);

            const isBlocked =
                reason === FILTERED_STATUS.FILTERED_BLACK_LIST || reason === FILTERED_STATUS.FILTERED_BLOCKED_SERVICE;

            const buttonType = isFiltered ? BLOCK_ACTIONS.UNBLOCK : BLOCK_ACTIONS.BLOCK;
            const onToggleBlock = () => {
                dispatch(toggleBlocking(buttonType, domain));
            };

            const isBlockedByResponse = originalResponse.length > 0 && isBlocked;
            const requestStatus = t(
                isBlockedByResponse ? 'blocked_by_cname_or_ip' : FILTERED_STATUS_TO_META_MAP[reason]?.LABEL || reason,
            );

            const protocol = t(SCHEME_TO_PROTOCOL_MAP[client_proto]) || '';

            const sourceData = getSourceData(tracker);

            const {
                confirmMessage,
                buttonKey: blockingClientKey,
                lastRuleInAllowlist,
            } = getBlockClientInfo(
                client,
                client_info?.disallowed || false,
                client_info?.disallowed_rule || '',
                allowedClients,
            );

            const blockingForClientKey = isFiltered ? 'unblock_for_this_client_only' : 'block_for_this_client_only';
            const clientNameBlockingFor = getBlockingClientName(clients, client);

            const onBlockingForClientClick = () => {
                dispatch(toggleBlockingForClient(buttonType, domain, clientNameBlockingFor));
            };

            const onBlockingClientClick = async () => {
                if (window.confirm(confirmMessage)) {
                    await dispatch(
                        toggleClientBlock(client, client_info?.disallowed || false, client_info?.disallowed_rule || ''),
                    );
                    await dispatch(updateLogs());
                    setModalOpened(false);
                }
            };

            const blockButton = (
                <>
                    <div className="title--border" />

                    <button
                        type="button"
                        className={classNames(
                            'button-action--arrow-option mb-1',
                            { 'bg--danger': !isBlocked },
                            { 'bg--green': isFiltered },
                        )}
                        onClick={onToggleBlock}>
                        {t(buttonType)}
                    </button>
                </>
            );

            const blockForClientButton = (
                <button
                    className="text-center font-weight-bold py-1 button-action--arrow-option"
                    onClick={onBlockingForClientClick}>
                    {t(blockingForClientKey)}
                </button>
            );

            const blockClientButton = (
                <button
                    className="text-center font-weight-bold py-1 button-action--arrow-option"
                    onClick={onBlockingClientClick}
                    disabled={processingSet || lastRuleInAllowlist}>
                    {t(blockingClientKey)}
                </button>
            );

            const detailedData = {
                time_table_header: formatTime(time, LONG_TIME_FORMAT),

                date: formatDateTime(time, DEFAULT_SHORT_DATE_FORMAT_OPTIONS),
                encryption_status: isBlocked ? <div className="bg--danger">{requestStatus}</div> : requestStatus,
                ...(FILTERED_STATUS.FILTERED_BLOCKED_SERVICE &&
                    service_name &&
                    services.allServices && { service_name: getServiceName(services.allServices, service_name) }),
                domain,
                type_table_header: type,
                protocol,
                known_tracker: hasTracker && 'title',
                table_name: tracker?.name,
                category_label: hasTracker && captitalizeWords(tracker.category),
                tracker_source: hasTracker && sourceData && (
                    <a href={sourceData.url} target="_blank" rel="noopener noreferrer" className="link--green">
                        {sourceData.name}
                    </a>
                ),
                response_details: 'title',
                install_settings_dns: upstream,
                ...(cached && {
                    served_from_cache_label: (
                        <svg className="icons icon--20 icon--green">
                            <use xlinkHref="#check" />
                        </svg>
                    ),
                }),
                elapsed: formattedElapsedMs,
                ...(rules.length > 0 && { rule_label: getRulesToFilterList(rules, filters, whitelistFilters) }),
                response_table_header: response?.join('\n'),
                response_code: status,
                client_details: 'title',
                ip_address: client,
                name: client_info?.name || client_id,
                country: client_info?.whois?.country,
                city: client_info?.whois?.city,
                network: client_info?.whois?.orgname,
                source_label: source,
                validated_with_dnssec: dnssec_enabled ? Boolean(answer_dnssec) : false,
                original_response: originalResponse?.join('\n'),
                [BUTTON_PREFIX + buttonType]: blockButton,
                [BUTTON_PREFIX + blockingForClientKey]: blockForClientButton,
                [BUTTON_PREFIX + blockingClientKey]: blockClientButton,
            };

            setDetailedDataCurrent(processContent(detailedData));
            setButtonType(buttonType);
            setModalOpened(true);
        };

        const isDetailed = useSelector((state: RootState) => state.queryLogs.isDetailed);

        const className = classNames(
            'd-flex px-5 logs__row',
            `logs__row--${FILTERED_STATUS_TO_META_MAP?.[reason]?.COLOR ?? QUERY_STATUS_COLORS.WHITE}`,
            {
                'logs__cell--detailed': isDetailed,
            },
        );

        return (
            <div style={style} className={className} onClick={onClick} role="row">
                <DateCell {...rowProps} />

                <DomainCell {...rowProps} />

                <ResponseCell {...rowProps} />

                <ClientCell {...rowProps} />
            </div>
        );
    },
);

Row.displayName = 'Row';

export default Row;
