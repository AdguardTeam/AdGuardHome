import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';

import s from './EmptyState.module.pcss';

type Props = {
    message: string;
    class?: string;
    messageClass?: string;
};

export const EmptyState = (props: Props) => (
    <div class={cn(s.root, props.class)} data-testid="stats-empty-state">
        <div class={s.iconWrap}>
            <Icon icon="not_found_search" color="gray" class={s.icon} />
        </div>

        <div class={cn(s.message, theme.text.t2, props.messageClass)}>{props.message}</div>
    </div>
);
