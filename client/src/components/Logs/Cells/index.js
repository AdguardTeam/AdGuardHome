import React, { memo } from 'react';
import classNames from 'classnames';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import propTypes from 'prop-types';
import {
    captitalizeWords,
    checkFiltered,
    formatDateTime,
    formatElapsedMs,
    formatTime,
    getBlockingClientName,
    getFilterName,
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
import '../Logs.css';
import { toggleClientBlock } from '../../../actions/access';
import { getBlockClientInfo, BUTTON_PREFIX } from './helpers';
import { updateLogs } from '../../../actions/queryLogs';

const Row = memo(({
    style,
    rowProps,
    rowProps: { reason },
    isSmallScreen,
    setDetailedDataCurrent,
    setButtonType,
    setModalOpened,
}) => {
    const dispatch = useDispatch();
    const { t } = useTranslation();
    const dnssec_enabled = useSelector((state) => state.dnsConfig.dnssec_enabled);
    const filters = useSelector((state) => state.filtering.filters, shallowEqual);
    const whitelistFilters = useSelector((state) => state.filtering.whitelistFilters, shallowEqual);
    const autoClients = useSelector((state) => state.dashboard.autoClients, shallowEqual);

    const clients = useSelector((state) => state.dashboard.clients);

    const onClick = () => {
        if (!isSmallScreen) {
            return;
        }
        const {
            answer_dnssec,
            client,
            domain,
            elapsedMs,
            info,
            info: { disallowed, disallowed_rule },
            reason,
            response,
            time,
            tracker,
            upstream,
            type,
            client_proto,
            filterId,
            rule,
            originalResponse,
            status,
            service_name,
        } = rowProps;

        const hasTracker = !!tracker;

        const autoClient = autoClients
            .find((autoClient) => autoClient.name === client);

        const { whois_info } = info;
        const country = whois_info?.country;
        const city = whois_info?.city;
        const network = whois_info?.orgname;

        const source = autoClient?.source;

        const formattedElapsedMs = formatElapsedMs(elapsedMs, t);
        const isFiltered = checkFiltered(reason);

        const isBlocked = reason === FILTERED_STATUS.FILTERED_BLACK_LIST
                || reason === FILTERED_STATUS.FILTERED_BLOCKED_SERVICE;

        const buttonType = isFiltered ? BLOCK_ACTIONS.UNBLOCK : BLOCK_ACTIONS.BLOCK;
        const onToggleBlock = () => {
            dispatch(toggleBlocking(buttonType, domain));
        };

        const isBlockedByResponse = originalResponse.length > 0 && isBlocked;
        const requestStatus = t(isBlockedByResponse ? 'blocked_by_cname_or_ip' : FILTERED_STATUS_TO_META_MAP[reason]?.LABEL || reason);

        const protocol = t(SCHEME_TO_PROTOCOL_MAP[client_proto]) || '';

        const sourceData = getSourceData(tracker);

        const filter = getFilterName(filters, whitelistFilters, filterId);

        const {
            confirmMessage,
            buttonKey: blockingClientKey,
            isNotInAllowedList,
        } = getBlockClientInfo(client, disallowed, disallowed_rule);

        const blockingForClientKey = isFiltered ? 'unblock_for_this_client_only' : 'block_for_this_client_only';
        const clientNameBlockingFor = getBlockingClientName(clients, client);

        const onBlockingForClientClick = () => {
            dispatch(toggleBlockingForClient(buttonType, domain, clientNameBlockingFor));
        };

        const onBlockingClientClick = async () => {
            if (window.confirm(confirmMessage)) {
                await dispatch(toggleClientBlock(client, disallowed, disallowed_rule));
                await dispatch(updateLogs());
                setModalOpened(false);
            }
        };

        const blockButton = <button
                className={classNames('title--border text-center button-action--arrow-option', { 'bg--danger': !isBlocked })}
                onClick={onToggleBlock}>
            {t(buttonType)}
        </button>;

        const blockForClientButton = <button
                className='text-center font-weight-bold py-2 button-action--arrow-option'
                onClick={onBlockingForClientClick}>
            {t(blockingForClientKey)}
        </button>;

        const blockClientButton = <button
                className='text-center font-weight-bold py-2 button-action--arrow-option'
                onClick={onBlockingClientClick}
                disabled={isNotInAllowedList}>
            {t(blockingClientKey)}
        </button>;

        const detailedData = {
            time_table_header: formatTime(time, LONG_TIME_FORMAT),
            date: formatDateTime(time, DEFAULT_SHORT_DATE_FORMAT_OPTIONS),
            encryption_status: isBlocked
                ? <div className="bg--danger">{requestStatus}</div> : requestStatus,
            ...(FILTERED_STATUS.FILTERED_BLOCKED_SERVICE && service_name
                && { service_name: getServiceName(service_name) }),
            domain,
            type_table_header: type,
            protocol,
            known_tracker: hasTracker && 'title',
            table_name: tracker?.name,
            category_label: hasTracker && captitalizeWords(tracker.category),
            tracker_source: hasTracker && sourceData
                    && <a
                            href={sourceData.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="link--green">{sourceData.name}
                    </a>,
            response_details: 'title',
            install_settings_dns: upstream,
            elapsed: formattedElapsedMs,
            filter: rule ? filter : null,
            rule_label: rule,
            response_table_header: response?.join('\n'),
            response_code: status,
            client_details: 'title',
            ip_address: client,
            name: info?.name,
            country,
            city,
            network,
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

    const isDetailed = useSelector((state) => state.queryLogs.isDetailed);

    const className = classNames('d-flex px-5 logs__row',
        `logs__row--${FILTERED_STATUS_TO_META_MAP?.[reason]?.COLOR ?? QUERY_STATUS_COLORS.WHITE}`, {
            'logs__cell--detailed': isDetailed,
        });

    return <div style={style} className={className} onClick={onClick} role="row">
        <DateCell {...rowProps} />
        <DomainCell {...rowProps} />
        <ResponseCell {...rowProps} />
        <ClientCell {...rowProps} />
    </div>;
});

Row.displayName = 'Row';

Row.propTypes = {
    style: propTypes.object,
    rowProps: propTypes.shape({
        reason: propTypes.string.isRequired,
        answer_dnssec: propTypes.bool.isRequired,
        client: propTypes.string.isRequired,
        domain: propTypes.string.isRequired,
        elapsedMs: propTypes.string.isRequired,
        info: propTypes.oneOfType([
            propTypes.string,
            propTypes.shape({
                whois_info: propTypes.shape({
                    country: propTypes.string,
                    city: propTypes.string,
                    orgname: propTypes.string,
                }),
            })]),
        response: propTypes.array.isRequired,
        time: propTypes.string.isRequired,
        tracker: propTypes.object,
        upstream: propTypes.string.isRequired,
        type: propTypes.string.isRequired,
        client_proto: propTypes.string.isRequired,
        filterId: propTypes.number,
        rule: propTypes.string,
        originalResponse: propTypes.array,
        status: propTypes.string.isRequired,
        service_name: propTypes.string,
    }).isRequired,
    isSmallScreen: propTypes.bool.isRequired,
    setDetailedDataCurrent: propTypes.func.isRequired,
    setButtonType: propTypes.func.isRequired,
    setModalOpened: propTypes.func.isRequired,
};

export default Row;
