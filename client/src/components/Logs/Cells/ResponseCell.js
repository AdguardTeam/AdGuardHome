import { useTranslation } from 'react-i18next';
import { shallowEqual, useSelector } from 'react-redux';
import classNames from 'classnames';
import React from 'react';
import propTypes from 'prop-types';
import {
    getRulesToFilterList,
    formatElapsedMs,
    getFilterNames,
    getServiceName,
} from '../../../helpers/helpers';
import { FILTERED_STATUS, FILTERED_STATUS_TO_META_MAP } from '../../../helpers/constants';
import IconTooltip from './IconTooltip';

const ResponseCell = ({
    elapsedMs,
    originalResponse,
    reason,
    response,
    status,
    upstream,
    rules,
    service_name,
}) => {
    const { t } = useTranslation();
    const filters = useSelector((state) => state.filtering.filters, shallowEqual);
    const whitelistFilters = useSelector((state) => state.filtering.whitelistFilters, shallowEqual);
    const isDetailed = useSelector((state) => state.queryLogs.isDetailed);

    const formattedElapsedMs = formatElapsedMs(elapsedMs, t);

    const isBlocked = reason === FILTERED_STATUS.FILTERED_BLACK_LIST
            || reason === FILTERED_STATUS.FILTERED_BLOCKED_SERVICE;

    const isBlockedByResponse = originalResponse.length > 0 && isBlocked;

    const statusLabel = t(isBlockedByResponse ? 'blocked_by_cname_or_ip' : FILTERED_STATUS_TO_META_MAP[reason]?.LABEL || reason);
    const boldStatusLabel = <span className="font-weight-bold">{statusLabel}</span>;

    const renderResponses = (responseArr) => {
        if (!responseArr || responseArr.length === 0) {
            return '';
        }

        return <div>{responseArr.map((response) => {
            const className = classNames('white-space--nowrap', {
                'overflow-break': response.length > 100,
            });

            return <div key={response} className={className}>{`${response}\n`}</div>;
        })}</div>;
    };

    const COMMON_CONTENT = {
        encryption_status: boldStatusLabel,
        install_settings_dns: upstream,
        elapsed: formattedElapsedMs,
        response_code: status,
        ...(service_name
                && { service_name: getServiceName(service_name) }
        ),
        ...(rules.length > 0
                && { rule_label: getRulesToFilterList(rules, filters, whitelistFilters) }
        ),
        response_table_header: renderResponses(response),
        original_response: renderResponses(originalResponse),
    };

    const content = rules.length > 0
        ? Object.entries(COMMON_CONTENT)
        : Object.entries({
            ...COMMON_CONTENT,
            filter: '',
        });

    const getDetailedInfo = (reason) => {
        switch (reason) {
            case FILTERED_STATUS.FILTERED_BLOCKED_SERVICE:
                if (!service_name) {
                    return formattedElapsedMs;
                }
                return getServiceName(service_name);
            case FILTERED_STATUS.FILTERED_BLACK_LIST:
            case FILTERED_STATUS.NOT_FILTERED_WHITE_LIST:
                return getFilterNames(rules, filters, whitelistFilters).join(', ');
            default:
                return formattedElapsedMs;
        }
    };

    const detailedInfo = getDetailedInfo(reason);

    return <div className="logs__cell logs__cell--response" role="gridcell">
        <IconTooltip
                className={classNames('icons mr-4 icon--24 icon--lightgray', { 'my-3': isDetailed })}
                columnClass='grid grid--limited'
                tooltipClass='px-5 pb-5 pt-4 mw-75 custom-tooltip__response-details'
                contentItemClass='text-truncate key-colon o-hidden'
                xlinkHref='question'
                title='response_details'
                content={content}
                placement='bottom'
        />
        <div className="text-truncate">
            <div className="text-truncate" title={statusLabel}>{statusLabel}</div>
            {isDetailed && <div
                    className="detailed-info d-none d-sm-block pt-1 text-truncate"
                    title={detailedInfo}>{detailedInfo}</div>}
        </div>
    </div>;
};

ResponseCell.propTypes = {
    elapsedMs: propTypes.string.isRequired,
    originalResponse: propTypes.array.isRequired,
    reason: propTypes.string.isRequired,
    response: propTypes.array.isRequired,
    status: propTypes.string.isRequired,
    upstream: propTypes.string.isRequired,
    rules: propTypes.arrayOf(propTypes.shape({
        text: propTypes.string.isRequired,
        filter_list_id: propTypes.number.isRequired,
    })),
    service_name: propTypes.string,
};

export default ResponseCell;
