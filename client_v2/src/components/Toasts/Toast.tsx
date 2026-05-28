import React, { useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { useDispatch } from 'react-redux';
import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import cn from 'clsx';
import { TOAST_TIMEOUTS } from '../../helpers/constants';

import { removeToast } from '../../actions';
import s from './styles.module.pcss';

type ToastAction = {
    text: string;
    actionType?: string;
    actionPayload?: any;
    callback?: () => void;
}

type ToastProps = {
    id: string;
    message: ReactNode;
    type: string;
    action?: ToastAction;
    code?: string;
}

const Toast = ({ id, message, type, action, code }: ToastProps) => {
    const dispatch = useDispatch();
    const [timerId, setTimerId] = useState(null);

    const clearRemoveToastTimeout = () => clearTimeout(timerId);
    const removeCurrentToast = () => dispatch(removeToast(id));
    const setRemoveToastTimeout = () => {
        const timeout = TOAST_TIMEOUTS[type];
        const timerId = setTimeout(removeCurrentToast, timeout);

        setTimerId(timerId);
    };

    useEffect(() => {
        setRemoveToastTimeout();
    }, []);

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
            onMouseOut={setRemoveToastTimeout}
        >
            <Icon
                icon={type === 'success' ? 'check' : 'attention'}
                className={cn(s.icon, s[type])}
            />

            <div className={s.content}>
                {message}
                {action && (
                    <>
                        <br />
                        <button
                            type="button"
                            className={cn(theme.link.link, theme.link.noDecoration)}
                            onClick={handleActionClick}
                        >
                            {action.text}
                        </button>
                    </>
                )}
            </div>
        </div>
    );
};

export default Toast;
