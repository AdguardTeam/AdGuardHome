import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';
import { renderInputField } from '../../helpers/form';
import { validateRequiredValue } from '../../helpers/validators';
import { FORM_NAME } from '../../helpers/constants';

const Form = (props) => {
    const {
        handleSubmit, processing, invalid, t,
    } = props;

    return (
        <form onSubmit={handleSubmit} className="card">
            <div className="card-body p-6">
                <div className="form__group form__group--settings">
                    <label className="form__label" htmlFor="username">
                        <Trans>username_label</Trans>
                    </label>
                    <Field
                        id="username1"
                        name="username"
                        type="text"
                        className="form-control"
                        component={renderInputField}
                        placeholder={t('username_placeholder')}
                        autoComplete="username"
                        disabled={processing}
                        validate={[validateRequiredValue]}
                    />
                </div>
                <div className="form__group form__group--settings">
                    <label className="form__label" htmlFor="password">
                        <Trans>password_label</Trans>
                    </label>
                    <Field
                        id="password"
                        name="password"
                        type="password"
                        className="form-control"
                        component={renderInputField}
                        placeholder={t('password_placeholder')}
                        autoComplete="current-password"
                        disabled={processing}
                        validate={[validateRequiredValue]}
                    />
                </div>
                <div className="form-footer">
                    <button
                        type="submit"
                        className="btn btn-success btn-block"
                        disabled={processing || invalid}
                    >
                        <Trans>sign_in</Trans>
                    </button>
                </div>
            </div>
        </form>
    );
};

Form.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
    processing: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
};

export default flow([
    withTranslation(),
    reduxForm({ form: FORM_NAME.LOGIN }),
])(Form);
