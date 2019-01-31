import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import Controls from './Controls';
import renderField from './renderField';
import { R_IPV4 } from '../../helpers/constants';

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

const port = (value) => {
    if (value < 1 || value > 65535) {
        return <Trans>form_error_port</Trans>;
    }
    return false;
};

const toNumber = value => value && parseInt(value, 10);

let Settings = (props) => {
    const {
        handleSubmit,
        interfaceIp,
        interfacePort,
        dnsIp,
        dnsPort,
        invalid,
    } = props;
    const dnsAddress = dnsPort && dnsPort !== 53 ? `${dnsIp}:${dnsPort}` : dnsIp;
    const interfaceAddress = interfacePort ? `http://${interfaceIp}:${interfacePort}` : `http://${interfaceIp}`;

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
                                component={renderField}
                                type="text"
                                className="form-control"
                                placeholder="0.0.0.0"
                                validate={[ipv4, required]}
                            />
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
                            />
                        </div>
                    </div>
                </div>
                <div className="setup__desc">
                    <Trans
                        components={[<a href={`http://${interfaceIp}`} key="0">link</a>]}
                        values={{ link: interfaceAddress }}
                    >
                        install_settings_interface_link
                    </Trans>
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
                                component={renderField}
                                type="text"
                                className="form-control"
                                placeholder="0.0.0.0"
                                validate={[ipv4, required]}
                            />
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
                            />
                        </div>
                    </div>
                </div>
                <p className="setup__desc">
                    <Trans
                        components={[<strong key="0">ip</strong>]}
                        values={{ ip: dnsAddress }}
                    >
                        install_settings_dns_desc
                    </Trans>
                </p>
            </div>
            <Controls invalid={invalid} />
        </form>
    );
};

Settings.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    interfaceIp: PropTypes.string.isRequired,
    dnsIp: PropTypes.string.isRequired,
    interfacePort: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.number,
    ]),
    dnsPort: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.number,
    ]),
    invalid: PropTypes.bool.isRequired,
    initialValues: PropTypes.object,
};

Settings.defaultProps = {
    interfaceIp: '192.168.0.1',
    interfacePort: 3000,
    dnsIp: '192.168.0.1',
    dnsPort: 53,
};

const selector = formValueSelector('install');

Settings = connect((state) => {
    const interfaceIp = selector(state, 'web.ip');
    const interfacePort = selector(state, 'web.port');
    const dnsIp = selector(state, 'dns.ip');
    const dnsPort = selector(state, 'dns.port');

    return {
        interfaceIp,
        interfacePort,
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
])(Settings);
