import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { withNamespaces, Trans } from 'react-i18next';
import flow from 'lodash/flow';

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

let Interface = (props) => {
    const {
        t,
        handleChange,
        interfaces,
        processing,
        interfaceValue,
        enabled,
    } = props;

    return (
        <form>
            {!processing && interfaces &&
                <div className="row">
                    <div className="col-sm-12 col-md-6">
                        <div className="form__group form__group--settings">
                            <label>{t('dhcp_interface_select')}</label>
                            <Field
                                name="interface_name"
                                component="select"
                                className="form-control custom-select"
                                onChange={handleChange}
                            >
                                <option value="" disabled={enabled}>{t('dhcp_interface_select')}</option>
                                {renderInterfaces(interfaces)}
                            </Field>
                        </div>
                    </div>
                    {interfaceValue &&
                        <div className="col-sm-12 col-md-6">
                            {interfaces[interfaceValue] &&
                                renderInterfaceValues(interfaces[interfaceValue])}
                        </div>
                    }
                </div>
            }
            <hr/>
        </form>
    );
};

Interface.propTypes = {
    handleChange: PropTypes.func,
    interfaces: PropTypes.object,
    processing: PropTypes.bool,
    interfaceValue: PropTypes.string,
    initialValues: PropTypes.object,
    enabled: PropTypes.bool,
    t: PropTypes.func,
};

const selector = formValueSelector('dhcpInterface');

Interface = connect((state) => {
    const interfaceValue = selector(state, 'interface_name');
    return {
        interfaceValue,
    };
})(Interface);

export default flow([
    withNamespaces(),
    reduxForm({ form: 'dhcpInterface' }),
])(Interface);
