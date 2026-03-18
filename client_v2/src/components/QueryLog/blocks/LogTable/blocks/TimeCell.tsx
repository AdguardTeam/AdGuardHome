import React from 'react';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { LogEntry } from 'panel/components/QueryLog/types';
import { formatLogDate, formatLogTime } from 'panel/components/QueryLog/helpers';

import s from '../LogTable.module.pcss';

type Props = {
    row: LogEntry;
};

export const TimeCell = ({ row }: Props) => (
    <div className={s.timeCell}>
        <span className={cn(s.time, theme.text.t3, theme.text.condenced)}>{formatLogTime(row.time)}</span>
        <span className={cn(s.secondaryLine, s.date, theme.text.t4)}>{formatLogDate(row.time)}</span>
    </div>
);
