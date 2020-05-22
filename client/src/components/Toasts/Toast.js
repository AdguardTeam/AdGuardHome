import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';

class Toast extends Component {
    componentDidMount() {
        const timeout = this.props.type === 'success' ? 5000 : 30000;

        setTimeout(() => {
            this.props.removeToast(this.props.id);
        }, timeout);
    }

    shouldComponentUpdate() {
        return false;
    }

    showMessage(t, type, message) {
        if (type === 'notice') {
            return <span dangerouslySetInnerHTML={{ __html: t(message) }} />;
        }

        return <Trans>{message}</Trans>;
    }

    render() {
        const {
            type, id, t, message,
        } = this.props;

        return (
            <div className={`toast toast--${type}`}>
                <p className="toast__content">
                    {this.showMessage(t, type, message)}
                </p>
                <button className="toast__dismiss" onClick={() => this.props.removeToast(id)}>
                    <svg stroke="#fff" fill="none" width="20" height="20" strokeWidth="2" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path d="m18 6-12 12"/><path d="m6 6 12 12"/></svg>
                </button>
            </div>
        );
    }
}

Toast.propTypes = {
    t: PropTypes.func.isRequired,
    id: PropTypes.string.isRequired,
    message: PropTypes.string.isRequired,
    type: PropTypes.string.isRequired,
    removeToast: PropTypes.func.isRequired,
};

export default withTranslation()(Toast);
