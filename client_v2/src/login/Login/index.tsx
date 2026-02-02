import React from 'react';
import { useDispatch, useSelector } from 'react-redux';

import intl from 'panel/common/intl';

import { PublicHeader } from 'panel/common/ui/PublicHeader';
import { Icons } from 'panel/common/ui/Icons';

import s from 'panel/common/ui/Header/Header.module.pcss';
import twosky from '../../../../.twosky.json';
import * as actionCreators from '../../actions/login';
import Toasts from '../../components/Toasts';
import Form, { LoginFormValues } from './Form';
import styles from './styles.module.pcss';

import { LoginState } from '../../initialState';

const LANGUAGES = twosky[1].languages;

export const Login = () => {
    const dispatch = useDispatch();
    const { processingLogin } = useSelector((state: LoginState) => state.login);

    const handleSubmit = ({ username: name, password }: LoginFormValues) => {
        dispatch(actionCreators.processLogin({ name, password }));
    };

    return (
        <div className={styles.loginWrapper}>
            <PublicHeader
                languages={LANGUAGES}
                dropdownClassName={s.dropdown}
                dropdownPosition="bottomRight"
            />
            <div className={styles.login}>
                <h1 className={styles.title}>
                    {intl.getMessage('login')}
                </h1>
                <Form onSubmit={handleSubmit} processing={processingLogin} />
            </div>

            <Toasts />

            <Icons />
        </div>
    );
};
