import React, { Fragment } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';
import format from 'date-fns/format';

import { renderField, renderSelectField, toNumber, port, isSafePort } from '../../../helpers/form';
import { EMPTY_DATE } from '../../../helpers/constants';
import i18n from '../../../i18n';

const validate = (values) => {
    const errors = {};

    if (values.port_dns_over_tls && values.port_https) {
        if (values.port_dns_over_tls === values.port_https) {
            errors.port_dns_over_tls = i18n.t('form_error_equal');
            errors.port_https = i18n.t('form_error_equal');
        }
    }

    return errors;
};

const clearFields = (change, setTlsConfig, t) => {
    const fields = {
        private_key: '',
        certificate_chain: '',
        port_https: 443,
        port_dns_over_tls: 853,
        server_name: '',
        force_https: false,
        enabled: false,
    };
    // eslint-disable-next-line no-alert
    if (window.confirm(t('encryption_reset'))) {
        Object.keys(fields).forEach(field => change(field, fields[field]));
        setTlsConfig(fields);
    }
};

let Form = (props) => {
    const {
        t,
        handleSubmit,
        handleChange,
        isEnabled,
        certificateChain,
        privateKey,
        change,
        invalid,
        submitting,
        processingConfig,
        processingValidate,
        not_after,
        valid_chain,
        valid_key,
        valid_cert,
        valid_pair,
        dns_names,
        key_type,
        issuer,
        subject,
        warning_validation,
        setTlsConfig,
    } = props;

    const isSavingDisabled = invalid
        || submitting
        || processingConfig
        || processingValidate
        || (isEnabled && (!privateKey || !certificateChain))
        || (privateKey && !valid_key)
        || (certificateChain && !valid_cert)
        || (privateKey && certificateChain && !valid_pair);

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <Field
                            name="enabled"
                            type="checkbox"
                            component={renderSelectField}
                            placeholder={t('encryption_enable')}
                            onChange={handleChange}
                        />
                    </div>
                    <div className="form__desc">
                        <Trans>encryption_enable_desc</Trans>
                    </div>
                    <hr/>
                </div>
                <div className="col-12">
                    <label className="form__label" htmlFor="server_name">
                        <Trans>encryption_server</Trans>
                    </label>
                </div>
                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <Field
                            id="server_name"
                            name="server_name"
                            component={renderField}
                            type="text"
                            className="form-control"
                            placeholder={t('encryption_server_enter')}
                            onChange={handleChange}
                            disabled={!isEnabled}
                        />
                        <div className="form__desc">
                            <Trans>encryption_server_desc</Trans>
                        </div>
                    </div>
                </div>
                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <Field
                            name="force_https"
                            type="checkbox"
                            component={renderSelectField}
                            placeholder={t('encryption_redirect')}
                            onChange={handleChange}
                            disabled={!isEnabled}
                        />
                        <div className="form__desc">
                            <Trans>encryption_redirect_desc</Trans>
                        </div>
                    </div>
                </div>
            </div>
            <div className="row">
                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <label className="form__label" htmlFor="port_https">
                            <Trans>encryption_https</Trans>
                        </label>
                        <Field
                            id="port_https"
                            name="port_https"
                            component={renderField}
                            type="number"
                            className="form-control"
                            placeholder={t('encryption_https')}
                            validate={[port, isSafePort]}
                            normalize={toNumber}
                            onChange={handleChange}
                            disabled={!isEnabled}
                        />
                        <div className="form__desc">
                            <Trans>encryption_https_desc</Trans>
                        </div>
                    </div>
                </div>
                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <label className="form__label" htmlFor="port_dns_over_tls">
                            <Trans>encryption_dot</Trans>
                        </label>
                        <Field
                            id="port_dns_over_tls"
                            name="port_dns_over_tls"
                            component={renderField}
                            type="number"
                            className="form-control"
                            placeholder={t('encryption_dot')}
                            validate={[port]}
                            normalize={toNumber}
                            onChange={handleChange}
                            disabled={!isEnabled}
                        />
                        <div className="form__desc">
                            <Trans>encryption_dot_desc</Trans>
                        </div>
                    </div>
                </div>
            </div>
            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <label className="form__label form__label--bold" htmlFor="certificate_chain">
                            <Trans>encryption_certificates</Trans>
                        </label>
                        <div className="form__desc form__desc--top">
                            <Trans
                                values={{ link: 'letsencrypt.org' }}
                                components={[<a href="https://letsencrypt.org/" key="0">link</a>]}
                            >
                                encryption_certificates_desc
                            </Trans>
                        </div>
                        <Field
                            id="certificate_chain"
                            name="certificate_chain"
                            component="textarea"
                            type="text"
                            className="form-control form-control--textarea"
                            placeholder={t('encryption_certificates_input')}
                            onChange={handleChange}
                            disabled={!isEnabled}
                        />
                        <div className="form__status">
                            {certificateChain &&
                                <Fragment>
                                    <div className="form__label form__label--bold">
                                        <Trans>encryption_status</Trans>:
                                    </div>
                                    <ul className="encryption__list">
                                        <li className={valid_chain ? 'text-success' : 'text-danger'}>
                                            {valid_chain ?
                                                <Trans>encryption_chain_valid</Trans>
                                                : <Trans>encryption_chain_invalid</Trans>
                                            }
                                        </li>
                                        {valid_cert &&
                                            <Fragment>
                                                {subject &&
                                                    <li>
                                                        <Trans>encryption_subject</Trans>:&nbsp;
                                                        {subject}
                                                    </li>
                                                }
                                                {issuer &&
                                                    <li>
                                                        <Trans>encryption_issuer</Trans>:&nbsp;
                                                        {issuer}
                                                    </li>
                                                }
                                                {not_after && not_after !== EMPTY_DATE &&
                                                    <li>
                                                        <Trans>encryption_expire</Trans>:&nbsp;
                                                        {format(not_after, 'YYYY-MM-DD HH:mm:ss')}
                                                    </li>
                                                }
                                                {dns_names &&
                                                    <li>
                                                        <Trans>encryption_hostnames</Trans>:&nbsp;
                                                        {dns_names}
                                                    </li>
                                                }
                                            </Fragment>
                                        }
                                    </ul>
                                </Fragment>
                            }
                        </div>
                    </div>
                </div>
            </div>
            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <label className="form__label form__label--bold" htmlFor="private_key">
                            <Trans>encryption_key</Trans>
                        </label>
                        <Field
                            id="private_key"
                            name="private_key"
                            component="textarea"
                            type="text"
                            className="form-control form-control--textarea"
                            placeholder={t('encryption_key_input')}
                            onChange={handleChange}
                            disabled={!isEnabled}
                        />
                        <div className="form__status">
                            {privateKey &&
                                <Fragment>
                                    <div className="form__label form__label--bold">
                                        <Trans>encryption_status</Trans>:
                                    </div>
                                    <ul className="encryption__list">
                                        <li className={valid_key ? 'text-success' : 'text-danger'}>
                                            {valid_key ?
                                                <Trans values={{ type: key_type }}>
                                                    encryption_key_valid
                                                </Trans>
                                                : <Trans values={{ type: key_type }}>
                                                    encryption_key_invalid
                                                </Trans>
                                            }
                                        </li>
                                    </ul>
                                </Fragment>
                            }
                        </div>
                    </div>
                </div>
                {warning_validation &&
                    <div className="col-12">
                        <p className="text-danger">
                            {warning_validation}
                        </p>
                    </div>
                }
            </div>

            <div className="btn-list mt-2">
                <button
                    type="submit"
                    className="btn btn-success btn-standart"
                    disabled={isSavingDisabled}
                >
                    <Trans>save_config</Trans>
                </button>
                <button
                    type="button"
                    className="btn btn-secondary btn-standart"
                    disabled={submitting || processingConfig}
                    onClick={() => clearFields(change, setTlsConfig, t)}
                >
                    <Trans>reset_settings</Trans>
                </button>
            </div>
        </form>
    );
};

Form.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    handleChange: PropTypes.func,
    isEnabled: PropTypes.bool.isRequired,
    certificateChain: PropTypes.string.isRequired,
    privateKey: PropTypes.string.isRequired,
    change: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
    initialValues: PropTypes.object.isRequired,
    processingConfig: PropTypes.bool.isRequired,
    processingValidate: PropTypes.bool.isRequired,
    status_key: PropTypes.string,
    not_after: PropTypes.string,
    warning_validation: PropTypes.string,
    valid_chain: PropTypes.bool,
    valid_key: PropTypes.bool,
    valid_cert: PropTypes.bool,
    valid_pair: PropTypes.bool,
    dns_names: PropTypes.string,
    key_type: PropTypes.string,
    issuer: PropTypes.string,
    subject: PropTypes.string,
    t: PropTypes.func.isRequired,
    setTlsConfig: PropTypes.func.isRequired,
};

const selector = formValueSelector('encryptionForm');

Form = connect((state) => {
    const isEnabled = selector(state, 'enabled');
    const certificateChain = selector(state, 'certificate_chain');
    const privateKey = selector(state, 'private_key');
    return {
        isEnabled,
        certificateChain,
        privateKey,
    };
})(Form);

export default flow([
    withNamespaces(),
    reduxForm({
        form: 'encryptionForm',
        validate,
    }),
])(Form);
