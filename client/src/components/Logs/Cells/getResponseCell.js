import React from 'react';
import classNames from 'classnames';
import { formatElapsedMs } from '../../../helpers/helpers';
import {
    FILTERED_STATUS,
    FILTERED_STATUS_TO_META_MAP,
} from '../../../helpers/constants';
import getIconTooltip from './getIconTooltip';

const getResponseCell = (row, filtering, t, isDetailed, getFilterName) => {
    const {
        reason, filterId, rule, status, upstream, elapsedMs, response, originalResponse,
    } = row.original;

    const { filters, whitelistFilters } = filtering;
    const formattedElapsedMs = formatElapsedMs(elapsedMs, t);

    const isBlocked = reason === FILTERED_STATUS.FILTERED_BLACK_LIST
        || reason === FILTERED_STATUS.FILTERED_BLOCKED_SERVICE;

    const isBlockedByResponse = originalResponse.length > 0 && isBlocked;

    const statusLabel = t(isBlockedByResponse ? 'blocked_by_cname_or_ip' : FILTERED_STATUS_TO_META_MAP[reason]?.label || reason);
    const boldStatusLabel = <span className="font-weight-bold">{statusLabel}</span>;
    const filter = getFilterName(filters, whitelistFilters, filterId, t);

    const renderResponses = (responseArr) => {
        if (responseArr?.length === 0) {
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
        filter,
        rule_label: rule,
        response_table_header: renderResponses(response),
        original_response: renderResponses(originalResponse),
    };

    const content = rule
        ? Object.entries(COMMON_CONTENT)
        : Object.entries({ ...COMMON_CONTENT, filter: '' });
    const detailedInfo = isBlocked ? filter : formattedElapsedMs;

    return (
        <div className="logs__row">
            {getIconTooltip({
                className: classNames('icons mr-4 icon--24 icon--lightgray', { 'my-3': isDetailed }),
                columnClass: 'grid grid--limited',
                tooltipClass: 'px-5 pb-5 pt-4 mw-75 custom-tooltip__response-details',
                contentItemClass: 'text-truncate key-colon o-hidden',
                xlinkHref: 'question',
                title: 'response_details',
                content,
                placement: 'bottom',
            })}
            <div className="text-truncate">
                <div className="text-truncate" title={statusLabel}>{statusLabel}</div>
                {isDetailed && <div
                    className="detailed-info d-none d-sm-block pt-1 text-truncate"
                    title={detailedInfo}>{detailedInfo}</div>}
            </div>
        </div>
    );
};

export default getResponseCell;
