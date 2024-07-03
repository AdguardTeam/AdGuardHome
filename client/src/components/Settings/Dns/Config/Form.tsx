import React from 'react';
import { shallowEqual, useSelector } from 'react-redux';

import { Field, reduxForm } from 'redux-form';
import { Trans, useTranslation } from 'react-i18next';
import {
    renderInputField,
    renderRadioField,
    renderTextareaField,
    CheckboxField,
    toNumber,
} from '../../../../helpers/form';
import {
    validateIpv4,
    validateIpv6,
    validateRequiredValue,
    validateIp,
    validateIPv4Subnet,
    validateIPv6Subnet,
} from '../../../../helpers/validators';

import { removeEmptyLines } from '../../../../helpers/helpers';
import { BLOCKING_MODES, FORM_NAME, UINT32_RANGE } from '../../../../helpers/constants';
import { RootState } from '../../../../initialState';

const checkboxes = [
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

const getFields = (processing: any, t: any) =>
    Object.values(BLOCKING_MODES)

        .map((mode: any) => (
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

interface ConfigFormProps {
    handleSubmit: (...args: unknown[]) => string;
    submitting: boolean;
    invalid: boolean;
    processing?: boolean;
}

const Form = ({ handleSubmit, submitting, invalid, processing }: ConfigFormProps) => {
    const { t } = useTranslation();
    const { blocking_mode, edns_cs_enabled, edns_cs_use_custom } = useSelector(
        (state: RootState) => state.form[FORM_NAME.BLOCKING_MODE].values ?? {},
        shallowEqual,
    );

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <label htmlFor="ratelimit" className="form__label form__label--with-desc">
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
                            validate={validateRequiredValue}
                            min={UINT32_RANGE.MIN}
                            max={UINT32_RANGE.MAX}
                        />
                    </div>
                </div>

                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <label htmlFor="ratelimit_subnet_len_ipv4" className="form__label form__label--with-desc">
                            <Trans>rate_limit_subnet_len_ipv4</Trans>
                        </label>

                        <div className="form__desc form__desc--top">
                            <Trans>rate_limit_subnet_len_ipv4_desc</Trans>
                        </div>

                        <Field
                            name="ratelimit_subnet_len_ipv4"
                            type="number"
                            component={renderInputField}
                            className="form-control"
                            placeholder={t('form_enter_rate_limit_subnet_len')}
                            normalize={toNumber}
                            validate={[validateRequiredValue, validateIPv4Subnet]}
                            min={0}
                            max={32}
                        />
                    </div>
                </div>

                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <label htmlFor="ratelimit_subnet_len_ipv6" className="form__label form__label--with-desc">
                            <Trans>rate_limit_subnet_len_ipv6</Trans>
                        </label>

                        <div className="form__desc form__desc--top">
                            <Trans>rate_limit_subnet_len_ipv6_desc</Trans>
                        </div>

                        <Field
                            name="ratelimit_subnet_len_ipv6"
                            type="number"
                            component={renderInputField}
                            className="form-control"
                            placeholder={t('form_enter_rate_limit_subnet_len')}
                            normalize={toNumber}
                            validate={[validateRequiredValue, validateIPv6Subnet]}
                            min={0}
                            max={128}
                        />
                    </div>
                </div>

                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <label htmlFor="ratelimit_whitelist" className="form__label form__label--with-desc">
                            <Trans>rate_limit_whitelist</Trans>
                        </label>

                        <div className="form__desc form__desc--top">
                            <Trans>rate_limit_whitelist_desc</Trans>
                        </div>

                        <Field
                            name="ratelimit_whitelist"
                            component={renderTextareaField}
                            type="text"
                            className="form-control"
                            placeholder={t('rate_limit_whitelist_placeholder')}
                            normalizeOnBlur={removeEmptyLines}
                        />
                    </div>
                </div>

                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <Field
                            name="edns_cs_enabled"
                            type="checkbox"
                            component={CheckboxField}
                            placeholder={t('edns_enable')}
                            disabled={processing}
                            subtitle={t('edns_cs_desc')}
                        />
                    </div>
                </div>

                <div className="col-12 form__group form__group--inner">
                    <div className="form__group ">
                        <Field
                            name="edns_cs_use_custom"
                            type="checkbox"
                            component={CheckboxField}
                            placeholder={t('edns_use_custom_ip')}
                            disabled={processing || !edns_cs_enabled}
                            subtitle={t('edns_use_custom_ip_desc')}
                        />
                    </div>

                    {edns_cs_use_custom && (
                        <Field
                            name="edns_cs_custom_ip"
                            component={renderInputField}
                            className="form-control"
                            placeholder={t('form_enter_ip')}
                            validate={[validateIp, validateRequiredValue]}
                        />
                    )}
                </div>

                {checkboxes.map(({ name, placeholder, subtitle }) => (
                    <div className="col-12" key={name}>
                        <div className="form__group form__group--settings">
                            <Field
                                name={name}
                                type="checkbox"
                                component={CheckboxField}
                                placeholder={t(placeholder)}
                                disabled={processing}
                                subtitle={t(subtitle)}
                            />
                        </div>
                    </div>
                ))}

                <div className="col-12">
                    <div className="form__group form__group--settings mb-4">
                        <label className="form__label form__label--with-desc">
                            <Trans>blocking_mode</Trans>
                        </label>

                        <div className="form__desc form__desc--top">
                            {Object.values(BLOCKING_MODES)

                                .map((mode: any) => (
                                    <li key={mode}>
                                        <Trans>{`blocking_mode_${mode}`}</Trans>
                                    </li>
                                ))}
                        </div>

                        <div className="custom-controls-stacked">{getFields(processing, t)}</div>
                    </div>
                </div>
                {blocking_mode === BLOCKING_MODES.custom_ip && (
                    <>
                        {customIps.map(({ description, name, validateIp }) => (
                            <div className="col-12 col-sm-6" key={name}>
                                <div className="form__group form__group--settings">
                                    <label className="form__label form__label--with-desc" htmlFor={name}>
                                        <Trans>{name}</Trans>
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
                            </div>
                        ))}
                    </>
                )}

                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <label htmlFor="blocked_response_ttl" className="form__label form__label--with-desc">
                            <Trans>blocked_response_ttl</Trans>
                        </label>

                        <div className="form__desc form__desc--top">
                            <Trans>blocked_response_ttl_desc</Trans>
                        </div>

                        <Field
                            name="blocked_response_ttl"
                            type="number"
                            component={renderInputField}
                            className="form-control"
                            placeholder={t('form_enter_blocked_response_ttl')}
                            normalize={toNumber}
                            validate={validateRequiredValue}
                            min={UINT32_RANGE.MIN}
                            max={UINT32_RANGE.MAX}
                        />
                    </div>
                </div>
            </div>

            <button
                type="submit"
                className="btn btn-success btn-standard btn-large"
                disabled={submitting || invalid || processing}>
                <Trans>save_btn</Trans>
            </button>
        </form>
    );
};

export default reduxForm<Record<string, any>, Omit<ConfigFormProps, 'invalid' | 'submitting' | 'handleSubmit'>>({
    form: FORM_NAME.BLOCKING_MODE,
})(Form);
