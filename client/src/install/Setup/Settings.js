import React, { Component, Fragment } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import Controls from './Controls';
import AddressList from './AddressList';
import renderField from './renderField';
import { getInterfaceIp } from '../../helpers/helpers';
import { ALL_INTERFACES_IP } from '../../helpers/constants';

const required = (value) => {
    if (value || value === 0) {
        return false;
    }
    return <Trans>form_error_required</Trans>;
};

const port = (value) => {
    if (value < 1 || value > 65535) {
        return <Trans>form_error_port</Trans>;
    }
    return false;
};

const toNumber = value => value && parseInt(value, 10);

const renderInterfaces = (interfaces => (
    Object.keys(interfaces).map((item) => {
        const option = interfaces[item];
        const {
            name,
            ip_addresses,
            flags,
        } = option;

        if (option && ip_addresses && ip_addresses.length > 0) {
            const ip = getInterfaceIp(option);
            const isDown = flags && flags.includes('down');

            if (isDown) {
                return (
                    <option value={ip} key={name} disabled>
                        <Fragment>
                            {name} - {ip} (<Trans>down</Trans>)
                        </Fragment>
                    </option>
                );
            }

            return (
                <option value={ip} key={name}>
                    {name} - {ip}
                </option>
            );
        }

        return false;
    })
));

class Settings extends Component {
    componentDidMount() {
        const { web, dns } = this.props.config;

        this.props.validateForm({
            web,
            dns,
        });
    }

    render() {
        const {
            handleSubmit,
            handleChange,
            handleAutofix,
            webIp,
            webPort,
            dnsIp,
            dnsPort,
            interfaces,
            invalid,
            config,
        } = this.props;
        const {
            status: webStatus,
            can_autofix: isWebFixAvailable,
        } = config.web;
        const {
            status: dnsStatus,
            can_autofix: isDnsFixAvailable,
        } = config.dns;

        return (
            <form className="setup__step" onSubmit={handleSubmit}>
                <div className="setup__group">
                    <div className="setup__subtitle">
                        <Trans>install_settings_title</Trans>
                    </div>
                    <div className="row">
                        <div className="col-8">
                            <div className="form-group">
                                <label>
                                    <Trans>install_settings_listen</Trans>
                                </label>
                                <Field
                                    name="web.ip"
                                    component="select"
                                    className="form-control custom-select"
                                    onChange={handleChange}
                                >
                                    <option value={ALL_INTERFACES_IP}>
                                        <Trans>install_settings_all_interfaces</Trans>
                                    </option>
                                    {renderInterfaces(interfaces)}
                                </Field>
                            </div>
                        </div>
                        <div className="col-4">
                            <div className="form-group">
                                <label>
                                    <Trans>install_settings_port</Trans>
                                </label>
                                <Field
                                    name="web.port"
                                    component={renderField}
                                    type="number"
                                    className="form-control"
                                    placeholder="80"
                                    validate={[port, required]}
                                    normalize={toNumber}
                                    onChange={handleChange}
                                />
                            </div>
                        </div>
                        <div className="col-12">
                            {webStatus &&
                                <div className="setup__error text-danger">
                                    {webStatus}
                                    {isWebFixAvailable &&
                                        <button
                                            type="button"
                                            className="btn btn-secondary btn-sm ml-2"
                                            onClick={() => handleAutofix('web', webIp, webPort)}
                                        >
                                            <Trans>fix</Trans>
                                        </button>
                                    }
                                </div>
                            }
                        </div>
                    </div>
                    <div className="setup__desc">
                        <Trans>install_settings_interface_link</Trans>
                        <div className="mt-1">
                            <AddressList
                                interfaces={interfaces}
                                address={webIp}
                                port={webPort}
                            />
                        </div>
                    </div>
                </div>
                <div className="setup__group">
                    <div className="setup__subtitle">
                        <Trans>install_settings_dns</Trans>
                    </div>
                    <div className="row">
                        <div className="col-8">
                            <div className="form-group">
                                <label>
                                    <Trans>install_settings_listen</Trans>
                                </label>
                                <Field
                                    name="dns.ip"
                                    component="select"
                                    className="form-control custom-select"
                                    onChange={handleChange}
                                >
                                    <option value={ALL_INTERFACES_IP}>
                                        <Trans>install_settings_all_interfaces</Trans>
                                    </option>
                                    {renderInterfaces(interfaces)}
                                </Field>
                            </div>
                        </div>
                        <div className="col-4">
                            <div className="form-group">
                                <label>
                                    <Trans>install_settings_port</Trans>
                                </label>
                                <Field
                                    name="dns.port"
                                    component={renderField}
                                    type="number"
                                    className="form-control"
                                    placeholder="80"
                                    validate={[port, required]}
                                    normalize={toNumber}
                                    onChange={handleChange}
                                />
                            </div>
                        </div>
                        <div className="col-12">
                            {dnsStatus &&
                                <div className="setup__error text-danger">
                                    {dnsStatus}
                                    {isDnsFixAvailable &&
                                        <button
                                            type="button"
                                            className="btn btn-secondary btn-sm ml-2"
                                            onClick={() => handleAutofix('dns', dnsIp, dnsPort)}
                                        >
                                            <Trans>fix</Trans>
                                        </button>
                                    }
                                </div>
                            }
                        </div>
                    </div>
                    <div className="setup__desc">
                        <Trans>install_settings_dns_desc</Trans>
                        <div className="mt-1">
                            <AddressList
                                interfaces={interfaces}
                                address={dnsIp}
                                port={dnsPort}
                                isDns={true}
                            />
                        </div>
                    </div>
                </div>
                <Controls invalid={invalid} />
            </form>
        );
    }
}

Settings.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    handleChange: PropTypes.func,
    handleAutofix: PropTypes.func,
    validateForm: PropTypes.func,
    webIp: PropTypes.string.isRequired,
    dnsIp: PropTypes.string.isRequired,
    config: PropTypes.object.isRequired,
    webPort: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.number,
    ]),
    dnsPort: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.number,
    ]),
    interfaces: PropTypes.object.isRequired,
    invalid: PropTypes.bool.isRequired,
    initialValues: PropTypes.object,
};

const selector = formValueSelector('install');

const SettingsForm = connect((state) => {
    const webIp = selector(state, 'web.ip');
    const webPort = selector(state, 'web.port');
    const dnsIp = selector(state, 'dns.ip');
    const dnsPort = selector(state, 'dns.port');

    return {
        webIp,
        webPort,
        dnsIp,
        dnsPort,
    };
})(Settings);

export default flow([
    withNamespaces(),
    reduxForm({
        form: 'install',
        destroyOnUnmount: false,
        forceUnregisterOnUnmount: true,
    }),
])(SettingsForm);
