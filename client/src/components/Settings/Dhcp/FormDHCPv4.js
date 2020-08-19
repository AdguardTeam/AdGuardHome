import React, { useCallback } from 'react';
import { shallowEqual, useSelector } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { useTranslation } from 'react-i18next';

import {
    renderInputField,
    toNumber,
} from '../../../helpers/form';
import { FORM_NAME } from '../../../helpers/constants';
import {
    validateIpv4,
    validateIsPositiveValue,
    validateRequiredValue,
    validateIpv4RangeEnd,
} from '../../../helpers/validators';

const FormDHCPv4 = ({
    handleSubmit,
    submitting,
    processingConfig,
    ipv4placeholders,
}) => {
    const { t } = useTranslation();
    const dhcp = useSelector((state) => state.form[FORM_NAME.DHCPv4], shallowEqual);
    const interfaces = useSelector((state) => state.form[FORM_NAME.DHCP_INTERFACES], shallowEqual);
    const interface_name = interfaces?.values?.interface_name;

    const isInterfaceIncludesIpv4 = useSelector(
        (state) => !!state.dhcp?.interfaces?.[interface_name]?.ipv4_addresses,
    );

    const isEmptyConfig = !Object.values(dhcp?.values?.v4 ?? {})
        .some(Boolean);

    const invalid = dhcp?.syncErrors || interfaces?.syncErrors || !isInterfaceIncludesIpv4
        || isEmptyConfig || submitting || processingConfig;

    const validateRequired = useCallback((value) => {
        if (isEmptyConfig) {
            return undefined;
        }
        return validateRequiredValue(value);
    }, [isEmptyConfig]);

    return <form onSubmit={handleSubmit}>
        <div className="row">
            <div className="col-lg-6">
                <div className="form__group form__group--settings">
                    <label>{t('dhcp_form_gateway_input')}</label>
                    <Field
                        name="v4.gateway_ip"
                        component={renderInputField}
                        type="text"
                        className="form-control"
                        placeholder={t(ipv4placeholders.gateway_ip)}
                        validate={[validateIpv4, validateRequired]}
                        disabled={!isInterfaceIncludesIpv4}
                    />
                </div>
                <div className="form__group form__group--settings">
                    <label>{t('dhcp_form_subnet_input')}</label>
                    <Field
                        name="v4.subnet_mask"
                        component={renderInputField}
                        type="text"
                        className="form-control"
                        placeholder={t(ipv4placeholders.subnet_mask)}
                        validate={[validateIpv4, validateRequired]}
                        disabled={!isInterfaceIncludesIpv4}
                    />
                </div>
            </div>
            <div className="col-lg-6">
                <div className="form__group form__group--settings">
                    <div className="row">
                        <div className="col-12">
                            <label>{t('dhcp_form_range_title')}</label>
                        </div>
                        <div className="col">
                            <Field
                                name="v4.range_start"
                                component={renderInputField}
                                type="text"
                                className="form-control"
                                placeholder={t(ipv4placeholders.range_start)}
                                validate={[validateIpv4]}
                                disabled={!isInterfaceIncludesIpv4}
                            />
                        </div>
                        <div className="col">
                            <Field
                                name="v4.range_end"
                                component={renderInputField}
                                type="text"
                                className="form-control"
                                placeholder={t(ipv4placeholders.range_end)}
                                validate={[validateIpv4, validateIpv4RangeEnd]}
                                disabled={!isInterfaceIncludesIpv4}
                            />
                        </div>
                    </div>
                </div>
                <div className="form__group form__group--settings">
                    <label>{t('dhcp_form_lease_title')}</label>
                    <Field
                        name="v4.lease_duration"
                        component={renderInputField}
                        type="number"
                        className="form-control"
                        placeholder={t(ipv4placeholders.lease_duration)}
                        validate={[validateIsPositiveValue, validateRequired]}
                        normalize={toNumber}
                        min={0}
                        disabled={!isInterfaceIncludesIpv4}
                    />
                </div>
            </div>
        </div>
        <div className="btn-list">
            <button
                type="submit"
                className="btn btn-success btn-standard"
                disabled={invalid}
            >
                {t('save_config')}
            </button>
        </div>
    </form>;
};

FormDHCPv4.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    initialValues: PropTypes.object.isRequired,
    processingConfig: PropTypes.bool.isRequired,
    change: PropTypes.func.isRequired,
    reset: PropTypes.func.isRequired,
    ipv4placeholders: PropTypes.object.isRequired,
};

export default reduxForm({
    form: FORM_NAME.DHCPv4,
})(FormDHCPv4);
