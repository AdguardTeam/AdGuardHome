import React, { useEffect, useState } from 'react';
import { useDispatch } from 'react-redux';
import { Icon } from 'panel/common/ui/Icon';
import cn from 'clsx';
import { TOAST_TIMEOUTS } from '../../helpers/constants';

import { removeToast } from '../../actions';
import s from './styles.module.pcss';

interface ToastProps {
    id: string;
    message: string;
    type: string;
}

const Toast = ({ id, message, type }: ToastProps) => {
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
        <div className={s.toast} onMouseOver={clearRemoveToastTimeout} onMouseOut={setRemoveToastTimeout}>
            <Icon icon={type === 'success' ? 'check' : 'attention'} className={cn(s.icon, s[type])} />

            <div className={s.content}>{message}</div>
        </div>
    );
};

export default Toast;
