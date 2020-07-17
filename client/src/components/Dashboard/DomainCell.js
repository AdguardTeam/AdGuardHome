import React from 'react';
import PropTypes from 'prop-types';

import { Trans } from 'react-i18next';
import { getSourceData, getTrackerData } from '../../helpers/trackers/trackers';
import Tooltip from '../ui/Tooltip';
import { captitalizeWords } from '../../helpers/helpers';

const renderLabel = (value) => <strong><Trans>{value}</Trans></strong>;

const renderLink = ({ url, name }) => <a
    className="tooltip-custom__content-link"
    target="_blank"
    rel="noopener noreferrer"
    href={url}
>
    <strong>{name}</strong>
</a>;


const getTrackerInfo = (trackerData) => [{
    key: 'name_table_header',
    value: trackerData,
    render: renderLink,
},
{
    key: 'category_label',
    value: captitalizeWords(trackerData.category),
    render: renderLabel,
},
{
    key: 'source_label',
    value: getSourceData(trackerData),
    render: renderLink,
}];

const DomainCell = ({ value }) => {
    const trackerData = getTrackerData(value);

    const content = trackerData && <div className="popover__list">
        <div className="tooltip-custom__content-title mb-1">
            <Trans>found_in_known_domain_db</Trans>
        </div>
        {getTrackerInfo(trackerData)
            .map(({ key, value, render }) => <div
                key={key}
                className="tooltip-custom__content-item"
            >
                <Trans>{key}</Trans>: {render(value)}
            </div>)}
    </div>;

    return (
        <div className="logs__row">
            <div className="logs__text" title={value}>
                {value}
            </div>
            {trackerData
            && <Tooltip content={content} placement="top"
                        className="tooltip-container tooltip-custom--wide">
                <svg className="icons icon--24 icon--green ml-1">
                    <use xlinkHref="#privacy" />
                </svg>
            </Tooltip>}
        </div>
    );
};

DomainCell.propTypes = {
    value: PropTypes.string.isRequired,
};

renderLink.propTypes = {
    url: PropTypes.string.isRequired,
    name: PropTypes.string.isRequired,
};

export default DomainCell;
