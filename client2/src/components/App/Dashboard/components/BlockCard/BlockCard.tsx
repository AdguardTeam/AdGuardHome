import React, { FC } from 'react';
import { AreaChart, Area, ResponsiveContainer } from 'recharts';

import s from './BlockCard.module.pcss';

interface BlockCardProps {
    overal: number | string;
    data?: number[];
    text?: string;
    color?: string;
    title: string;
}

const BlockCard: FC<BlockCardProps> = ({ overal, data, color, title, text }) => {
    return (
        <div className={s.container}>
            <div className={s.title}>{title}</div>
            <div className={s.overal}>{overal}</div>
            {data && (
                <ResponsiveContainer width="100%" height={25}>
                    <AreaChart data={data.map((n) => ({ name: 'data', value: n }))}>
                        <Area dataKey="value" stroke={color} fill={color} dot={false} />
                    </AreaChart>
                </ResponsiveContainer>
            )}
            {text && (
                <div>
                    {text}
                </div>
            )}
        </div>
    );
};

export default BlockCard;
