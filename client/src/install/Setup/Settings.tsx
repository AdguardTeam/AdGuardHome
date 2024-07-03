import React, { Component } from 'react';
import { connect } from 'react-redux';

import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';
import i18n, { TFunction } from 'i18next';

import Controls from './Controls';

import AddressList from './AddressList';

import { getInterfaceIp } from '../../helpers/helpers';
import {
    ALL_INTERFACES_IP,
    FORM_NAME,
    ADDRESS_IN_USE_TEXT,
    PORT_53_FAQ_LINK,
    STATUS_RESPONSE,
    STANDARD_DNS_PORT,
    STANDARD_WEB_PORT,
} from '../../helpers/constants';

import { renderInputField, toNumber } from '../../helpers/form';
import { validateRequiredValue, validateInstallPort } from '../../helpers/validators';
import { DhcpInterface } from '../../initialState';

const renderInterfaces = (interfaces: DhcpInterface[]) =>
    Object.values(interfaces).map((option: DhcpInterface) => {
        const { name, ip_addresses, flags } = option;

        if (option && ip_addresses?.length > 0) {
            const ip = getInterfaceIp(option);
            const isUp = flags?.includes('up');

            return (
                <option value={ip} key={name} disabled={!isUp}>
                    {name} - {ip} {!isUp && `(${i18n.t('down')})`}
                </option>
            );
        }

        return null;
    });

type Props = {
    handleSubmit: (...args: unknown[]) => string;
    handleChange?: (...args: unknown[]) => unknown;
    handleFix: (...args: unknown[]) => unknown;
    validateForm?: (...args: unknown[]) => unknown;
    webIp: string;
    dnsIp: string;
    config: {
        web: {
            status: string;
            can_autofix: boolean;
        };
        dns: {
            status: string;
            can_autofix: boolean;
        };
        staticIp: {
            ip: string;
            static: string;
        };
    };
    webPort?: number;
    dnsPort?: number;
    interfaces: DhcpInterface[];
    invalid: boolean;
    initialValues?: object;
    t: TFunction;
};

class Settings extends Component<Props> {
    componentDidMount() {
        const { webIp, webPort, dnsIp, dnsPort } = this.props;

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

    getStaticIpMessage = (staticIp: { ip: string; static: string }) => {
        const { static: status, ip } = staticIp;

        switch (status) {
            case STATUS_RESPONSE.NO: {
                return (
                    <>
                        <div className="mb-2">
                            <Trans values={{ ip }} components={[<strong key="0">text</strong>]}>
                                install_static_configure
                            </Trans>
                        </div>

                        <button
                            type="button"
                            className="btn btn-outline-primary btn-sm"
                            onClick={() => this.handleStaticIp(ip)}>
                            <Trans>set_static_ip</Trans>
                        </button>
                    </>
                );
            }
            case STATUS_RESPONSE.ERROR: {
                return (
                    <div className="text-danger">
                        <Trans>install_static_error</Trans>
                    </div>
                );
            }
            case STATUS_RESPONSE.YES: {
                return (
                    <div className="text-success">
                        <Trans>install_static_ok</Trans>
                    </div>
                );
            }
            default:
                return null;
        }
    };

    handleAutofix = (type: any) => {
        const {
            webIp,

            webPort,

            dnsIp,

            dnsPort,

            handleFix,
        } = this.props;

        const web = {
            ip: webIp,
            port: webPort,
            autofix: false,
        };
        const dns = {
            ip: dnsIp,
            port: dnsPort,
            autofix: false,
        };
        const set_static_ip = false;

        if (type === 'web') {
            web.autofix = true;
        } else {
            dns.autofix = true;
        }

        handleFix(web, dns, set_static_ip);
    };

    handleStaticIp = (ip: any) => {
        const {
            webIp,

            webPort,

            dnsIp,

            dnsPort,

            handleFix,
        } = this.props;

        const web = {
            ip: webIp,
            port: webPort,
            autofix: false,
        };
        const dns = {
            ip: dnsIp,
            port: dnsPort,
            autofix: false,
        };
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

            t,
        } = this.props;
        const { status: webStatus, can_autofix: isWebFixAvailable } = config.web;
        const { status: dnsStatus, can_autofix: isDnsFixAvailable } = config.dns;
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
                                    onChange={handleChange}>
                                    <option value={ALL_INTERFACES_IP}>
                                        {this.props.t('install_settings_all_interfaces')}
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
                                    placeholder={STANDARD_WEB_PORT.toString()}
                                    validate={[validateInstallPort, validateRequiredValue]}
                                    normalize={toNumber}
                                    onChange={handleChange}
                                />
                            </div>
                        </div>

                        <div className="col-12">
                            {webStatus && (
                                <div className="setup__error text-danger">
                                    {webStatus}
                                    {isWebFixAvailable && (
                                        <button
                                            type="button"
                                            className="btn btn-secondary btn-sm ml-2"
                                            onClick={() => this.handleAutofix('web')}>
                                            <Trans>fix</Trans>
                                        </button>
                                    )}
                                </div>
                            )}

                            <hr className="divider--small" />
                        </div>
                    </div>

                    <div className="setup__desc">
                        <Trans>install_settings_interface_link</Trans>

                        <div className="mt-1">
                            <AddressList interfaces={interfaces} address={webIp} port={webPort} />
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
                                    onChange={handleChange}>
                                    <option value={ALL_INTERFACES_IP}>{t('install_settings_all_interfaces')}</option>
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
                                    placeholder={STANDARD_WEB_PORT.toString()}
                                    validate={[validateInstallPort, validateRequiredValue]}
                                    normalize={toNumber}
                                    onChange={handleChange}
                                />
                            </div>
                        </div>

                        <div className="col-12">
                            {dnsStatus && (
                                <>
                                    <div className="setup__error text-danger">
                                        {dnsStatus}
                                        {isDnsFixAvailable && (
                                            <button
                                                type="button"
                                                className="btn btn-secondary btn-sm ml-2"
                                                onClick={() => this.handleAutofix('dns')}>
                                                <Trans>fix</Trans>
                                            </button>
                                        )}
                                    </div>
                                    {isDnsFixAvailable && (
                                        <div className="text-muted mb-2">
                                            <p className="mb-1">
                                                <Trans>autofix_warning_text</Trans>
                                            </p>

                                            <Trans components={[<li key="0">text</li>]}>autofix_warning_list</Trans>

                                            <p className="mb-1">
                                                <Trans>autofix_warning_result</Trans>
                                            </p>
                                        </div>
                                    )}
                                </>
                            )}
                            {dnsPort === STANDARD_DNS_PORT &&
                                !isDnsFixAvailable &&
                                dnsStatus.includes(ADDRESS_IN_USE_TEXT) && (
                                    <Trans
                                        components={[
                                            <a
                                                href={PORT_53_FAQ_LINK}
                                                key="0"
                                                target="_blank"
                                                rel="noopener noreferrer">
                                                link
                                            </a>,
                                        ]}>
                                        port_53_faq_link
                                    </Trans>
                                )}

                            <hr className="divider--small" />
                        </div>
                    </div>

                    <div className="setup__desc">
                        <Trans>install_settings_dns_desc</Trans>

                        <div className="mt-1">
                            <AddressList interfaces={interfaces} address={dnsIp} port={dnsPort} isDns={true} />
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

const selector = formValueSelector(FORM_NAME.INSTALL);

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
    withTranslation(),
    reduxForm({
        form: FORM_NAME.INSTALL,
        destroyOnUnmount: false,
        forceUnregisterOnUnmount: true,
    }),
])(SettingsForm);
