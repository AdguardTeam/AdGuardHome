import React, { Component, Fragment } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import Controls from './Controls';
import AddressList from './AddressList';

import { getInterfaceIp } from '../../helpers/helpers';
import { ALL_INTERFACES_IP } from '../../helpers/constants';
import { renderInputField, required, validInstallPort, toNumber } from '../../helpers/form';

const STATIC_STATUS = {
    ENABLED: 'yes',
    DISABLED: 'no',
    ERROR: 'error',
};

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
        const {
            webIp, webPort, dnsIp, dnsPort,
        } = this.props;

        this.props.validateForm({
            web: {
                ip: webIp,
                port: webPort,
            },
            dns: {
                ip: dnsIp,
                port: dnsPort,
            },
        });
    }

    getStaticIpMessage = (staticIp) => {
        const { static: status, ip } = staticIp;

        if (!status) {
            return '';
        }

        return (
            <Fragment>
                {status === STATIC_STATUS.DISABLED && (
                    <Fragment>
                        <div className="mb-2">
                            <Trans values={{ ip }} components={[<strong key="0">text</strong>]}>
                                install_static_configure
                            </Trans>
                        </div>
                        <button
                            type="button"
                            className="btn btn-outline-primary btn-sm"
                            onClick={() => this.handleStaticIp(ip)}
                        >
                            <Trans>set_static_ip</Trans>
                        </button>
                    </Fragment>
                )}
                {status === STATIC_STATUS.ERROR && (
                    <div className="text-danger">
                        <Trans>install_static_error</Trans>
                    </div>
                )}
                {status === STATIC_STATUS.ENABLED && (
                    <div className="text-success">
                        <Trans>
                            install_static_ok
                        </Trans>
                    </div>
                )}
            </Fragment>
        );
    };

    handleAutofix = (type) => {
        const {
            webIp,
            webPort,
            dnsIp,
            dnsPort,
            handleFix,
        } = this.props;

        const web = { ip: webIp, port: webPort, autofix: false };
        const dns = { ip: dnsIp, port: dnsPort, autofix: false };
        const set_static_ip = false;

        if (type === 'web') {
            web.autofix = true;
        } else {
            dns.autofix = true;
        }

        handleFix(web, dns, set_static_ip);
    };

    handleStaticIp = (ip) => {
        const {
            webIp,
            webPort,
            dnsIp,
            dnsPort,
            handleFix,
        } = this.props;

        const web = { ip: webIp, port: webPort, autofix: false };
        const dns = { ip: dnsIp, port: dnsPort, autofix: false };
        const set_static_ip = true;

        if (window.confirm(this.props.t('confirm_static_ip', { ip }))) {
            handleFix(web, dns, set_static_ip);
        }
    };

    render() {
        const {
            handleSubmit,
            handleChange,
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
        const { staticIp } = config;

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
                                    component={renderInputField}
                                    type="number"
                                    className="form-control"
                                    placeholder="80"
                                    validate={[validInstallPort, required]}
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
                                    onClick={() => this.handleAutofix('web')}
                                >
                                    <Trans>fix</Trans>
                                </button>
                                }
                                <hr className="divider--small" />
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
                                    component={renderInputField}
                                    type="number"
                                    className="form-control"
                                    placeholder="80"
                                    validate={[validInstallPort, required]}
                                    normalize={toNumber}
                                    onChange={handleChange}
                                />
                            </div>
                        </div>
                        <div className="col-12">
                            {dnsStatus &&
                            <Fragment>
                                <div className="setup__error text-danger">
                                    {dnsStatus}
                                    {isDnsFixAvailable &&
                                    <button
                                        type="button"
                                        className="btn btn-secondary btn-sm ml-2"
                                        onClick={() => this.handleAutofix('dns')}
                                    >
                                        <Trans>fix</Trans>
                                    </button>
                                    }
                                </div>
                                {isDnsFixAvailable &&
                                <div className="text-muted mb-2">
                                    <p className="mb-1">
                                        <Trans>autofix_warning_text</Trans>
                                    </p>
                                    <Trans components={[<li key="0">text</li>]}>
                                        autofix_warning_list
                                    </Trans>
                                    <p className="mb-1">
                                        <Trans>autofix_warning_result</Trans>
                                    </p>
                                </div>}
                                <hr className="divider--small" />
                            </Fragment>
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

                <div className="setup__group">
                    <div className="setup__subtitle">
                        <Trans>static_ip</Trans>
                    </div>

                    <div className="mb-2">
                        <Trans>static_ip_desc</Trans>
                    </div>

                    {this.getStaticIpMessage(staticIp)}
                </div>

                <Controls invalid={invalid} />
            </form>
        );
    }
}

Settings.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    handleChange: PropTypes.func,
    handleFix: PropTypes.func.isRequired,
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
    t: PropTypes.func.isRequired,
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
