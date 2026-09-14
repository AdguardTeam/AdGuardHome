import { Show } from 'solid-js';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { Link } from 'panel/common/ui/Link';
import { RoutePath } from 'panel/components/Routes/Paths';

import s from './NameCell.module.pcss';

type Props = {
    name: string;
    testId?: string;
    /** When set, the name becomes a QueryLog link filtered by this value. */
    queryLogSearch?: string;
};

/** Name cell with the shared truncation styles used by the top stats tables. */
export const NameCell = (props: Props) => {
    const content = (
        <span
            class={cn(theme.text.t3, theme.text.condenced, theme.common.textOverflow, s.nameCell)}
            title={props.name}
            data-testid={props.testId}
        >
            {props.name}
        </span>
    );

    return (
        <Show when={props.queryLogSearch} fallback={content}>
            <Link
                to={RoutePath.QueryLog}
                query={{ search: `"${props.queryLogSearch}"` }}
                class={cn(theme.text.t3, theme.text.condenced, s.nameCellLink)}
                title={props.name}
            >
                {content}
            </Link>
        </Show>
    );
};
