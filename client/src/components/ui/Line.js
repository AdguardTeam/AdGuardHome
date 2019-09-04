import React from 'react';
import PropTypes from 'prop-types';
import { ResponsiveLine } from '@nivo/line';

import './Line.css';

const Line = ({ data, color }) => (
    data &&
        <ResponsiveLine
            data={data}
            margin={{
                top: 15,
                right: 0,
                bottom: 1,
                left: 20,
            }}
            minY="auto"
            stacked={false}
            curve='linear'
            axisBottom={null}
            axisLeft={null}
            enableGridX={false}
            enableGridY={false}
            enableDots={false}
            enableArea={true}
            animate={false}
            colorBy={() => (color)}
            tooltip={slice => (
                <div>
                    {slice.data.map(d => (
                        <div key={d.serie.id} className="line__tooltip">
                            <span className="line__tooltip-text">
                                <strong>{d.data.y}</strong>
                                <br/>
                                <small>{d.data.x}</small>
                            </span>
                        </div>
                    ))}
                </div>
            )}
            theme={{
                tooltip: {
                    container: {
                        padding: '0',
                        background: '#333',
                        borderRadius: '4px',
                    },
                },
            }}
        />
);

Line.propTypes = {
    data: PropTypes.array.isRequired,
    color: PropTypes.string.isRequired,
};

export default Line;
