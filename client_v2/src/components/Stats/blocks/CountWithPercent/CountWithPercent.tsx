import { Show, createMemo } from 'solid-js';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { formatCompactNumber } from 'panel/helpers/helpers';
import { computePercent } from 'panel/helpers/statistics';
import { Link } from 'panel/common/ui/Link';
import { RoutePath } from 'panel/components/Routes/Paths';

import s from './CountWithPercent.module.pcss';

type Props = {
    count: number;
    total: number;
    tone?: 'default' | 'danger';
    /** When set, the count becomes a QueryLog link filtered by this value. */
    queryLogSearch?: string;
    /** When set, renders a progress bar filled to this percent (0-100) before the count. */
    progress?: number;
};

export const CountWithPercent = (props: Props) => {
    const search = createMemo(() => props.queryLogSearch);

    // Integer percent for large shares (>= 10% or 0), one decimal for small ones.
    const percentText = createMemo(() => {
        const percent = computePercent(props.count, props.total);
        return percent >= 10 || percent === 0 ? String(Math.round(percent)) : percent.toFixed(1);
    });

    return (
        <span
            class={cn(
                theme.text.t3,
                theme.text.condenced,
                s.countCell,
                props.progress != null && s.countCellWithProgress,
                props.tone === 'danger' && s.danger,
            )}
            data-testid="stats-count-cell"
        >
            <Show when={props.progress != null}>
                <span class={s.progressBar} data-testid="stats-progress-bar">
                    <span
                        class={cn(
                            s.progressBarFill,
                            props.tone === 'danger' && s.progressBarFillDanger,
                        )}
                        style={{ width: `${props.progress}%` }}
                        data-testid="stats-progress-bar-fill"
                    />
                </span>
            </Show>
            <Show when={search()} fallback={formatCompactNumber(props.count)}>
                <Link
                    to={RoutePath.QueryLog}
                    query={{ search: `"${search()}"` }}
                    class={cn(theme.text.t3, theme.text.condenced, s.countLink)}
                >
                    {formatCompactNumber(props.count)}
                </Link>
            </Show>
            <span class={s.percent} data-testid="stats-percent">
                ({percentText()}%)
            </span>
        </span>
    );
};
