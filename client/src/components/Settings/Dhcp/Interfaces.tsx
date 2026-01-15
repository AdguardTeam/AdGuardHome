import React from 'react';
import { useSelector } from 'react-redux';
import { useFormContext } from 'react-hook-form';
import { Trans, useTranslation } from 'react-i18next';

import { validateRequiredValue } from '../../../helpers/validators';
import { RootState } from '../../../initialState';
import { DhcpFormValues } from '.';

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

interface RenderInterfaceValuesProps {
    gateway_ip: string;
    hardware_address: string;
    ip_addresses: string[];
}

const renderInterfaceValues = ({ gateway_ip, hardware_address, ip_addresses }: RenderInterfaceValuesProps) => (
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
    const {
        register,
        watch,
        formState: { errors },
    } = useFormContext<DhcpFormValues>();

    const { processingInterfaces, interfaces, enabled } = useSelector((store: RootState) => store.dhcp);

    const interface_name = watch('interface_name');

    if (processingInterfaces || !interfaces) {
        return null;
    }

    const interfaceValue = interface_name && interfaces[interface_name];

    return (
        <div className="row dhcp__interfaces">
            <div className="col col__dhcp">
                <label htmlFor="interface_name" className="form__label">
                    {t('dhcp_interface_select')}
                </label>
                <select
                    id="interface_name"
                    data-testid="interface_name"
                    className="form-control custom-select pl-4 col-md"
                    disabled={enabled}
                    {...register('interface_name', {
                        validate: validateRequiredValue,
                    })}>
                    <option value="" disabled={enabled}>
                        {t('dhcp_interface_select')}
                    </option>
                    {renderInterfaces(interfaces)}
                </select>
                {errors.interface_name && (
                    <div className="form__message form__message--error">{t(errors.interface_name.message)}</div>
                )}
            </div>
            {interfaceValue &&
                renderInterfaceValues({
                    gateway_ip: interfaceValue.gateway_ip,
                    hardware_address: interfaceValue.hardware_address,
                    ip_addresses: interfaceValue.ip_addresses,
                })}
        </div>
    );
};

export default Interfaces;
