import React from 'react';
import { ResponsiveLine } from '@nivo/line';
import PropTypes from 'prop-types';

import Card from '../ui/Card';

const Statistics = props => (
    <Card title="Statistics" subtitle="for the last 24 hours" bodyType="card-graph" refresh={props.refreshButton}>
        {props.history ?
            <ResponsiveLine
                data={props.history}
                margin={{
                    top: 50,
                    right: 40,
                    bottom: 80,
                    left: 80,
                }}
                minY="auto"
                stacked={false}
                curve='monotoneX'
                axisBottom={{
                    orient: 'bottom',
                    tickSize: 5,
                    tickPadding: 5,
                    tickRotation: -45,
                    legendOffset: 50,
                    legendPosition: 'center',
                }}
                axisLeft={{
                    orient: 'left',
                    tickSize: 5,
                    tickPadding: 5,
                    tickRotation: 0,
                    legendOffset: -40,
                    legendPosition: 'center',
                }}
                enableArea={true}
                dotSize={10}
                dotColor="inherit:darker(0.3)"
                dotBorderWidth={2}
                dotBorderColor="#ffffff"
                dotLabel="y"
                dotLabelYOffset={-12}
                animate={true}
                motionStiffness={90}
                motionDamping={15}
            />
            :
            <h2 className="text-muted">Empty data</h2>
        }
    </Card>
);

Statistics.propTypes = {
    history: PropTypes.array.isRequired,
    refreshButton: PropTypes.node,
};

export default Statistics;
