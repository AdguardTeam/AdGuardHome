import React, { useCallback } from 'react';
import { shallowEqual, useSelector } from 'react-redux';

import { Field, reduxForm } from 'redux-form';
import { useTranslation } from 'react-i18next';

import { renderInputField, toNumber } from '../../../helpers/form';
import { FORM_NAME, UINT32_RANGE } from '../../../helpers/constants';
import {
    validateIpv4,
    validateRequiredValue,
    validateIpv4RangeEnd,
    validateGatewaySubnetMask,
    validateIpForGatewaySubnetMask,
    validateNotInRange,
} from '../../../helpers/validators';
import { RootState } from '../../../initialState';

interface FormDHCPv4Props {
    handleSubmit: (...args: unknown[]) => string;
    submitting: boolean;
    initialValues: { v4?: any };
    processingConfig?: boolean;
    change: (field: string, value: any) => void;
    reset: () => void;
    ipv4placeholders?: {
        gateway_ip: string;
        subnet_mask: string;
        range_start: string;
        range_end: string;
        lease_duration: string;
    };
}

const FormDHCPv4 = ({ handleSubmit, submitting, processingConfig, ipv4placeholders }: FormDHCPv4Props) => {
    const { t } = useTranslation();

    const dhcp = useSelector((state: RootState) => state.form[FORM_NAME.DHCPv4], shallowEqual);

    const interfaces = useSelector((state: RootState) => state.form[FORM_NAME.DHCP_INTERFACES], shallowEqual);
    const interface_name = interfaces?.values?.interface_name;

    const isInterfaceIncludesIpv4 = useSelector(
        (state: RootState) => !!state.dhcp?.interfaces?.[interface_name]?.ipv4_addresses,
    );

    const isEmptyConfig = !Object.values(dhcp?.values?.v4 ?? {}).some(Boolean);

    const invalid =
        dhcp?.syncErrors ||
        interfaces?.syncErrors ||
        !isInterfaceIncludesIpv4 ||
        isEmptyConfig ||
        submitting ||
        processingConfig;

    const validateRequired = useCallback(
        (value) => {
            if (isEmptyConfig) {
                return undefined;
            }
            return validateRequiredValue(value);
        },
        [isEmptyConfig],
    );

    return (
        <form onSubmit={handleSubmit}>
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
                            validate={[validateIpv4, validateRequired, validateNotInRange]}
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
                            validate={[validateRequired, validateGatewaySubnetMask]}
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
                                    validate={[validateIpv4, validateIpForGatewaySubnetMask]}
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
                                    validate={[validateIpv4, validateIpv4RangeEnd, validateIpForGatewaySubnetMask]}
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
                            validate={validateRequired}
                            normalize={toNumber}
                            min={1}
                            max={UINT32_RANGE.MAX}
                            disabled={!isInterfaceIncludesIpv4}
                        />
                    </div>
                </div>
            </div>

            <div className="btn-list">
                <button type="submit" className="btn btn-success btn-standard" disabled={invalid}>
                    {t('save_config')}
                </button>
            </div>
        </form>
    );
};

export default reduxForm<
    Record<string, any>,
    Omit<FormDHCPv4Props, 'submitting' | 'handleSubmit' | 'reset' | 'change'>
>({
    form: FORM_NAME.DHCPv4,
})(FormDHCPv4);
