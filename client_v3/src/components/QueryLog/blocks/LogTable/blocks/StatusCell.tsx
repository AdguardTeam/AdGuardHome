import { createMemo } from 'solid-js';
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

export const StatusCell = (props: Props) => {
    const statusKey = createMemo(() =>
        getQueryStatusKey(props.row.reason, props.row.originalResponse ?? []),
    );

    return (
        <div class={s.statusCell}>
            <span class={cn(s.status, getStatusClassName(props.row.reason), theme.text.t3)}>
                {getQueryStatusLabel(statusKey())}
            </span>
            <span class={cn(s.secondaryLine, theme.text.t4)}>
                {getQueryStatusDetails(props.row.elapsedMs)}
            </span>
        </div>
    );
};
