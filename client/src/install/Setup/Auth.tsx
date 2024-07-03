import React from 'react';

import { Field, reduxForm } from 'redux-form';
import { withTranslation, Trans } from 'react-i18next';
import flow from 'lodash/flow';

import i18n from '../../i18n';

import Controls from './Controls';

import { renderInputField } from '../../helpers/form';
import { FORM_NAME } from '../../helpers/constants';
import { validatePasswordLength } from '../../helpers/validators';

const required = (value: any) => {
    if (value || value === 0) {
        return false;
    }

    return <Trans>form_error_required</Trans>;
};

const validate = (values: any) => {
    const errors: { confirm_password?: string } = {};

    if (values.confirm_password !== values.password) {
        errors.confirm_password = i18n.t('form_error_password');
    }

    return errors;
};

interface AuthProps {
    handleSubmit: (...args: unknown[]) => string;
    pristine: boolean;
    invalid: boolean;
    t: (...args: unknown[]) => string;
}

const Auth = (props: AuthProps) => {
    const { handleSubmit, pristine, invalid, t } = props;

    return (
        <form className="setup__step" onSubmit={handleSubmit}>
            <div className="setup__group">
                <div className="setup__subtitle">
                    <Trans>install_auth_title</Trans>
                </div>

                <p className="setup__desc">
                    <Trans>install_auth_desc</Trans>
                </p>

                <div className="form-group">
                    <label>
                        <Trans>install_auth_username</Trans>
                    </label>

                    <Field
                        name="username"
                        component={renderInputField}
                        type="text"
                        className="form-control"
                        placeholder={t('install_auth_username_enter')}
                        validate={[required]}
                        autoComplete="username"
                    />
                </div>

                <div className="form-group">
                    <label>
                        <Trans>install_auth_password</Trans>
                    </label>

                    <Field
                        name="password"
                        component={renderInputField}
                        type="password"
                        className="form-control"
                        placeholder={t('install_auth_password_enter')}
                        validate={[required, validatePasswordLength]}
                        autoComplete="new-password"
                    />
                </div>

                <div className="form-group">
                    <label>
                        <Trans>install_auth_confirm</Trans>
                    </label>

                    <Field
                        name="confirm_password"
                        component={renderInputField}
                        type="password"
                        className="form-control"
                        placeholder={t('install_auth_confirm')}
                        validate={[required]}
                        autoComplete="new-password"
                    />
                </div>
            </div>

            <Controls pristine={pristine} invalid={invalid} />
        </form>
    );
};

export default flow([
    withTranslation(),
    reduxForm({
        form: FORM_NAME.INSTALL,
        destroyOnUnmount: false,
        forceUnregisterOnUnmount: true,
        validate,
    }),
])(Auth);
