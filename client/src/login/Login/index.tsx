import React, { useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { Trans } from 'react-i18next';

import * as actionCreators from '../../actions/login';

import { Logo } from '../../components/ui/svg/logo';
import Toasts from '../../components/Toasts';
import Footer from '../../components/ui/Footer';
import Icons from '../../components/ui/Icons';
import Form, { LoginFormValues } from './Form';

import './Login.css';
import '../../components/ui/Tabler.css';
import { LoginState } from '../../initialState';

export const Login = () => {
    const dispatch = useDispatch();
    const { processingLogin } = useSelector((state: LoginState) => state.login);
    const [isForgotPasswordVisible, setIsForgotPasswordVisible] = useState(false);

    const handleSubmit = ({ username: name, password }: LoginFormValues) => {
        dispatch(actionCreators.processLogin({ name, password }));
    };

    const toggleText = () => {
        setIsForgotPasswordVisible((prev) => !prev);
    };

    return (
        <div className="login">
            <div className="login__form">
                <div className="text-center mb-6">
                    <Logo className="h-6 login__logo" />
                </div>

                <Form onSubmit={handleSubmit} processing={processingLogin} />

                <div className="login__info">
                    <button type="button" className="btn btn-link login__link" onClick={toggleText}>
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
};
