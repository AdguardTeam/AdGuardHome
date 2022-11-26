import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import {
    renderInputField,
    CheckboxField,
    renderRadioField,
    toNumber,
} from '../../../helpers/form';
import {
    validateServerName, validateIsSafePort, validatePort, validatePortQuic, validatePortTLS,
} from '../../../helpers/validators';
import i18n from '../../../i18n';
import KeyStatus from './KeyStatus';
import CertificateStatus from './CertificateStatus';
import {
    DNS_OVER_QUIC_PORT, DNS_OVER_TLS_PORT, FORM_NAME, STANDARD_HTTPS_PORT, ENCRYPTION_SOURCE,
} from '../../../helpers/constants';

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
        private_key_path: '',
        certificate_path: '',
        port_https: STANDARD_HTTPS_PORT,
        port_dns_over_tls: DNS_OVER_TLS_PORT,
        port_dns_over_quic: DNS_OVER_QUIC_PORT,
        server_name: '',
        force_https: false,
        enabled: false,
        private_key_saved: false,
    };
    // eslint-disable-next-line no-alert
    if (window.confirm(t('encryption_reset'))) {
        Object.keys(fields)
            .forEach((field) => change(field, fields[field]));
        setTlsConfig(fields);
    }
};

const validationMessage = (warningValidation, isWarning) => {
    if (!warningValidation) {
        return null;
    }

    if (isWarning) {
        return (
            <div className="col-12">
                <p><Trans>encryption_warning</Trans>: {warningValidation}</p>
            </div>
        );
    }

    return (
        <div className="col-12">
            <p className="text-danger">{warningValidation}</p>
        </div>
    );
};

let Form = (props) => {
    const {
        t,
        handleSubmit,
        handleChange,
        isEnabled,
        certificateChain,
        privateKey,
        certificatePath,
        privateKeyPath,
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
        certificateSource,
        privateKeySource,
        privateKeySaved,
    } = props;

    const isSavingDisabled = invalid
        || submitting
        || processingConfig
        || processingValidate
        || !valid_key
        || !valid_cert
        || !valid_pair;

    const isWarning = valid_key && valid_cert && valid_pair;

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <Field
                            name="enabled"
                            type="checkbox"
                            component={CheckboxField}
                            placeholder={t('encryption_enable')}
                            onChange={handleChange}
                        />
                    </div>
                    <div className="form__desc">
                        <Trans>encryption_enable_desc</Trans>
                    </div>
                    <hr />
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
                            component={renderInputField}
                            type="text"
                            className="form-control"
                            placeholder={t('encryption_server_enter')}
                            onChange={handleChange}
                            disabled={!isEnabled}
                            validate={validateServerName}
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
                            component={CheckboxField}
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
                            component={renderInputField}
                            type="number"
                            className="form-control"
                            placeholder={t('encryption_https')}
                            validate={[validatePort, validateIsSafePort]}
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
                            component={renderInputField}
                            type="number"
                            className="form-control"
                            placeholder={t('encryption_dot')}
                            validate={[validatePortTLS]}
                            normalize={toNumber}
                            onChange={handleChange}
                            disabled={!isEnabled}
                        />
                        <div className="form__desc">
                            <Trans>encryption_dot_desc</Trans>
                        </div>
                    </div>
                </div>
                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <label className="form__label" htmlFor="port_dns_over_quic">
                            <Trans>encryption_doq</Trans>
                        </label>
                        <Field
                                id="port_dns_over_quic"
                                name="port_dns_over_quic"
                                component={renderInputField}
                                type="number"
                                className="form-control"
                                placeholder={t('encryption_doq')}
                                validate={[validatePortQuic]}
                                normalize={toNumber}
                                onChange={handleChange}
                                disabled={!isEnabled}
                        />
                        <div className="form__desc">
                            <Trans>encryption_doq_desc</Trans>
                        </div>
                    </div>
                </div>
            </div>
            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <label
                            className="form__label form__label--with-desc form__label--bold"
                            htmlFor="certificate_chain"
                        >
                            <Trans>encryption_certificates</Trans>
                        </label>
                        <div className="form__desc form__desc--top">
                            <Trans
                                values={{ link: 'letsencrypt.org' }}
                                components={[
                                    <a target="_blank" rel="noopener noreferrer" href="https://letsencrypt.org/" key="0">
                                        link
                                    </a>,
                                ]}
                            >
                                encryption_certificates_desc
                            </Trans>
                        </div>

                        <div className="form__inline mb-2">
                            <div className="custom-controls-stacked">
                                <Field
                                    name="certificate_source"
                                    component={renderRadioField}
                                    type="radio"
                                    className="form-control mr-2"
                                    value="path"
                                    placeholder={t('encryption_certificates_source_path')}
                                    disabled={!isEnabled}
                                />
                                <Field
                                    name="certificate_source"
                                    component={renderRadioField}
                                    type="radio"
                                    className="form-control mr-2"
                                    value="content"
                                    placeholder={t('encryption_certificates_source_content')}
                                    disabled={!isEnabled}
                                />
                            </div>
                        </div>

                        {certificateSource === ENCRYPTION_SOURCE.CONTENT && (
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
                        )}
                        {certificateSource === ENCRYPTION_SOURCE.PATH && (
                            <Field
                                id="certificate_path"
                                name="certificate_path"
                                component={renderInputField}
                                type="text"
                                className="form-control"
                                placeholder={t('encryption_certificate_path')}
                                onChange={handleChange}
                                disabled={!isEnabled}
                            />
                        )}
                    </div>
                    <div className="form__status">
                        {(certificateChain || certificatePath) && (
                            <CertificateStatus
                                validChain={valid_chain}
                                validCert={valid_cert}
                                subject={subject}
                                issuer={issuer}
                                notAfter={not_after}
                                dnsNames={dns_names}
                            />
                        )}
                    </div>
                </div>
            </div>
            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings mt-3">
                        <label className="form__label form__label--bold" htmlFor="private_key">
                            <Trans>encryption_key</Trans>
                        </label>

                        <div className="form__inline mb-2">
                            <div className="custom-controls-stacked">
                                <Field
                                    name="key_source"
                                    component={renderRadioField}
                                    type="radio"
                                    className="form-control mr-2"
                                    value={ENCRYPTION_SOURCE.PATH}
                                    placeholder={t('encryption_key_source_path')}
                                    disabled={!isEnabled}
                                />
                                <Field
                                    name="key_source"
                                    component={renderRadioField}
                                    type="radio"
                                    className="form-control mr-2"
                                    value={ENCRYPTION_SOURCE.CONTENT}
                                    placeholder={t('encryption_key_source_content')}
                                    disabled={!isEnabled}
                                />
                            </div>
                        </div>

                        {privateKeySource === ENCRYPTION_SOURCE.PATH && (
                            <Field
                                name="private_key_path"
                                component={renderInputField}
                                type="text"
                                className="form-control"
                                placeholder={t('encryption_private_key_path')}
                                onChange={handleChange}
                                disabled={!isEnabled}
                            />
                        )}
                        {privateKeySource === ENCRYPTION_SOURCE.CONTENT && [
                            <Field
                                key="private_key_saved"
                                name="private_key_saved"
                                type="checkbox"
                                className="form__group form__group--settings mb-2"
                                component={CheckboxField}
                                disabled={!isEnabled}
                                placeholder={t('use_saved_key')}
                                onChange={(event) => {
                                    if (event.target.checked) {
                                        change('private_key', '');
                                    }
                                    if (handleChange) {
                                        handleChange(event);
                                    }
                                }}
                            />,
                            <Field
                                id="private_key"
                                key="private_key"
                                name="private_key"
                                component="textarea"
                                type="text"
                                className="form-control form-control--textarea"
                                placeholder={t('encryption_key_input')}
                                onChange={handleChange}
                                disabled={!isEnabled || privateKeySaved}
                            />,
                        ]}
                    </div>
                    <div className="form__status">
                        {(privateKey || privateKeyPath) && (
                            <KeyStatus validKey={valid_key} keyType={key_type} />
                        )}
                    </div>
                </div>
                {validationMessage(warning_validation, isWarning)}
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
    certificatePath: PropTypes.string.isRequired,
    privateKeyPath: PropTypes.string.isRequired,
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
    dns_names: PropTypes.arrayOf(PropTypes.string),
    key_type: PropTypes.string,
    issuer: PropTypes.string,
    subject: PropTypes.string,
    t: PropTypes.func.isRequired,
    setTlsConfig: PropTypes.func.isRequired,
    certificateSource: PropTypes.string,
    privateKeySource: PropTypes.string,
    privateKeySaved: PropTypes.bool,
};

const selector = formValueSelector(FORM_NAME.ENCRYPTION);

Form = connect((state) => {
    const isEnabled = selector(state, 'enabled');
    const certificateChain = selector(state, 'certificate_chain');
    const privateKey = selector(state, 'private_key');
    const certificatePath = selector(state, 'certificate_path');
    const privateKeyPath = selector(state, 'private_key_path');
    const certificateSource = selector(state, 'certificate_source');
    const privateKeySource = selector(state, 'key_source');
    const privateKeySaved = selector(state, 'private_key_saved');
    return {
        isEnabled,
        certificateChain,
        privateKey,
        certificatePath,
        privateKeyPath,
        certificateSource,
        privateKeySource,
        privateKeySaved,
    };
})(Form);

export default flow([
    withTranslation(),
    reduxForm({
        form: FORM_NAME.ENCRYPTION,
        validate,
    }),
])(Form);
