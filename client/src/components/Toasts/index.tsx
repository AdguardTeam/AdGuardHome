import React, { createRef } from 'react';
import { useSelector, shallowEqual } from 'react-redux';

import { CSSTransition, TransitionGroup } from 'react-transition-group';
import { TOAST_TRANSITION_TIMEOUT } from '../../helpers/constants';

import Toast from './Toast';
import './Toast.css';
import { RootState } from '../../initialState';

const Toasts = () => {
    const toasts = useSelector((state: RootState) => state.toasts, shallowEqual);

    return (
        <TransitionGroup className="toasts">
            {toasts.notices?.map((toast: any) => {
                const { id } = toast;
                const nodeRef = createRef<HTMLDivElement>();

                return (
                    <CSSTransition key={id} timeout={TOAST_TRANSITION_TIMEOUT} classNames="toast" nodeRef={nodeRef}>
                        <Toast ref={nodeRef} {...toast} />
                    </CSSTransition>
                );
            })}
        </TransitionGroup>
    );
};

export default Toasts;
