import React from 'react';

import { Trans } from 'react-i18next';
import { getSourceData, getTrackerData } from '../../helpers/trackers/trackers';

import Tooltip from '../ui/Tooltip';

import { captitalizeWords } from '../../helpers/helpers';

const renderLabel = (value: any) => (
    <strong>
        <Trans>{value}</Trans>
    </strong>
);

interface renderLinkProps {
    url: string;
    name: string;
}

const renderLink = ({ url, name }: renderLinkProps) => (
    <a className="tooltip-custom__content-link" target="_blank" rel="noopener noreferrer" href={url}>
        <strong>{name}</strong>
    </a>
);

const getTrackerInfo = (trackerData: any) => [
    {
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
    },
];

interface DomainCellProps {
    value: string;
}

const DomainCell = ({ value }: DomainCellProps) => {
    const trackerData = getTrackerData(value);

    const content = trackerData && (
        <div className="popover__list">
            <div className="tooltip-custom__content-title mb-1">
                <Trans>found_in_known_domain_db</Trans>
            </div>
            {getTrackerInfo(trackerData).map(({ key, value, render }) => (
                <div key={key} className="tooltip-custom__content-item">
                    <Trans>{key}</Trans>: {render(value)}
                </div>
            ))}
        </div>
    );

    return (
        <div className="logs__row">
            <div className="logs__text" title={value}>
                {value}
            </div>
            {trackerData && (
                <Tooltip content={content} placement="top" className="tooltip-container tooltip-custom--wide">
                    <svg className="icons icon--24 icon--green ml-1">
                        <use xlinkHref="#privacy" />
                    </svg>
                </Tooltip>
            )}
        </div>
    );
};

export default DomainCell;
