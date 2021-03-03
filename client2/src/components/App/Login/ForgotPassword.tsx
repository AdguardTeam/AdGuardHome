import React, { FC, useContext } from 'react';
import { Button } from 'antd';
import cn from 'classnames';

import { CommonLayout } from 'Common/ui/layouts';
import { code } from 'Common/formating';
import { Link } from 'Common/ui';
import Store from 'Store';
import theme from 'Lib/theme';

import s from './Login.module.pcss';
import { RoutePath } from '../Routes/Paths';

const ForgotPassword: FC = () => {
    const store = useContext(Store);
    const { ui: { intl } } = store;

    return (
        <CommonLayout className={cn(theme.content.content, theme.content.content_auth)}>
            <div className={cn(theme.content.container, theme.content.container_auth)}>
                <div className={s.title}>
                    {intl.getMessage('login_password_title')}
                </div>

                <p className={s.paragraph}>
                    {intl.getMessage('login_password_hash')}
                </p>

                <div className={s.list}>
                    <div className={s.step}>
                        {intl.getMessage('login_password_step_1')}
                    </div>
                    <div className={s.step}>
                        {intl.getMessage('login_password_step_2', { code })}
                    </div>
                    <div className={s.step}>
                        {intl.getMessage('login_password_step_3', { code })}
                    </div>
                    <div className={s.step}>
                        {intl.getMessage('login_password_step_4')}
                    </div>
                    <div className={s.step}>
                        {intl.getMessage('login_password_step_5')}
                    </div>
                </div>

                <p className={s.paragraph}>
                    {intl.getMessage('login_password_result')}
                </p>

                <Link to={RoutePath.Login}>
                    <Button
                        type="primary"
                        size="large"
                        block
                    >
                        {intl.getMessage('back')}
                    </Button>
                </Link>
            </div>
        </CommonLayout>
    );
};

export default ForgotPassword;
