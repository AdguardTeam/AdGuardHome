import React, { Component } from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import { withNamespaces, Trans } from 'react-i18next';

import Card from '../ui/Card';
import Cell from '../ui/Cell';
import Popover from '../ui/Popover';

import { getTrackerData } from '../../helpers/trackers/trackers';
import { getPercent } from '../../helpers/helpers';
import { STATUS_COLORS } from '../../helpers/constants';

class QueriedDomains extends Component {
    getPercentColor = (percent) => {
        if (percent > 10) {
            return STATUS_COLORS.red;
        } else if (percent > 5) {
            return STATUS_COLORS.yellow;
        }
        return STATUS_COLORS.green;
    };

    columns = [
        {
            Header: <Trans>domain</Trans>,
            accessor: 'domain',
            Cell: (row) => {
                const { value } = row;
                const trackerData = getTrackerData(value);

                return (
                    <div className="logs__row">
                        <div className="logs__text" title={value}>
                            {value}
                        </div>
                        {trackerData && <Popover data={trackerData} />}
                    </div>
                );
            },
        },
        {
            Header: <Trans>requests_count</Trans>,
            accessor: 'count',
            maxWidth: 190,
            Cell: ({ value }) => {
                const percent = getPercent(this.props.dnsQueries, value);
                const percentColor = this.getPercentColor(percent);

                return <Cell value={value} percent={percent} color={percentColor} />;
            },
        },
    ];

    render() {
        const {
            t, refreshButton, topQueriedDomains, subtitle,
        } = this.props;
        return (
            <Card
                title={t('stats_query_domain')}
                subtitle={subtitle}
                bodyType="card-table"
                refresh={refreshButton}
            >
                <ReactTable
                    data={topQueriedDomains.map(item => ({
                        domain: item.name,
                        count: item.count,
                    }))}
                    columns={this.columns}
                    showPagination={false}
                    noDataText={t('no_domains_found')}
                    minRows={6}
                    className="-striped -highlight card-table-overflow stats__table"
                />
            </Card>
        );
    }
}

QueriedDomains.propTypes = {
    topQueriedDomains: PropTypes.array.isRequired,
    dnsQueries: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
    subtitle: PropTypes.string.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(QueriedDomains);
