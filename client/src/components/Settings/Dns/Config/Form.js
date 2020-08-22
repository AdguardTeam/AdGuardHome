import React from 'react';
import PropTypes from 'prop-types';
import { shallowEqual, useSelector } from 'react-redux';
import { Field, reduxForm } from 'redux-form';
import { Trans, useTranslation } from 'react-i18next';
import {
    renderInputField,
    renderRadioField,
    renderCheckboxField,
    toNumber,
} from '../../../../helpers/form';
import {
    validateBiggerOrEqualZeroValue,
    validateIpv4,
    validateIpv6,
    validateRequiredValue,
} from '../../../../helpers/validators';
import { BLOCKING_MODES, FORM_NAME } from '../../../../helpers/constants';

const checkboxes = [
    {
        name: 'edns_cs_enabled',
        placeholder: 'edns_enable',
        subtitle: 'edns_cs_desc',
    },
    {
        name: 'dnssec_enabled',
        placeholder: 'dnssec_enable',
        subtitle: 'dnssec_enable_desc',
    },
    {
        name: 'disable_ipv6',
        placeholder: 'disable_ipv6',
        subtitle: 'disable_ipv6_desc',
    },
];

const customIps = [
    {
        description: 'blocking_ipv4_desc',
        name: 'blocking_ipv4',
        validateIp: validateIpv4,
    },
    {
        description: 'blocking_ipv6_desc',
        name: 'blocking_ipv6',
        validateIp: validateIpv6,
    },
];

const getFields = (processing, t) => Object.values(BLOCKING_MODES)
    .map((mode) => (
        <Field
            key={mode}
            name="blocking_mode"
            type="radio"
            component={renderRadioField}
            value={mode}
            placeholder={t(mode)}
            disabled={processing}
        />
    ));

const Form = ({
    handleSubmit, submitting, invalid, processing,
}) => {
    const { t } = useTranslation();
    const {
        blocking_mode,
    } = useSelector((state) => state.form[FORM_NAME.BLOCKING_MODE].values ?? {}, shallowEqual);

    return <form onSubmit={handleSubmit}>
        <div className="row">
            <div className="col-12 col-sm-6">
                <div className="form__group form__group--settings">
                    <label htmlFor="ratelimit"
                           className="form__label form__label--with-desc">
                        <Trans>rate_limit</Trans>
                    </label>
                    <div className="form__desc form__desc--top">
                        <Trans>rate_limit_desc</Trans>
                    </div>
                    <Field
                        name="ratelimit"
                        type="number"
                        component={renderInputField}
                        className="form-control"
                        placeholder={t('form_enter_rate_limit')}
                        normalize={toNumber}
                        validate={[validateRequiredValue, validateBiggerOrEqualZeroValue]}
                    />
                </div>
            </div>
            {checkboxes.map(({ name, placeholder, subtitle }) => <div className="col-12" key={name}>
                <div className="form__group form__group--settings">
                    <Field
                        name={name}
                        type="checkbox"
                        component={renderCheckboxField}
                        placeholder={t(placeholder)}
                        disabled={processing}
                        subtitle={t(subtitle)}
                    />
                </div>
            </div>)}
            <div className="col-12">
                <div className="form__group form__group--settings mb-4">
                    <label className="form__label form__label--with-desc">
                        <Trans>blocking_mode</Trans>
                    </label>
                    <div className="form__desc form__desc--top">
                        {Object.values(BLOCKING_MODES)
                            .map((mode) => (
                                <li key={mode}>
                                    <Trans>{`blocking_mode_${mode}`}</Trans>
                                </li>
                            ))}
                    </div>
                    <div className="custom-controls-stacked">
                        {getFields(processing, t)}
                    </div>
                </div>
            </div>
            {blocking_mode === BLOCKING_MODES.custom_ip && (
                <>
                    {customIps.map(({
                        description,
                        name,
                        validateIp,
                    }) => <div className="col-12 col-sm-6" key={name}>
                        <div className="form__group form__group--settings">
                            <label className="form__label form__label--with-desc"
                                   htmlFor={name}><Trans>{name}</Trans>
                            </label>
                            <div className="form__desc form__desc--top">
                                <Trans>{description}</Trans>
                            </div>
                            <Field
                                name={name}
                                component={renderInputField}
                                className="form-control"
                                placeholder={t('form_enter_ip')}
                                validate={[validateIp, validateRequiredValue]}
                            />
                        </div>
                    </div>)}
                </>
            )}
        </div>
        <button
            type="submit"
            className="btn btn-success btn-standard btn-large"
            disabled={submitting || invalid || processing}
        >
            <Trans>save_btn</Trans>
        </button>
    </form>;
};

Form.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
    processing: PropTypes.bool.isRequired,
};

export default reduxForm({ form: FORM_NAME.BLOCKING_MODE })(Form);
