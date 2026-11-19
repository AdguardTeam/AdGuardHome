import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

import s from './StatusError.module.pcss';

type StatusErrorProps = {
    message: string;
    processing: boolean;
    onRetry: () => void;
};

export const StatusError = (props: StatusErrorProps) => {
    return (
        <div class={s.root} data-testid="status-error">
            <div class={cn(theme.title.h0, s.face)} aria-hidden="true">
                :(
            </div>
            <div class={cn(theme.text.t3, s.message)}>{props.message}</div>
            <button
                type="button"
                class={cn(theme.text.t3, theme.link.link, theme.link.noDecoration)}
                disabled={props.processing}
                onClick={() => props.onRetry()}
            >
                {intl.getMessage('try_again')}
            </button>
        </div>
    );
};
