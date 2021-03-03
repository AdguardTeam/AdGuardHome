import React, { FC, useContext } from 'react';
import { Button } from 'antd';
import { observer } from 'mobx-react-lite';
import { Formik, FormikHelpers } from 'formik';
import cn from 'classnames';

import { Input } from 'Common/controls';
import { CommonLayout } from 'Common/ui/layouts';
import { Link } from 'Common/ui';
import { RoutePath } from 'Components/App/Routes/Paths';
import Store from 'Store';
import theme from 'Lib/theme';

import s from './Login.module.pcss';

type FormValues = {
    name: string;
    password: string;
};

const Login: FC = observer(() => {
    const store = useContext(Store);
    const { ui: { intl }, login } = store;

    const onSubmit = async (values: FormValues, { setSubmitting }: FormikHelpers<FormValues>) => {
        const { name, password } = values;

        const error = await login.login({
            name,
            password,
        });

        if (error) {
            setSubmitting(false);
        }
    };

    const initialValues: FormValues = {
        name: '',
        password: '',
    };

    return (
        <CommonLayout className={cn(theme.content.content, theme.content.content_auth)}>
            <div className={cn(theme.content.container, theme.content.container_auth)}>
                <div className={cn(s.title, s.title_form)}>
                    {intl.getMessage('login')}
                </div>

                <Formik
                    initialValues={initialValues}
                    onSubmit={onSubmit}
                >
                    {({
                        values,
                        handleSubmit,
                        setFieldValue,
                        isSubmitting,
                    }) => (
                        <form noValidate onSubmit={handleSubmit}>
                            <Input
                                name="name"
                                type="text"
                                placeholder={intl.getMessage('username')}
                                value={values.name}
                                onChange={(v) => setFieldValue('name', v)}
                                autoFocus
                            />
                            <Input
                                name="password"
                                type="password"
                                placeholder={intl.getMessage('password')}
                                value={values.password}
                                onChange={(v) => setFieldValue('password', v)}
                            />
                            <Button
                                type="primary"
                                size="large"
                                htmlType="submit"
                                disabled={!values.name || !values.password || isSubmitting}
                                block
                            >
                                {intl.getMessage('sign_in')}
                            </Button>
                        </form>
                    )}
                </Formik>

                <div className={theme.text.center}>
                    <Link
                        to={RoutePath.ForgotPassword}
                        className={cn(theme.link.link, theme.link.gray, s.link)}
                    >
                        {intl.getMessage('login_password_link')}
                    </Link>
                </div>
            </div>
        </CommonLayout>
    );
});

export default Login;
