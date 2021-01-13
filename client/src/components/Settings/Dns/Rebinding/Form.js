import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, useTranslation } from 'react-i18next';
import { shallowEqual, useSelector } from 'react-redux';
import { renderTextareaField, CheckboxField } from '../../../../helpers/form';
import { removeEmptyLines } from '../../../../helpers/helpers';
import { FORM_NAME } from '../../../../helpers/constants';

const fields = [
    {
        id: 'rebinding_allowed_hosts',
        title: 'rebinding_allowed_hosts_title',
        subtitle: 'rebinding_allowed_hosts_desc',
        normalizeOnBlur: removeEmptyLines,
    },
];

const Form = ({
    handleSubmit, submitting, invalid,
}) => {
    const { t } = useTranslation();
    const { processingSetConfig } = useSelector((state) => state.dnsConfig, shallowEqual);

    const renderField = ({
        id, title, subtitle, disabled = processingSetConfig, normalizeOnBlur,
    }) => <div key={id} className="form__group mb-5">
            <label className="form__label form__label--with-desc" htmlFor={id}>
                <Trans>{title}</Trans>
            </label>
            <div className="form__desc form__desc--top">
                <Trans>{subtitle}</Trans>
            </div>
            <Field
                id={id}
                name={id}
                component={renderTextareaField}
                type="text"
                className="form-control form-control--textarea font-monospace"
                disabled={disabled}
                normalizeOnBlur={normalizeOnBlur}
            />
        </div>;

    renderField.propTypes = {
        id: PropTypes.string,
        title: PropTypes.string,
        subtitle: PropTypes.string,
        disabled: PropTypes.bool,
        normalizeOnBlur: PropTypes.func,
    };

    return (
        <form onSubmit={handleSubmit}>
            <div className="col-12">
                <div className="form__group form__group--settings">
                    <Field
                        name={'rebinding_protection_enabled'}
                        type="checkbox"
                        component={CheckboxField}
                        placeholder={t('rebinding_protection_enabled')}
                        subtitle={t('rebinding_protection_enabled_desc')}
                        disabled={processingSetConfig}
                    />
                </div>
            </div>

            {fields.map(renderField)}

            <div className="card-actions">
                <div className="btn-list">
                    <button
                        type="submit"
                        className="btn btn-success btn-standard"
                        disabled={submitting || invalid || processingSetConfig}
                    >
                        <Trans>save_config</Trans>
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
};

export default reduxForm({ form: FORM_NAME.REBINDING })(Form);
