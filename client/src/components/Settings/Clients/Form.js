import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import { renderField, renderSelectField, ipv4, mac, required } from '../../../helpers/form';
import { CLIENT_ID } from '../../../helpers/constants';

let Form = (props) => {
    const {
        t,
        handleSubmit,
        reset,
        pristine,
        submitting,
        clientIdentifier,
        useGlobalSettings,
        toggleClientModal,
        processingAdding,
        processingUpdating,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="modal-body">
                <div className="form__group">
                    <div className="form-inline mb-3">
                        <strong className="mr-3">
                            <Trans>client_identifier</Trans>
                        </strong>
                        <label className="mr-3">
                            <Field
                                name="identifier"
                                component={renderField}
                                type="radio"
                                className="form-control mr-2"
                                value="ip"
                            />{' '}
                            <Trans>ip_address</Trans>
                        </label>
                        <label>
                            <Field
                                name="identifier"
                                component={renderField}
                                type="radio"
                                className="form-control mr-2"
                                value="mac"
                            />{' '}
                            MAC
                        </label>
                    </div>
                    {clientIdentifier === CLIENT_ID.IP && (
                        <div className="form__group">
                            <Field
                                id="ip"
                                name="ip"
                                component={renderField}
                                type="text"
                                className="form-control"
                                placeholder={t('form_enter_ip')}
                                validate={[ipv4, required]}
                            />
                        </div>
                    )}
                    {clientIdentifier === CLIENT_ID.MAC && (
                        <div className="form__group">
                            <Field
                                id="mac"
                                name="mac"
                                component={renderField}
                                type="text"
                                className="form-control"
                                placeholder={t('form_enter_mac')}
                                validate={[mac, required]}
                            />
                        </div>
                    )}
                    <div className="form__desc">
                        <Trans
                            components={[
                                <a href="#settings_dhcp" key="0">
                                    link
                                </a>,
                            ]}
                        >
                            client_identifier_desc
                        </Trans>
                    </div>
                </div>

                <div className="form__group">
                    <Field
                        id="name"
                        name="name"
                        component={renderField}
                        type="text"
                        className="form-control"
                        placeholder={t('form_client_name')}
                        validate={[required]}
                    />
                </div>

                <div className="mb-4">
                    <strong>Settings</strong>
                </div>

                <div className="form__group">
                    <Field
                        name="use_global_settings"
                        type="checkbox"
                        component={renderSelectField}
                        placeholder={t('client_global_settings')}
                    />
                </div>

                <div className="form__group">
                    <Field
                        name="filtering_enabled"
                        type="checkbox"
                        component={renderSelectField}
                        placeholder={t('block_domain_use_filters_and_hosts')}
                        disabled={useGlobalSettings}
                    />
                </div>

                <div className="form__group">
                    <Field
                        name="safebrowsing_enabled"
                        type="checkbox"
                        component={renderSelectField}
                        placeholder={t('use_adguard_browsing_sec')}
                        disabled={useGlobalSettings}
                    />
                </div>

                <div className="form__group">
                    <Field
                        name="parental_enabled"
                        type="checkbox"
                        component={renderSelectField}
                        placeholder={t('use_adguard_parental')}
                        disabled={useGlobalSettings}
                    />
                </div>

                <div className="form__group">
                    <Field
                        name="safesearch_enabled"
                        type="checkbox"
                        component={renderSelectField}
                        placeholder={t('enforce_safe_search')}
                        disabled={useGlobalSettings}
                    />
                </div>
            </div>

            <div className="modal-footer">
                <div className="btn-list">
                    <button
                        type="button"
                        className="btn btn-secondary btn-standard"
                        disabled={submitting}
                        onClick={() => {
                            reset();
                            toggleClientModal();
                        }}
                    >
                        <Trans>cancel_btn</Trans>
                    </button>
                    <button
                        type="submit"
                        className="btn btn-success btn-standard"
                        disabled={submitting || pristine || processingAdding || processingUpdating}
                    >
                        <Trans>save_btn</Trans>
                    </button>
                </div>
            </div>
        </form>
    );
};

Form.propTypes = {
    pristine: PropTypes.bool.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    reset: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    toggleClientModal: PropTypes.func.isRequired,
    clientIdentifier: PropTypes.string,
    useGlobalSettings: PropTypes.bool,
    t: PropTypes.func.isRequired,
    processingAdding: PropTypes.bool.isRequired,
    processingUpdating: PropTypes.bool.isRequired,
};

const selector = formValueSelector('clientForm');

Form = connect((state) => {
    const clientIdentifier = selector(state, 'identifier');
    const useGlobalSettings = selector(state, 'use_global_settings');
    return {
        clientIdentifier,
        useGlobalSettings,
    };
})(Form);

export default flow([
    withNamespaces(),
    reduxForm({
        form: 'clientForm',
        enableReinitialize: true,
    }),
])(Form);
