import React, { useEffect, useState, forwardRef } from 'react';
import { Trans } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { TOAST_TIMEOUTS } from '../../helpers/constants';

import { removeToast } from '../../actions';

interface ToastProps {
    id: string;
    message: string;
    type: string;
    options?: object;
}

const Toast = forwardRef<HTMLDivElement, ToastProps>(({ id, message, type, options }, ref) => {
    const dispatch = useDispatch();
    const [timerId, setTimerId] = useState<NodeJS.Timeout | null>(null);

    const clearRemoveToastTimeout = () => {
        if (timerId) {
            clearTimeout(timerId);
        }
    };
    const removeCurrentToast = () => dispatch(removeToast(id));
    const setRemoveToastTimeout = () => {
        const timeout = TOAST_TIMEOUTS[type];
        const newTimerId = setTimeout(removeCurrentToast, timeout);

        setTimerId(newTimerId);
    };

    useEffect(() => {
        setRemoveToastTimeout();
        return () => clearRemoveToastTimeout();
    }, []);

    return (
        <div
            ref={ref}
            className={`toast toast--${type}`}
            onMouseOver={clearRemoveToastTimeout}
            onMouseOut={setRemoveToastTimeout}>
            <p className="toast__content">
                <Trans i18nKey={message} {...options} />
            </p>

            <button className="toast__dismiss" onClick={removeCurrentToast}>
                <svg
                    stroke="#fff"
                    fill="none"
                    width="20"
                    height="20"
                    strokeWidth="2"
                    viewBox="0 0 24 24"
                    xmlns="http://www.w3.org/2000/svg">
                    <path d="m18 6-12 12" />

                    <path d="m6 6 12 12" />
                </svg>
            </button>
        </div>
    );
});

Toast.displayName = 'Toast';

export default Toast;
