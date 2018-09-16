import { connect } from 'react-redux';
import React from 'react';
import PropTypes from 'prop-types';
import { CSSTransition, TransitionGroup } from 'react-transition-group';
import * as actionCreators from '../../actions';
import Toast from './Toast';

import './Toast.css';

const Toasts = props => (
    <TransitionGroup className="toasts">
        {props.toasts.notices && props.toasts.notices.map((toast) => {
            const { id } = toast;
            return (
                <CSSTransition
                    key={id}
                    timeout={500}
                    classNames="toast"
                >
                    <Toast removeToast={props.removeToast} {...toast} />
                </CSSTransition>
            );
        })}
    </TransitionGroup>
);

Toasts.propTypes = {
    toasts: PropTypes.object,
    removeToast: PropTypes.func,
};

const mapStateToProps = (state) => {
    const { toasts } = state;
    const props = { toasts };
    return props;
};

export default connect(
    mapStateToProps,
    actionCreators,
)(Toasts);

