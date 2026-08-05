import cn from 'clsx';

import theme from 'panel/lib/theme';
import type { NormalizedQueryLogItem } from 'panel/helpers/helpers';
import { formatLogDate, formatLogTime } from 'panel/components/QueryLog/helpers';

import s from '../LogTable.module.pcss';

type Props = {
    row: NormalizedQueryLogItem;
};

export const TimeCell = (props: Props) => (
    <div class={s.timeCell}>
        <span class={cn(s.time, theme.text.t3, theme.text.condenced)}>
            {formatLogTime(props.row.time)}
        </span>
        <span class={cn(s.secondaryLine, s.date, theme.text.t4)}>
            {formatLogDate(props.row.time)}
        </span>
    </div>
);
