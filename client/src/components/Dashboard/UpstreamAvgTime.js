import React from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import round from 'lodash/round';
import { withTranslation, Trans } from 'react-i18next';

import Card from '../ui/Card';
import DomainCell from './DomainCell';
import { DASHBOARD_TABLES_DEFAULT_PAGE_SIZE, TABLES_MIN_ROWS } from '../../helpers/constants';

const TimeCell = ({ value }) => {
    if (!value) {
        return '–';
    }

    const valueInMilliseconds = round(value * 1000);

    return (
        <div className="logs__row o-hidden">
            <span className="logs__text logs__text--full" title={valueInMilliseconds}>
                {valueInMilliseconds}&nbsp;ms
            </span>
        </div>
    );
};

TimeCell.propTypes = {
    value: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.number,
    ]),
};

const UpstreamAvgTime = ({
    t,
    refreshButton,
    topUpstreamsAvgTime,
    subtitle,
}) => (
    <Card
        title={t('average_processing_time')}
        subtitle={subtitle}
        bodyType="card-table"
        refresh={refreshButton}
    >
        <ReactTable
            data={topUpstreamsAvgTime.map(({ name: domain, count }) => ({
                domain,
                count,
            }))}
            columns={[
                {
                    Header: <Trans>upstream</Trans>,
                    accessor: 'domain',
                    Cell: DomainCell,
                },
                {
                    Header: <Trans>processing_time</Trans>,
                    accessor: 'count',
                    maxWidth: 190,
                    Cell: TimeCell,
                },
            ]}
            showPagination={false}
            noDataText={t('no_upstreams_data_found')}
            minRows={TABLES_MIN_ROWS}
            defaultPageSize={DASHBOARD_TABLES_DEFAULT_PAGE_SIZE}
            className="-highlight card-table-overflow--limited stats__table"
        />
    </Card>
);

UpstreamAvgTime.propTypes = {
    topUpstreamsAvgTime: PropTypes.array.isRequired,
    refreshButton: PropTypes.node.isRequired,
    subtitle: PropTypes.string.isRequired,
    t: PropTypes.func.isRequired,
};

export default withTranslation()(UpstreamAvgTime);
