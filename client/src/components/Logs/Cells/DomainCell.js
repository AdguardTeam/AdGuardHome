import React from 'react';
import { useSelector } from 'react-redux';
import classNames from 'classnames';
import propTypes from 'prop-types';
import { useTranslation } from 'react-i18next';
import {
    DEFAULT_SHORT_DATE_FORMAT_OPTIONS,
    LONG_TIME_FORMAT,
    SCHEME_TO_PROTOCOL_MAP,
} from '../../../helpers/constants';
import { captitalizeWords, formatDateTime, formatTime } from '../../../helpers/helpers';
import { getSourceData } from '../../../helpers/trackers/trackers';
import IconTooltip from './IconTooltip';

const DomainCell = ({
    answer_dnssec,
    client_proto,
    domain,
    time,
    tracker,
    type,
}) => {
    const { t } = useTranslation();
    const dnssec_enabled = useSelector((state) => state.dnsConfig.dnssec_enabled);
    const isDetailed = useSelector((state) => state.queryLogs.isDetailed);

    const hasTracker = !!tracker;

    const lockIconClass = classNames('icons icon--24 d-none d-sm-block', {
        'icon--green': answer_dnssec,
        'icon--disabled': !answer_dnssec,
        'my-3': isDetailed,
    });

    const privacyIconClass = classNames('icons mx-2 icon--24 d-none d-sm-block', {
        'icon--green': hasTracker,
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
        const className = classNames('text-truncate o-hidden', {
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

    const valueClass = classNames('w-100 text-truncate', {
        'px-2 d-flex justify-content-center flex-column': isDetailed,
    });

    const details = [ip, protocol].filter(Boolean)
        .join(', ');

    return <div className="d-flex o-hidden logs__cell logs__cell logs__cell--domain" role="gridcell">
        {dnssec_enabled && <IconTooltip
                className={lockIconClass}
                tooltipClass='py-4 px-5 pb-45'
                canShowTooltip={!!answer_dnssec}
                xlinkHref='lock'
                columnClass='w-100'
                content='validated_with_dnssec'
                placement='bottom'
        />}
        <IconTooltip className={privacyIconClass} tooltipClass='pt-4 pb-5 px-5 mw-75'
                     xlinkHref='privacy' contentItemClass='key-colon' renderContent={renderContent}
                     place='bottom' />
        <div className={valueClass}>
            <div className="text-truncate" title={domain}>{domain}</div>
            {details && isDetailed
            && <div className="detailed-info d-none d-sm-block text-truncate"
                    title={details}>{details}</div>}
        </div>
    </div>;
};

DomainCell.propTypes = {
    answer_dnssec: propTypes.bool.isRequired,
    client_proto: propTypes.string.isRequired,
    domain: propTypes.string.isRequired,
    time: propTypes.string.isRequired,
    type: propTypes.string.isRequired,
    tracker: propTypes.object,
};

export default DomainCell;
