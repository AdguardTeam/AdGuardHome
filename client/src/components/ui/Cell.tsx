import React from 'react';

import LogsSearchLink from './LogsSearchLink';

import { formatNumber } from '../../helpers/helpers';

interface CellProps {
    value: number;
    percent: number;
    color: string;
    search?: string;
    onSearchRedirect?: (...args: unknown[]) => string;
}

const Cell = ({ value, percent, color, search }: CellProps) => (
    <div className="stats__row">
        <div className="stats__row-value mb-1">
            <strong>
                {search ? <LogsSearchLink search={search}>{formatNumber(value)}</LogsSearchLink> : formatNumber(value)}
            </strong>

            <small className="ml-3 text-muted">{percent}%</small>
        </div>

        <div className="progress progress-xs">
            <div
                className="progress-bar"
                style={{
                    width: `${percent}%`,
                    backgroundColor: color,
                }}
            />
        </div>
    </div>
);

export default Cell;
