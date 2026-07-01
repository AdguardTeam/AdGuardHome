import { type JSX } from 'solid-js';
import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';

import s from './Banner.module.pcss';
import theme from 'panel/lib/theme';

export type BannerVariant = 'info' | 'warning' | 'critical';

type Props = {
    variant: BannerVariant;
    message: JSX.Element;
    action?: JSX.Element;
    onClose?: () => void;
    'data-testid'?: string;
};

export const Banner = (props: Props) => (
    <div
        class={cn(s.banner, s[props.variant])}
        role={props.variant === 'critical' ? 'alert' : 'status'}
        aria-live={props.variant === 'critical' ? 'assertive' : 'polite'}
        data-testid={props['data-testid']}
    >
        <div class={cn(s.message, theme.text.t3)}>{props.message}</div>
        {props.action && <div class={s.action}>{props.action}</div>}
        {props.onClose && (
            <button
                type="button"
                class={s.close}
                onClick={props.onClose}
                aria-label="Close notification"
                data-testid={`${props['data-testid']}-close`}
            >
                <Icon icon="cross" class={s.closeIcon} />
            </button>
        )}
    </div>
);
