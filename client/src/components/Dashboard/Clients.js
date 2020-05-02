import React, { Fragment } from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import Card from '../ui/Card';
import Cell from '../ui/Cell';

import { getPercent } from '../../helpers/helpers';
import { STATUS_COLORS } from '../../helpers/constants';
import { formatClientCell } from '../../helpers/formatClientCell';

const getClientsPercentColor = (percent) => {
    if (percent > 50) {
        return STATUS_COLORS.green;
    } else if (percent > 10) {
        return STATUS_COLORS.yellow;
    }
    return STATUS_COLORS.red;
};

const countCell = dnsQueries =>
    function cell(row) {
        const { value } = row;
        const percent = getPercent(dnsQueries, value);
        const percentColor = getClientsPercentColor(percent);

        return <Cell value={value} percent={percent} color={percentColor} />;
    };

const renderBlockingButton = (blocked, ip, handleClick, processing) => {
    let buttonProps = {
        className: 'btn-outline-danger',
        text: 'block_btn',
        type: 'block',
    };

    if (blocked) {
        buttonProps = {
            className: 'btn-outline-secondary',
            text: 'unblock_btn',
            type: 'unblock',
        };
    }

    return (
        <div className="table__action">
            <button
                type="button"
                className={`btn btn-sm ${buttonProps.className}`}
                onClick={() => handleClick(buttonProps.type, ip)}
                disabled={processing}
            >
                <Trans>{buttonProps.text}</Trans>
            </button>
        </div>
    );
};

const isBlockedClient = (clients, ip) => !!(clients && clients.includes(ip));

const clientCell = (t, toggleClientStatus, processing, disallowedClients) =>
    function cell(row) {
        const { value } = row;
        const blocked = isBlockedClient(disallowedClients, value);

        return (
            <Fragment>
                <div className="logs__row logs__row--overflow logs__row--column">
                    {formatClientCell(row, t)}
                </div>
                {renderBlockingButton(blocked, value, toggleClientStatus, processing)}
            </Fragment>
        );
    };

const Clients = ({
    t,
    refreshButton,
    topClients,
    subtitle,
    dnsQueries,
    toggleClientStatus,
    processingAccessSet,
    disallowedClients,
}) => (
    <Card
        title={t('top_clients')}
        subtitle={subtitle}
        bodyType="card-table"
        refresh={refreshButton}
    >
        <ReactTable
            data={topClients.map(({
                name: ip, count, info, blocked,
            }) => ({
                ip,
                count,
                info,
                blocked,
            }))}
            columns={[
                {
                    Header: 'IP',
                    accessor: 'ip',
                    sortMethod: (a, b) =>
                        parseInt(a.replace(/\./g, ''), 10) - parseInt(b.replace(/\./g, ''), 10),
                    Cell: clientCell(t, toggleClientStatus, processingAccessSet, disallowedClients),
                },
                {
                    Header: <Trans>requests_count</Trans>,
                    accessor: 'count',
                    minWidth: 180,
                    maxWidth: 200,
                    Cell: countCell(dnsQueries),
                },
            ]}
            showPagination={false}
            noDataText={t('no_clients_found')}
            minRows={6}
            defaultPageSize={100}
            className="-highlight card-table-overflow--limited clients__table"
            getTrProps={(_state, rowInfo) => {
                if (!rowInfo) {
                    return {};
                }

                const { ip } = rowInfo.original;

                if (isBlockedClient(disallowedClients, ip)) {
                    return {
                        className: 'red',
                    };
                }

                return {
                    className: '',
                };
            }}
        />
    </Card>
);

Clients.propTypes = {
    topClients: PropTypes.array.isRequired,
    dnsQueries: PropTypes.number.isRequired,
    refreshButton: PropTypes.node.isRequired,
    clients: PropTypes.array.isRequired,
    autoClients: PropTypes.array.isRequired,
    subtitle: PropTypes.string.isRequired,
    t: PropTypes.func.isRequired,
    toggleClientStatus: PropTypes.func.isRequired,
    processingAccessSet: PropTypes.bool.isRequired,
    disallowedClients: PropTypes.string.isRequired,
};

export default withNamespaces()(Clients);
