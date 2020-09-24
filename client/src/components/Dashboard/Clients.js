import React from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import { Trans, useTranslation } from 'react-i18next';

import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import classNames from 'classnames';
import Card from '../ui/Card';
import Cell from '../ui/Cell';

import { getPercent, sortIp } from '../../helpers/helpers';
import { BLOCK_ACTIONS, STATUS_COLORS } from '../../helpers/constants';
import { toggleClientBlock } from '../../actions/access';
import { renderFormattedClientCell } from '../../helpers/renderFormattedClientCell';
import { getStats } from '../../actions/stats';

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

const renderBlockingButton = (ip, disallowed, disallowed_rule) => {
    const dispatch = useDispatch();
    const { t } = useTranslation();
    const processingSet = useSelector((state) => state.access.processingSet);

    const buttonClass = classNames('button-action button-action--main', {
        'button-action--unblock': disallowed,
    });

    const toggleClientStatus = async (ip, disallowed, disallowed_rule) => {
        const confirmMessage = disallowed
            ? t('client_confirm_unblock', { ip: disallowed_rule })
            : `${t('adg_will_drop_dns_queries')} ${t('client_confirm_block', { ip })}`;

        if (window.confirm(confirmMessage)) {
            await dispatch(toggleClientBlock(ip, disallowed, disallowed_rule));
            await dispatch(getStats());
        }
    };

    const onClick = () => toggleClientStatus(ip, disallowed, disallowed_rule);

    const text = disallowed ? BLOCK_ACTIONS.UNBLOCK : BLOCK_ACTIONS.BLOCK;

    const isNotInAllowedList = disallowed && disallowed_rule === '';
    return <div className="table__action pl-4">
        <button
                type="button"
                className={buttonClass}
                onClick={isNotInAllowedList ? undefined : onClick}
                disabled={isNotInAllowedList || processingSet}
                title={t(isNotInAllowedList ? 'client_not_in_allowed_clients' : text)}
        >
            <Trans>{text}</Trans>
        </button>
    </div>;
};

const ClientCell = (row) => {
    const { value, original: { info, info: { disallowed, disallowed_rule } } } = row;

    return <>
        <div className="logs__row logs__row--overflow logs__row--column d-flex align-items-center">
            {renderFormattedClientCell(value, info, true)}
            {renderBlockingButton(value, disallowed, disallowed_rule)}
        </div>
    </>;
};

const Clients = ({
    refreshButton,
    subtitle,
}) => {
    const { t } = useTranslation();
    const topClients = useSelector((state) => state.stats.topClients, shallowEqual);

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

                    const { info: { disallowed } } = rowInfo.original;

                    return disallowed ? { className: 'logs__row--red' } : {};
                }}
        />
    </Card>;
};

Clients.propTypes = {
    refreshButton: PropTypes.node.isRequired,
    subtitle: PropTypes.string.isRequired,
};

export default Clients;
