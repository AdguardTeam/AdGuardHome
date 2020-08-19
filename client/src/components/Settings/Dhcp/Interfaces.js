import React from 'react';
import { shallowEqual, useSelector } from 'react-redux';
import { Field, reduxForm } from 'redux-form';
import { Trans, useTranslation } from 'react-i18next';
import propTypes from 'prop-types';
import { renderSelectField } from '../../../helpers/form';
import { validateRequiredValue } from '../../../helpers/validators';
import { FORM_NAME } from '../../../helpers/constants';

const renderInterfaces = (interfaces) => Object.keys(interfaces)
    .map((item) => {
        const option = interfaces[item];
        const { name } = option;

        const [interfaceIPv4] = option?.ipv4_addresses ?? [];
        const [interfaceIPv6] = option?.ipv6_addresses ?? [];

        const optionContent = [name, interfaceIPv4, interfaceIPv6].filter(Boolean).join(' - ');

        return <option value={name} key={name}>{optionContent}</option>;
    });


const getInterfaceValues = ({
    gateway_ip,
    hardware_address,
    ip_addresses,
}) => [
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
        render: (ip_addresses) => ip_addresses
            .map((ip) => <span key={ip} className="interface__ip">{ip}</span>),
    },
];

const renderInterfaceValues = ({
    gateway_ip,
    hardware_address,
    ip_addresses,
}) => <div className='d-flex align-items-end col-6'>
    <ul className="list-unstyled m-0">
        {getInterfaceValues({
            gateway_ip,
            hardware_address,
            ip_addresses,
        }).map(({ name, value, render }) => value && <li key={name}>
            <span className="interface__title"><Trans>{name}</Trans>: </span>
            {render?.(value) || value}
        </li>)}
    </ul>
</div>;

const Interfaces = () => {
    const { t } = useTranslation();

    const {
        processingInterfaces,
        interfaces,
        enabled,
    } = useSelector((store) => store.dhcp, shallowEqual);

    const interface_name = useSelector(
        (store) => store.form[FORM_NAME.DHCP_INTERFACES]?.values?.interface_name,
    );

    const interfaceValue = interface_name && interfaces[interface_name];

    return !processingInterfaces
            && interfaces
            && <>
                <div className="row align-items-center pb-2">
                    <div className="col-6">
                        <Field
                                name="interface_name"
                                component={renderSelectField}
                                className="form-control custom-select"
                                validate={[validateRequiredValue]}
                                label='dhcp_interface_select'
                        >
                            <option value='' disabled={enabled}>
                                {t('dhcp_interface_select')}
                            </option>
                            {renderInterfaces(interfaces)}
                        </Field>
                    </div>
                    {interfaceValue
                    && renderInterfaceValues(interfaceValue)}
                </div>
            </>;
};

renderInterfaceValues.propTypes = {
    gateway_ip: propTypes.string.isRequired,
    hardware_address: propTypes.string.isRequired,
    ip_addresses: propTypes.arrayOf(propTypes.string).isRequired,
};

export default reduxForm({
    form: FORM_NAME.DHCP_INTERFACES,
})(Interfaces);
