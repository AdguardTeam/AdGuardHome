import React from 'react';
import { useSelector, shallowEqual } from 'react-redux';
import { CSSTransition, TransitionGroup } from 'react-transition-group';
import { TOAST_TRANSITION_TIMEOUT } from '../../helpers/constants';
import Toast from './Toast';
import './Toast.css';

const Toasts = () => {
    const toasts = useSelector((state) => state.toasts, shallowEqual);

    return <TransitionGroup className="toasts">
        {toasts.notices?.map((toast) => {
            const { id } = toast;
            return <CSSTransition
                    key={id}
                    timeout={TOAST_TRANSITION_TIMEOUT}
                    classNames="toast"
            >
                <Toast {...toast} />
            </CSSTransition>;
        })}
    </TransitionGroup>;
};

export default Toasts;
