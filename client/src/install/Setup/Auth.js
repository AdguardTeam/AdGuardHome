import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { withNamespaces, Trans } from 'react-i18next';
import flow from 'lodash/flow';

import Controls from './Controls';
import validate from './validate';
import renderField from './renderField';

const required = (value) => {
    if (value || value === 0) {
        return false;
    }
    return <Trans>form_error_required</Trans>;
};

const Auth = (props) => {
    const {
        handleSubmit,
        submitting,
        pristine,
        t,
    } = props;

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
                        component={renderField}
                        type="text"
                        className="form-control"
                        placeholder={ t('install_auth_username_enter') }
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
                        component={renderField}
                        type="password"
                        className="form-control"
                        placeholder={ t('install_auth_password_enter') }
                        validate={[required]}
                        autoComplete="new-password"
                    />
                </div>
                <div className="form-group">
                    <label>
                        <Trans>install_auth_confirm</Trans>
                    </label>
                    <Field
                        name="confirm_password"
                        component={renderField}
                        type="password"
                        className="form-control"
                        placeholder={ t('install_auth_confirm') }
                        validate={[required]}
                        autoComplete="new-password"
                    />
                </div>
            </div>
            <Controls submitting={submitting} pristine={pristine} />
        </form>
    );
};

Auth.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    pristine: PropTypes.bool.isRequired,
    submitting: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
};

export default flow([
    withNamespaces(),
    reduxForm({
        form: 'install',
        destroyOnUnmount: false,
        forceUnregisterOnUnmount: true,
        validate,
    }),
])(Auth);
