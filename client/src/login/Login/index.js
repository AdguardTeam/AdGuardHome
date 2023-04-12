import React, { Component } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import flow from 'lodash/flow';
import { withTranslation, Trans } from 'react-i18next';

import * as actionCreators from '../../actions/login';
import logo from '../../components/ui/svg/logo.svg';
import Toasts from '../../components/Toasts';
import Footer from '../../components/ui/Footer';
import Icons from '../../components/ui/Icons';
import Form from './Form';

import './Login.css';
import '../../components/ui/Tabler.css';

class Login extends Component {
    state = {
        isForgotPasswordVisible: false,
    };

    handleSubmit = ({ username: name, password }) => {
        this.props.processLogin({ name, password });
    };

    toggleText = () => {
        this.setState((prevState) => ({
            isForgotPasswordVisible: !prevState.isForgotPasswordVisible,
        }));
    };

    render() {
        const { processingLogin } = this.props.login;
        const { isForgotPasswordVisible } = this.state;

        return (
            <div className="login">
                <div className="login__form">
                    <div className="text-center mb-6">
                        <img src={logo} className="h-6 login__logo" alt="logo" />
                    </div>
                    <Form onSubmit={this.handleSubmit} processing={processingLogin} />
                    <div className="login__info">
                        <button
                            type="button"
                            className="btn btn-link login__link"
                            onClick={this.toggleText}
                        >
                            <Trans>forgot_password</Trans>
                        </button>
                        {isForgotPasswordVisible && (
                            <div className="login__message">
                                <Trans
                                    components={[
                                        <a
                                            href="https://github.com/AdguardTeam/AdGuardHome/wiki/Configuration#password-reset"
                                            key="0"
                                            target="_blank"
                                            rel="noopener noreferrer"
                                        >
                                            link
                                        </a>,
                                    ]}
                                >
                                    forgot_password_desc
                                </Trans>
                            </div>
                        )}
                    </div>
                </div>
                <Footer />
                <Toasts />
                <Icons />
            </div>
        );
    }
}

Login.propTypes = {
    login: PropTypes.object.isRequired,
    processLogin: PropTypes.func.isRequired,
};

const mapStateToProps = ({ login, toasts }) => ({ login, toasts });

export default flow([
    withTranslation(),
    connect(
        mapStateToProps,
        actionCreators,
    ),
])(Login);
