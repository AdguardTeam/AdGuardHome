import React from 'react';
import classNames from 'classnames';
import PropTypes from 'prop-types';
import getHintElement from './getHintElement';
import {
    DEFAULT_SHORT_DATE_FORMAT_OPTIONS,
    LONG_TIME_FORMAT,
    SCHEME_TO_PROTOCOL_MAP,
} from '../../../helpers/constants';
import { captitalizeWords, formatDateTime, formatTime } from '../../../helpers/helpers';
import { getSourceData } from '../../../helpers/trackers/trackers';

const getDomainCell = (props) => {
    const {
        row, t, isDetailed, dnssec_enabled,
    } = props;

    const {
        tracker, type, answer_dnssec, client_proto, domain, time,
    } = row.original;

    const hasTracker = !!tracker;

    const lockIconClass = classNames('icons', 'icon--small', 'd-none', 'd-sm-block', 'cursor--pointer', {
        'icon--active': answer_dnssec,
        'icon--disabled': !answer_dnssec,
        'my-3': isDetailed,
    });

    const privacyIconClass = classNames('icons', 'mx-2', 'icon--small', 'd-none', 'd-sm-block', 'cursor--pointer', {
        'icon--active': hasTracker,
        'icon--disabled': !hasTracker,
        'my-3': isDetailed,
    });

    const protocol = t(SCHEME_TO_PROTOCOL_MAP[client_proto]) || '';
    const ip = type ? `${t('type_table_header')}: ${type}` : '';

    const requestDetailsObj = {
        time_table_header: formatTime(time, LONG_TIME_FORMAT),
        date: formatDateTime(time, DEFAULT_SHORT_DATE_FORMAT_OPTIONS),
        domain,
        type_table_header: type,
        protocol,
    };

    const sourceData = getSourceData(tracker);

    const knownTrackerDataObj = {
        name_table_header: tracker?.name,
        category_label: hasTracker && captitalizeWords(tracker.category),
        source_label: sourceData
            && <a href={sourceData.url} target="_blank" rel="noopener noreferrer"
                  className="link--green">{sourceData.name}</a>,
    };

    const renderGrid = (content, idx) => {
        const preparedContent = typeof content === 'string' ? t(content) : content;
        const className = classNames('text-truncate key-colon o-hidden', {
            'overflow-break': preparedContent.length > 100,
        });
        return <div key={idx} className={className}>{preparedContent}</div>;
    };

    const getGrid = (contentObj, title, className) => [
        <div key={title} className={classNames('pb-2 grid--title', className)}>{t(title)}</div>,
        <div key={`${title}-1`}
             className="grid grid--limited">{React.Children.map(Object.entries(contentObj), renderGrid)}</div>,
    ];

    const requestDetails = getGrid(requestDetailsObj, 'request_details');

    const renderContent = hasTracker ? requestDetails.concat(getGrid(knownTrackerDataObj, 'known_tracker', 'pt-4')) : requestDetails;

    const trackerHint = getHintElement({
        className: privacyIconClass,
        tooltipClass: 'pt-4 pb-5 px-5 mw-75',
        xlinkHref: 'privacy',
        contentItemClass: 'key-colon',
        renderContent,
        place: 'bottom',
    });

    const valueClass = classNames('w-100', {
        'px-2 d-flex justify-content-center flex-column': isDetailed,
    });

    const details = [ip, protocol].filter(Boolean)
        .join(', ');

    return (
        <div className="logs__row o-hidden">
            {dnssec_enabled && getHintElement({
                className: lockIconClass,
                tooltipClass: 'py-4 px-5 pb-45',
                canShowTooltip: answer_dnssec,
                xlinkHref: 'lock',
                columnClass: 'w-100',
                content: 'validated_with_dnssec',
                placement: 'bottom',
            })}
            {trackerHint}
            <div className={valueClass}>
                <div className="text-truncate" title={domain}>{domain}</div>
                {details && isDetailed
                && <div className="detailed-info d-none d-sm-block text-truncate"
                        title={details}>{details}</div>}
            </div>
        </div>
    );
};

getDomainCell.propTypes = {
    row: PropTypes.object.isRequired,
    t: PropTypes.func.isRequired,
    isDetailed: PropTypes.bool.isRequired,
    toggleBlocking: PropTypes.func.isRequired,
    autoClients: PropTypes.array.isRequired,
    dnssec_enabled: PropTypes.bool.isRequired,
};

export default getDomainCell;
