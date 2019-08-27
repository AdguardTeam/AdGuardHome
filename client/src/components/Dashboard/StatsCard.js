import React from 'react';
import PropTypes from 'prop-types';

import { STATUS_COLORS } from '../../helpers/constants';
import Card from '../ui/Card';
import Line from '../ui/Line';

const StatsCard = ({
    total, lineData, percent, title, color,
}) => (
    <Card type="card--full" bodyType="card-wrap">
        <div className="card-body-stats">
            <div className={`card-value card-value-stats text-${color}`}>{total}</div>
            <div className="card-title-stats">{title}</div>
        </div>
        {percent >= 0 && (<div className={`card-value card-value-percent text-${color}`}>{percent}</div>)}
        <div className="card-chart-bg">
            <Line data={lineData} color={STATUS_COLORS[color]} />
        </div>
    </Card>
);

StatsCard.propTypes = {
    total: PropTypes.number.isRequired,
    lineData: PropTypes.array.isRequired,
    title: PropTypes.object.isRequired,
    color: PropTypes.string.isRequired,
    percent: PropTypes.number,
};

export default StatsCard;
