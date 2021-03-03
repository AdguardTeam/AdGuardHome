import React, { FC, useContext } from 'react';
import cn from 'classnames';
import { observer } from 'mobx-react-lite';
import { FormikHelpers } from 'formik';

import { Input } from 'Common/controls';
import theme from 'Lib/theme';
import Store from 'Store/installStore';

import StepButtons from '../StepButtons';
import { FormValues } from '../../Install';

interface AuthProps {
    values: FormValues;
    setFieldValue: FormikHelpers<FormValues>['setFieldValue'];
}

const Auth: FC<AuthProps> = observer(({
    values,
    setFieldValue,
}) => {
    const { ui: { intl } } = useContext(Store);

    return (
        <>
            <div className={theme.install.title}>
                {intl.getMessage('install_auth_title')}
            </div>
            <div className={cn(theme.install.text, theme.install.text_block)}>
                {intl.getMessage('install_auth_description')}
            </div>
            <Input
                placeholder={intl.getMessage('login')}
                type="username"
                name="username"
                value={values.username}
                onChange={(v) => setFieldValue('username', v)}
            />
            <Input
                placeholder={intl.getMessage('password')}
                type="password"
                name="password"
                value={values.password}
                onChange={(v) => setFieldValue('password', v)}
            />
            <StepButtons
                setFieldValue={setFieldValue}
                currentStep={2}
                values={values}
            />
        </>
    );
});

export default Auth;
