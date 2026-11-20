import { For, Show } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

import s from './StatusError.module.pcss';

type StatusErrorProps = {
    message: string;
    processing: boolean;
    /** Explains what the user can do, when the cause of the failure is known. */
    reasons?: string[];
    action?: string;
    /** Omitted when retrying cannot help, e.g. a DNS server that failed to start. */
    onRetry?: () => void;
};

export const StatusError = (props: StatusErrorProps) => {
    return (
        <div class={s.root} data-testid="status-error">
            <div class={cn(theme.title.h0, s.face)} aria-hidden="true">
                :(
            </div>
            <div class={cn(theme.text.t3, s.message)}>{props.message}</div>
            <Show when={props.reasons?.length || props.action}>
                <div class={s.details}>
                    <Show when={props.reasons?.length}>
                        <div class={cn(theme.text.t3, s.reasonsLabel)}>
                            {intl.getMessage('dns_start_error_reasons')}
                        </div>
                        <ul class={cn(theme.text.t3, s.reasons)}>
                            <For each={props.reasons ?? []}>{(reason) => <li>{reason}</li>}</For>
                        </ul>
                    </Show>
                    <Show when={props.action}>
                        <div class={theme.text.t3}>{props.action}</div>
                    </Show>
                </div>
            </Show>
            <Show when={props.onRetry}>
                <button
                    type="button"
                    class={cn(theme.text.t3, theme.link.link, theme.link.noDecoration)}
                    disabled={props.processing}
                    onClick={() => props.onRetry?.()}
                >
                    {intl.getMessage('try_again')}
                </button>
            </Show>
        </div>
    );
};
