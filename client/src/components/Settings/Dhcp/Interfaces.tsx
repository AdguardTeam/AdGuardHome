import React from 'react';
import { shallowEqual, useSelector } from 'react-redux';

import { Field, reduxForm } from 'redux-form';
import { Trans, useTranslation } from 'react-i18next';

import { renderSelectField } from '../../../helpers/form';
import { validateRequiredValue } from '../../../helpers/validators';
import { FORM_NAME } from '../../../helpers/constants';
import { RootState } from '../../../initialState';

const renderInterfaces = (interfaces: any) =>
    Object.keys(interfaces).map((item) => {
        const option = interfaces[item];
        const { name } = option;

        const [interfaceIPv4] = option?.ipv4_addresses ?? [];
        const [interfaceIPv6] = option?.ipv6_addresses ?? [];

        const optionContent = [name, interfaceIPv4, interfaceIPv6].filter(Boolean).join(' - ');

        return (
            <option value={name} key={name}>
                {optionContent}
            </option>
        );
    });

const getInterfaceValues = ({ gateway_ip, hardware_address, ip_addresses }: any) => [
    {
        name: 'dhcp_form_gateway_input',
        value: gateway_ip,
    },
    {
        name: 'dhcp_hardware_address',
        value: hardware_address,
    },
    {
        name: 'dhcp_ip_addresses',
        value: ip_addresses,
        render: (ip_addresses: any) =>
            ip_addresses.map((ip: any) => (
                <span key={ip} className="interface__ip">
                    {ip}
                </span>
            )),
    },
];

interface renderInterfaceValuesProps {
    gateway_ip: string;
    hardware_address: string;
    ip_addresses: string[];
}

const renderInterfaceValues = ({ gateway_ip, hardware_address, ip_addresses }: renderInterfaceValuesProps) => (
    <div className="d-flex align-items-end dhcp__interfaces-info">
        <ul className="list-unstyled m-0">
            {getInterfaceValues({
                gateway_ip,
                hardware_address,
                ip_addresses,
            }).map(
                ({ name, value, render }) =>
                    value && (
                        <li key={name}>
                            <span className="interface__title">
                                <Trans>{name}</Trans>:{' '}
                            </span>
                            {render?.(value) || value}
                        </li>
                    ),
            )}
        </ul>
    </div>
);

const Interfaces = () => {
    const { t } = useTranslation();

    const { processingInterfaces, interfaces, enabled } = useSelector((store: RootState) => store.dhcp, shallowEqual);

    const interface_name =
        useSelector((store: RootState) => store.form[FORM_NAME.DHCP_INTERFACES]?.values?.interface_name);

    if (processingInterfaces || !interfaces) {
        return null;
    }

    const interfaceValue = interface_name && interfaces[interface_name];

    return (
        <div className="row dhcp__interfaces">
            <div className="col col__dhcp">
                <Field
                    name="interface_name"
                    component={renderSelectField}
                    className="form-control custom-select pl-4 col-md"
                    validate={[validateRequiredValue]}
                    label="dhcp_interface_select">
                    <option value="" disabled={enabled}>
                        {t('dhcp_interface_select')}
                    </option>
                    {renderInterfaces(interfaces)}
                </Field>
            </div>
            {interfaceValue && renderInterfaceValues({
                gateway_ip: interfaceValue.gateway_ip,
                hardware_address: interfaceValue.hardware_address,
                ip_addresses: interfaceValue.ip_addresses
            })}
        </div>
    );
};

export default reduxForm({
    form: FORM_NAME.DHCP_INTERFACES,
})(Interfaces);
