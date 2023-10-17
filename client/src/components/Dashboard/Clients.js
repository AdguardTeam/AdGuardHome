import React, { useState } from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import { Trans, useTranslation } from 'react-i18next';

import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import classNames from 'classnames';
import Card from '../ui/Card';
import Cell from '../ui/Cell';

import { getPercent, sortIp } from '../../helpers/helpers';
import {
    BLOCK_ACTIONS,
    DASHBOARD_TABLES_DEFAULT_PAGE_SIZE,
    STATUS_COLORS,
    TABLES_MIN_ROWS,
} from '../../helpers/constants';
import { toggleClientBlock } from '../../actions/access';
import { renderFormattedClientCell } from '../../helpers/renderFormattedClientCell';
import { getStats } from '../../actions/stats';
import IconTooltip from '../Logs/Cells/IconTooltip';

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
    const allowedСlients = useSelector((state) => state.access.allowed_clients, shallowEqual);

    const [isOptionsOpened, setOptionsOpened] = useState(false);

    const toggleClientStatus = async (ip, disallowed, disallowed_rule) => {
        let confirmMessage;

        if (disallowed) {
            confirmMessage = t('client_confirm_unblock', { ip: disallowed_rule || ip });
        } else {
            confirmMessage = `${t('adg_will_drop_dns_queries')} ${t('client_confirm_block', { ip })}`;
            if (allowedСlients.length > 0) {
                confirmMessage = confirmMessage.concat(`\n\n${t('filter_allowlist', { disallowed_rule })}`);
            }
        }

        if (window.confirm(confirmMessage)) {
            await dispatch(toggleClientBlock(ip, disallowed, disallowed_rule));
            await dispatch(getStats());
        }
    };

    const onClick = () => {
        toggleClientStatus(ip, disallowed, disallowed_rule);
        setOptionsOpened(false);
    };

    const text = disallowed ? BLOCK_ACTIONS.UNBLOCK : BLOCK_ACTIONS.BLOCK;

    const lastRuleInAllowlist = !disallowed && allowedСlients === disallowed_rule;
    const disabled = processingSet || lastRuleInAllowlist;
    return (
        <div className="table__action">
            <button
                type="button"
                className="btn btn-icon btn-sm px-0"
                onClick={() => setOptionsOpened(true)}
            >
                <svg className="icon24 icon--lightgray button-action__icon">
                    <use xlinkHref="#bullets" />
                </svg>
            </button>
            {isOptionsOpened && (
                <IconTooltip
                    className="icon24"
                    tooltipClass="button-action--arrow-option-container"
                    xlinkHref="bullets"
                    triggerClass="btn btn-icon btn-sm px-0 button-action__hidden-trigger"
                    content={(
                        <button
                            className={classNames('button-action--arrow-option px-4 py-1', disallowed ? 'bg--green' : 'bg--danger')}
                            onClick={onClick}
                            disabled={disabled}
                            title={lastRuleInAllowlist ? t('last_rule_in_allowlist', { disallowed_rule }) : ''}
                        >
                            <Trans>{text}</Trans>
                        </button>
                    )}
                    placement="bottom-end"
                    trigger="click"
                    onVisibilityChange={setOptionsOpened}
                    defaultTooltipShown={true}
                    delayHide={0}
                />
            )}
        </div>
    );
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

    return (
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
                        Header: <Trans>client_table_header</Trans>,
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
                minRows={TABLES_MIN_ROWS}
                defaultPageSize={DASHBOARD_TABLES_DEFAULT_PAGE_SIZE}
                className="-highlight card-table-overflow--limited clients__table"
                getTrProps={(_state, rowInfo) => {
                    if (!rowInfo) {
                        return {};
                    }

                    const { info: { disallowed } } = rowInfo.original;

                    return disallowed ? { className: 'logs__row--red' } : {};
                }}
            />
        </Card>
    );
};

Clients.propTypes = {
    refreshButton: PropTypes.node.isRequired,
    subtitle: PropTypes.string.isRequired,
};

export default Clients;
