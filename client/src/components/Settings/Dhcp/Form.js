import React, { Fragment } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { withNamespaces, Trans } from 'react-i18next';
import flow from 'lodash/flow';

import { R_IPV4 } from '../../../helpers/constants';

const required = (value) => {
    if (value || value === 0) {
        return false;
    }
    return <Trans>form_error_required</Trans>;
};

const ipv4 = (value) => {
    if (value && !new RegExp(R_IPV4).test(value)) {
        return <Trans>form_error_ip_format</Trans>;
    }
    return false;
};

const isPositive = (value) => {
    if ((value || value === 0) && (value <= 0)) {
        return <Trans>form_error_positive</Trans>;
    }
    return false;
};

const toNumber = value => value && parseInt(value, 10);

const renderField = ({
    input, className, placeholder, type, disabled, meta: { touched, error },
}) => (
    <Fragment>
        <input
            {...input}
            placeholder={placeholder}
            type={type}
            className={className}
            disabled={disabled}
        />
        {!disabled && touched && (error && <span className="form__message form__message--error">{error}</span>)}
    </Fragment>
);

const renderInterfaces = (interfaces => (
    Object.keys(interfaces).map((item) => {
        const option = interfaces[item];
        const { name } = option;
        const onlyIPv6 = option.ip_addresses.every(ip => ip.includes(':'));
        let interfaceIP = option.ip_addresses[0];

        if (!onlyIPv6) {
            option.ip_addresses.forEach((ip) => {
                if (!ip.includes(':')) {
                    interfaceIP = ip;
                }
            });
        }

        return (
            <option value={name} key={name} disabled={onlyIPv6}>
                {name} - {interfaceIP}
            </option>
        );
    })
));

const renderInterfaceValues = (interfaceValues => (
    <ul className="list-unstyled mt-1 mb-0">
        <li>
            <span className="interface__title">MTU: </span>
            {interfaceValues.mtu}
        </li>
        <li>
            <span className="interface__title"><Trans>dhcp_hardware_address</Trans>: </span>
            {interfaceValues.hardware_address}
        </li>
        <li>
            <span className="interface__title"><Trans>dhcp_ip_addresses</Trans>: </span>
            {
                interfaceValues.ip_addresses
                    .map(ip => <span key={ip} className="interface__ip">{ip}</span>)
            }
        </li>
    </ul>
));

let Form = (props) => {
    const {
        t,
        handleSubmit,
        pristine,
        submitting,
        interfaces,
        processing,
        interfaceValue,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
                <div className="col-12">
                    {!processing && interfaces &&
                        <div className="row">
                            <div className="col-sm-12 col-md-6">
                                <div className="form__group form__group--dhcp">
                                    <label>{t('dhcp_interface_select')}</label>
                                    <Field
                                        name="interface_name"
                                        component="select"
                                        className="form-control custom-select"
                                    >
                                        <option value="">{t('dhcp_interface_select')}</option>
                                        {renderInterfaces(interfaces)}
                                    </Field>
                                </div>
                            </div>
                            {interfaceValue &&
                                <div className="col-sm-12 col-md-6">
                                    {renderInterfaceValues(interfaces[interfaceValue])}
                                </div>
                            }
                        </div>
                    }
                    <hr/>
                </div>
                <div className="col-lg-6">
                    <div className="form__group form__group--dhcp">
                        <label>{t('dhcp_form_gateway_input')}</label>
                        <Field
                            name="gateway_ip"
                            component={renderField}
                            type="text"
                            className="form-control"
                            placeholder={t('dhcp_form_gateway_input')}
                            validate={[ipv4, required]}
                        />
                    </div>
                    <div className="form__group form__group--dhcp">
                        <label>{t('dhcp_form_subnet_input')}</label>
                        <Field
                            name="subnet_mask"
                            component={renderField}
                            type="text"
                            className="form-control"
                            placeholder={t('dhcp_form_subnet_input')}
                            validate={[ipv4, required]}
                        />
                    </div>
                </div>
                <div className="col-lg-6">
                    <div className="form__group form__group--dhcp">
                        <div className="row">
                            <div className="col-12">
                                <label>{t('dhcp_form_range_title')}</label>
                            </div>
                            <div className="col">
                                <Field
                                    name="range_start"
                                    component={renderField}
                                    type="text"
                                    className="form-control"
                                    placeholder={t('dhcp_form_range_start')}
                                    validate={[ipv4, required]}
                                />
                            </div>
                            <div className="col">
                                <Field
                                    name="range_end"
                                    component={renderField}
                                    type="text"
                                    className="form-control"
                                    placeholder={t('dhcp_form_range_end')}
                                    validate={[ipv4, required]}
                                />
                            </div>
                        </div>
                    </div>
                    <div className="form__group form__group--dhcp">
                        <label>{t('dhcp_form_lease_title')}</label>
                        <Field
                            name="lease_duration"
                            component={renderField}
                            type="number"
                            className="form-control"
                            placeholder={t('dhcp_form_lease_input')}
                            validate={[required, isPositive]}
                            normalize={toNumber}
                        />
                    </div>
                </div>
            </div>

            <button
                type="submit"
                className="btn btn-success btn-standart"
                disabled={pristine || submitting}
            >
                {t('save_config')}
            </button>
        </form>
    );
};

Form.propTypes = {
    handleSubmit: PropTypes.func,
    pristine: PropTypes.bool,
    submitting: PropTypes.bool,
    interfaces: PropTypes.object,
    processing: PropTypes.bool,
    interfaceValue: PropTypes.string,
    initialValues: PropTypes.object,
    t: PropTypes.func,
};

const selector = formValueSelector('dhcpForm');

Form = connect((state) => {
    const interfaceValue = selector(state, 'interface_name');
    return {
        interfaceValue,
    };
})(Form);

export default flow([
    withNamespaces(),
    reduxForm({ form: 'dhcpForm' }),
])(Form);
