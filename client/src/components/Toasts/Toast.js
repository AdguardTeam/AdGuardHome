import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

class Toast extends Component {
    componentDidMount() {
        const timeout = this.props.type === 'error' ? 30000 : 5000;

        setTimeout(() => {
            this.props.removeToast(this.props.id);
        }, timeout);
    }

    shouldComponentUpdate() {
        return false;
    }

    render() {
        return (
            <div className={`toast toast--${this.props.type}`}>
                <p className="toast__content">
                    <Trans>{this.props.message}</Trans>
                </p>
                <button className="toast__dismiss" onClick={() => this.props.removeToast(this.props.id)}>
                    <svg stroke="#fff" fill="none" width="20" height="20" strokeWidth="2" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="m18 6-12 12"/><path d="m6 6 12 12"/></svg>
                </button>
            </div>
        );
    }
}

Toast.propTypes = {
    id: PropTypes.string.isRequired,
    message: PropTypes.string.isRequired,
    type: PropTypes.string.isRequired,
    removeToast: PropTypes.func.isRequired,
};

export default withNamespaces()(Toast);
