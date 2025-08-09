import React, { useEffect, useState } from 'react';
import { Trans } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { TOAST_TIMEOUTS } from '../../helpers/constants';

import { removeToast } from '../../actions';
import { Icon } from 'panel/common/ui/Icon';

interface ToastProps {
    id: string;
    message: string;
    type: string;
    options?: object;
}

const Toast = ({ id, message, type, options }: ToastProps) => {
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

    return (
        <div className="toast" onMouseOver={clearRemoveToastTimeout} onMouseOut={setRemoveToastTimeout}>
            <Icon icon={type === 'success' ? 'check' : 'cross'} className={`toast__icon toast__icon_${type}`} />

            <div className="toast__content">
                <Trans i18nKey={message} {...options} />
            </div>
        </div>
    );
};

export default Toast;
