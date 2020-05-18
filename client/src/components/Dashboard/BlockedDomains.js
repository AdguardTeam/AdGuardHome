import React from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import { withNamespaces, Trans } from 'react-i18next';

import Card from '../ui/Card';
import Cell from '../ui/Cell';
import DomainCell from './DomainCell';

import { getPercent } from '../../helpers/helpers';
import { STATUS_COLORS } from '../../helpers/constants';

const CountCell = totalBlocked =>
    function cell(row) {
        const { value } = row;
        const percent = getPercent(totalBlocked, value);

        return <Cell value={value} percent={percent} color={STATUS_COLORS.red} />;
    };

const BlockedDomains = ({
    t,
    refreshButton,
    topBlockedDomains,
    subtitle,
    blockedFiltering,
    replacedSafebrowsing,
    replacedParental,
}) => {
    const totalBlocked = blockedFiltering + replacedSafebrowsing + replacedParental;

    return (
        <Card
            title={t('top_blocked_domains')}
            subtitle={subtitle}
            bodyType="card-table"
            refresh={refreshButton}
        >
            <ReactTable
                data={topBlockedDomains.map(({ name: domain, count }) => ({
                    domain,
                    count,
                }))}
                columns={[
                    {
                        Header: <Trans>domain</Trans>,
                        accessor: 'domain',
                        Cell: DomainCell,
                    },
                    {
                        Header: <Trans>requests_count</Trans>,
                        accessor: 'count',
                        maxWidth: 190,
                        Cell: CountCell(totalBlocked),
                    },
                ]}
                showPagination={false}
                noDataText={t('no_domains_found')}
                minRows={6}
                defaultPageSize={100}
                className="-highlight card-table-overflow--limited stats__table"
            />
        </Card>
    );
};

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
