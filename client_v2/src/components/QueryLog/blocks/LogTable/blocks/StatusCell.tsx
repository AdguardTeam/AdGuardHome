import { createMemo, Show } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import type { NormalizedQueryLogItem } from 'panel/helpers/helpers';
import {
    getQueryStatusLabel,
    getQueryStatusDetails,
    getQueryStatusKey,
    getStatusClassName,
} from 'panel/components/QueryLog/helpers';

import s from '../LogTable.module.pcss';

type Props = {
    row: NormalizedQueryLogItem;
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
                <Show when={props.row.cached}>
                    {' / '}
                    <span>{intl.getMessage('query_log_cached')}</span>
                </Show>
            </span>
        </div>
    );
};
