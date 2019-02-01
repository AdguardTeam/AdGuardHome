import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import { renderField, renderSelectField, toNumber, port } from '../../../helpers/form';
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

const Form = (props) => {
    const {
        t,
        handleSubmit,
        reset,
        invalid,
        submitting,
        processing,
        statusCert,
        statusKey,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
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
                            validate={[port]}
                            normalize={toNumber}
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
                        />
                        <div className="form__status">
                            {statusCert &&
                                <Fragment>
                                    <div className="form__label form__label--bold">
                                        <Trans>encryption_status</Trans>:
                                    </div>
                                    <div>
                                        {statusCert}
                                    </div>
                                </Fragment>
                            }
                            {/* <div>
                                <Trans values={{ domains: '*.example.org, example.org' }}>
                                    encryption_certificates_for
                                </Trans>
                            </div>
                            <div>
                                <Trans values={{ date: '2022-01-01' }}>
                                    encryption_expire
                                </Trans>
                            </div> */}
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
                            placeholder="Copy/paste your PEM-encoded private key for your cerficate here."
                        />
                        <div className="form__status">
                            {statusKey &&
                                <Fragment>
                                    <div className="form__label form__label--bold">
                                        <Trans>encryption_status</Trans>:
                                    </div>
                                    <div>
                                        {statusKey}
                                    </div>
                                </Fragment>
                            }
                        </div>
                    </div>
                </div>
            </div>

            <div className="btn-list">
                <button
                    type="submit"
                    className="btn btn-success btn-standart"
                    disabled={invalid || submitting || processing}
                >
                    <Trans>save_config</Trans>
                </button>
                <button
                    type="submit"
                    className="btn btn-secondary btn-standart"
                    disabled={submitting || processing}
                    onClick={reset}
                >
                    <Trans>reset_settings</Trans>
                </button>
            </div>
        </form>
    );
};

Form.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    reset: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
    initialValues: PropTypes.object.isRequired,
    processing: PropTypes.bool.isRequired,
    statusCert: PropTypes.string,
    statusKey: PropTypes.string,
    t: PropTypes.func.isRequired,
};

export default flow([
    withNamespaces(),
    reduxForm({
        form: 'encryptionForm',
        validate,
    }),
])(Form);
