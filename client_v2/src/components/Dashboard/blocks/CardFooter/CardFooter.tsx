import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Link } from 'panel/common/ui/Link';
import type { QueryParams, RoutePathKey } from 'panel/components/Routes/Paths';

import s from './CardFooter.module.pcss';

type Props = {
    to: RoutePathKey;
    testId: string;
    query?: QueryParams;
};

export const CardFooter = (props: Props) => (
    <div class={s.cardFooter}>
        <Link
            to={props.to}
            class={cn(theme.text.t3, theme.link.link, theme.link.hoverDecoration)}
            data-testid={props.testId}
            query={props.query}
        >
            {intl.getMessage('show_more')}
        </Link>
    </div>
);
