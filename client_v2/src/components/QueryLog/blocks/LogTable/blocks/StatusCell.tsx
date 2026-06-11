import React from 'react';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { LogEntry } from 'panel/components/QueryLog/types';
import {
    getQueryStatusLabel,
    getQueryStatusDetails,
    getQueryStatusKey,
    getStatusClassName,
} from 'panel/components/QueryLog/helpers';

import s from '../LogTable.module.pcss';

type Props = {
    row: LogEntry;
};

export const StatusCell = ({ row }: Props) => {
    const statusKey = getQueryStatusKey(row.reason, row.originalResponse ?? []);

    return (
        <div className={s.statusCell}>
            <span className={cn(s.status, getStatusClassName(row.reason), theme.text.t3)}>
                {getQueryStatusLabel(statusKey)}
            </span>
            <span className={cn(s.secondaryLine, theme.text.t4)}>
                {getQueryStatusDetails(row.elapsedMs)}
            </span>
        </div>
    );
};
