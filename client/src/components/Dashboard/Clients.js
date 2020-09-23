import React from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import { Trans, useTranslation } from 'react-i18next';

import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import classNames from 'classnames';
import Card from '../ui/Card';
import Cell from '../ui/Cell';

import { getPercent, getIpMatchListStatus, sortIp } from '../../helpers/helpers';
import { BLOCK_ACTIONS, IP_MATCH_LIST_STATUS, STATUS_COLORS } from '../../helpers/constants';
import { toggleClientBlock } from '../../actions/access';
import { renderFormattedClientCell } from '../../helpers/renderFormattedClientCell';

const getClientsPercentColor = (percent) => {
    if (percent > 50) {
        return STATUS_COLORS.green;
    }
    if (percent > 10) {
        return STATUS_COLORS.yellow;
    }
    return STATUS_COLORS.red;
};

const CountCell = (row) => {
    const { value, original: { ip } } = row;
    const numDnsQueries = useSelector((state) => state.stats.numDnsQueries, shallowEqual);

    const percent = getPercent(numDnsQueries, value);
    const percentColor = getClientsPercentColor(percent);

    return <Cell value={value} percent={percent} color={percentColor} search={ip} />;
};

const renderBlockingButton = (ip) => {
    const dispatch = useDispatch();
    const { t } = useTranslation();
    const processingSet = useSelector((state) => state.access.processingSet);
    const disallowed_clients = useSelector(
        (state) => state.access.disallowed_clients, shallowEqual,
    );

    const ipMatchListStatus = getIpMatchListStatus(ip, disallowed_clients);

    if (ipMatchListStatus === IP_MATCH_LIST_STATUS.CIDR) {
        return null;
    }

    const isNotFound = ipMatchListStatus === IP_MATCH_LIST_STATUS.NOT_FOUND;
    const type = isNotFound ? BLOCK_ACTIONS.BLOCK : BLOCK_ACTIONS.UNBLOCK;
    const text = type;

    const buttonClass = classNames('button-action button-action--main', {
        'button-action--unblock': !isNotFound,
    });

    const toggleClientStatus = (type, ip) => {
        const confirmMessage = type === BLOCK_ACTIONS.BLOCK
            ? `${t('adg_will_drop_dns_queries')} ${t('client_confirm_block', { ip })}`
            : t('client_confirm_unblock', { ip });

        if (window.confirm(confirmMessage)) {
            dispatch(toggleClientBlock(type, ip));
        }
    };

    const onClick = () => toggleClientStatus(type, ip);

    return <div className="table__action pl-4">
                <button
                        type="button"
                        className={buttonClass}
                        onClick={onClick}
                        disabled={processingSet}
                >
                    <Trans>{text}</Trans>
                </button>
            </div>;
};

const ClientCell = (row) => {
    const { value, original: { info } } = row;

    return <>
        <div className="logs__row logs__row--overflow logs__row--column d-flex align-items-center">
            {renderFormattedClientCell(value, info, true)}
            {renderBlockingButton(value)}
        </div>
    </>;
};

const Clients = ({
    refreshButton,
    subtitle,
}) => {
    const { t } = useTranslation();
    const topClients = useSelector((state) => state.stats.topClients, shallowEqual);
    const disallowedClients = useSelector((state) => state.access.disallowed_clients, shallowEqual);

    return <Card
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
                        sortMethod: sortIp,
                        Cell: ClientCell,
                    },
                    {
                        Header: <Trans>requests_count</Trans>,
                        accessor: 'count',
                        minWidth: 180,
                        maxWidth: 200,
                        Cell: CountCell,
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

                    return getIpMatchListStatus(ip, disallowedClients) === IP_MATCH_LIST_STATUS.NOT_FOUND ? {} : { className: 'logs__row--red' };
                }}
        />
    </Card>;
};

Clients.propTypes = {
    refreshButton: PropTypes.node.isRequired,
    subtitle: PropTypes.string.isRequired,
};

export default Clients;
