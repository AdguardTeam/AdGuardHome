import { type JSX, onMount, onCleanup, Show } from 'solid-js';
import { Icon } from 'panel/common/ui/Icon';
import cn from 'clsx';
import { getUndoCallback, clearUndoCallback } from 'panel/stores/toasts';
import { TOAST_TIMEOUTS } from '../../helpers/constants';

import { removeToast } from 'panel/stores/toasts';
import s from './styles.module.pcss';

type ToastAction = {
    text: string;
    actionType?: string;
    actionPayload?: any;
    callback?: () => void;
};

type ToastProps = {
    id: string;
    message: JSX.Element;
    type: string;
    actionLabel?: string;
    undoId?: string;
    action?: ToastAction;
    code?: string;
};

const Toast = (props: ToastProps) => {
    let timerRef: ReturnType<typeof setTimeout> | null = null;

    const removeCurrentToast = () => removeToast(props.id);
    const setRemoveToastTimeout = () => {
        const timeout = TOAST_TIMEOUTS[props.type];
        timerRef = setTimeout(removeCurrentToast, timeout);
    };

    onMount(() => {
        setRemoveToastTimeout();
    });

    onCleanup(() => {
        if (timerRef) {
            clearTimeout(timerRef);
        }
    });

    const clearRemoveToastTimeout = () => {
        if (timerRef) {
            clearTimeout(timerRef);
        }
    };

    const resetRemoveToastTimeout = () => {
        clearRemoveToastTimeout();
        setRemoveToastTimeout();
    };

    const handleAction = async () => {
        clearRemoveToastTimeout();

        if (props.undoId) {
            const onUndo = getUndoCallback(props.undoId);
            if (onUndo) {
                await onUndo();
                clearUndoCallback(props.undoId);
            }
        }

        removeCurrentToast();
    };

    const handleActionClick = () => {
        if (props.action) {
            if (props.action.callback) {
                props.action.callback();
            }
        }

        removeCurrentToast();
    };

    return (
        <div
            class={s.toast}
            data-testid="toast"
            data-toast-type={props.type}
            data-toast-code={props.code}
            onMouseOver={clearRemoveToastTimeout}
            onMouseOut={resetRemoveToastTimeout}
        >
            <div class={s.messageRow}>
                <Icon
                    icon={(props.type === 'success' ? 'check' : 'attention') as any}
                    class={cn(s.icon, s[props.type])}
                />

                <div class={s.content}>{props.message}</div>
            </div>

            <Show when={props.actionLabel}>
                <button
                    type="button"
                    class={s.actionButton}
                    data-testid="toast-action"
                    onClick={handleAction}
                >
                    {props.actionLabel}
                </button>
            </Show>

            <Show when={props.action && !props.actionLabel}>
                <button type="button" class={s.actionButton} onClick={handleActionClick}>
                    {props.action!.text}
                </button>
            </Show>
        </div>
    );
};

export default Toast;
