import React, { Component } from 'react';
import { connect } from 'react-redux';
import flow from 'lodash/flow';
import { withTranslation, Trans } from 'react-i18next';

import * as actionCreators from '../../actions/login';

import { Logo } from '../../components/ui/svg/logo';

import Toasts from '../../components/Toasts';

import Footer from '../../components/ui/Footer';

import Icons from '../../components/ui/Icons';

import Form from './Form';

import './Login.css';
import '../../components/ui/Tabler.css';

type LoginProps = {
    login: {
        processingLogin: boolean;
    };
    processLogin: (args: { name: string; password: string }) => unknown;
};

type LoginState = {
    isForgotPasswordVisible: boolean;
};

class Login extends Component<LoginProps, LoginState> {
    state = {
        isForgotPasswordVisible: false,
    };

    handleSubmit = ({ username: name, password }: { username: string; password: string }) => {
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
                        <Logo className="h-6 login__logo" />
                    </div>

                    <Form onSubmit={this.handleSubmit} processing={processingLogin} />

                    <div className="login__info">
                        <button type="button" className="btn btn-link login__link" onClick={this.toggleText}>
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
                                            rel="noopener noreferrer">
                                            link
                                        </a>,
                                    ]}>
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

const mapStateToProps = ({ login, toasts }: any) => ({ login, toasts });

export default flow([withTranslation(), connect(mapStateToProps, actionCreators)])(Login);
