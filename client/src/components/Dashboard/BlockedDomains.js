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

class BlockedDomains extends Component {
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
                const { blockedFiltering, replacedSafebrowsing, replacedParental } = this.props;
                const blocked = blockedFiltering + replacedSafebrowsing + replacedParental;
                const percent = getPercent(blocked, value);

                return <Cell value={value} percent={percent} color={STATUS_COLORS.red} />;
            },
        },
    ];

    render() {
        const {
            t, refreshButton, topBlockedDomains, subtitle,
        } = this.props;

        return (
            <Card
                title={t('top_blocked_domains')}
                subtitle={subtitle}
                bodyType="card-table"
                refresh={refreshButton}
            >
                <ReactTable
                    data={topBlockedDomains.map(item => ({
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

BlockedDomains.propTypes = {
    topBlockedDomains: PropTypes.array.isRequired,
    blockedFiltering: PropTypes.number.isRequired,
    replacedSafebrowsing: PropTypes.number.isRequired,
    replacedParental: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
    subtitle: PropTypes.string.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(BlockedDomains);
