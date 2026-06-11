import React, { useEffect, useRef } from 'react';
import type { ReactNode } from 'react';
import { useDispatch } from 'react-redux';
import { Icon } from 'panel/common/ui/Icon';
import cn from 'clsx';
import { getUndoCallback, clearUndoCallback } from 'panel/actions/toasts';
import { TOAST_TIMEOUTS } from '../../helpers/constants';

import { removeToast } from '../../actions';
import s from './styles.module.pcss';

type ToastAction = {
    text: string;
    actionType?: string;
    actionPayload?: any;
    callback?: () => void;
};

type ToastProps = {
    id: string;
    message: ReactNode;
    type: string;
    actionLabel?: string;
    undoId?: string;
    action?: ToastAction;
    code?: string;
};

const Toast = ({ id, message, type, actionLabel, undoId, action, code }: ToastProps) => {
    const dispatch = useDispatch();
    const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    const removeCurrentToast = () => dispatch(removeToast(id));
    const setRemoveToastTimeout = () => {
        const timeout = TOAST_TIMEOUTS[type];
        timerRef.current = setTimeout(removeCurrentToast, timeout);
    };

    useEffect(() => {
        setRemoveToastTimeout();
        return () => {
            if (timerRef.current) {
                clearTimeout(timerRef.current);
            }
        };
    }, []);

    const clearRemoveToastTimeout = () => {
        if (timerRef.current) {
            clearTimeout(timerRef.current);
        }
    };

    const resetRemoveToastTimeout = () => {
        clearRemoveToastTimeout();
        setRemoveToastTimeout();
    };

    const handleAction = async () => {
        clearRemoveToastTimeout();

        if (undoId) {
            const onUndo = getUndoCallback(undoId);
            if (onUndo) {
                await onUndo();
                clearUndoCallback(undoId);
            }
        }

        removeCurrentToast();
    };

    const handleActionClick = () => {
        if (action) {
            if (action.callback) {
                action.callback();
            } else if (action.actionType) {
                dispatch({ type: action.actionType, payload: action.actionPayload });
            }

            removeCurrentToast();
        }
    };

    return (
        <div
            className={s.toast}
            data-testid="toast"
            data-toast-type={type}
            data-toast-code={code}
            onMouseOver={clearRemoveToastTimeout}
            onMouseOut={resetRemoveToastTimeout}
        >
            <div className={s.messageRow}>
                <Icon
                    icon={type === 'success' ? 'check' : 'attention'}
                    className={cn(s.icon, s[type])}
                />

                <div className={s.content}>{message}</div>
            </div>

            {actionLabel && (
                <button
                    type="button"
                    className={s.actionButton}
                    data-testid="toast-action"
                    onClick={handleAction}
                >
                    {actionLabel}
                </button>
            )}

            {action && !actionLabel && (
                <button type="button" className={s.actionButton} onClick={handleActionClick}>
                    {action.text}
                </button>
            )}
        </div>
    );
};

export default Toast;
