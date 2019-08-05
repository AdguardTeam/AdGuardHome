import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import Tabs from '../../ui/Tabs';
import { toggleAllServices } from '../../../helpers/helpers';
import { renderField, renderRadioField, renderSelectField, renderServiceField, ipv4, mac, required } from '../../../helpers/form';
import { CLIENT_ID, SERVICES } from '../../../helpers/constants';
import './Service.css';

const settingsCheckboxes = [
    {
        name: 'use_global_settings',
        placeholder: 'client_global_settings',
    },
    {
        name: 'filtering_enabled',
        placeholder: 'block_domain_use_filters_and_hosts',
    },
    {
        name: 'safebrowsing_enabled',
        placeholder: 'use_adguard_browsing_sec',
    },
    {
        name: 'parental_enabled',
        placeholder: 'use_adguard_parental',
    },
    {
        name: 'safesearch_enabled',
        placeholder: 'enforce_safe_search',
    },
];

let Form = (props) => {
    const {
        t,
        handleSubmit,
        reset,
        change,
        pristine,
        submitting,
        clientIdentifier,
        useGlobalSettings,
        useGlobalServices,
        toggleClientModal,
        processingAdding,
        processingUpdating,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="modal-body">
                <div className="form__group">
                    <div className="form__inline mb-2">
                        <strong className="mr-3">
                            <Trans>client_identifier</Trans>
                        </strong>
                        <div className="custom-controls-stacked">
                            <Field
                                name="identifier"
                                component={renderRadioField}
                                type="radio"
                                className="form-control mr-2"
                                value="ip"
                                placeholder={t('ip_address')}
                            />
                            <Field
                                name="identifier"
                                component={renderRadioField}
                                type="radio"
                                className="form-control mr-2"
                                value="mac"
                                placeholder="MAC"
                            />
                        </div>
                    </div>
                    <div className="row">
                        <div className="col col-sm-6">
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
                        </div>
                        <div className="col col-sm-6">
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
                    </div>
                    <div className="form__desc">
                        <Trans
                            components={[
                                <a href="#dhcp" key="0">
                                    link
                                </a>,
                            ]}
                        >
                            client_identifier_desc
                        </Trans>
                    </div>
                </div>

                <Tabs controlClass="form">
                    <div label="settings" title={props.t('main_settings')}>
                        {settingsCheckboxes.map(setting => (
                            <div className="form__group" key={setting.name}>
                                <Field
                                    name={setting.name}
                                    type="checkbox"
                                    component={renderSelectField}
                                    placeholder={t(setting.placeholder)}
                                    disabled={setting.name !== 'use_global_settings' ? useGlobalSettings : false}
                                />
                            </div>
                        ))}
                    </div>
                    <div label="services" title={props.t('block_services')}>
                        <div className="form__group">
                            <Field
                                name="use_global_blocked_services"
                                type="checkbox"
                                component={renderServiceField}
                                placeholder={t('blocked_services_global')}
                                modifier="service--global"
                            />
                            <div className="row mb-4">
                                <div className="col-6">
                                    <button
                                        type="button"
                                        className="btn btn-secondary btn-block"
                                        disabled={useGlobalServices}
                                        onClick={() => toggleAllServices(SERVICES, change, true)}
                                    >
                                        <Trans>block_all</Trans>
                                    </button>
                                </div>
                                <div className="col-6">
                                    <button
                                        type="button"
                                        className="btn btn-secondary btn-block"
                                        disabled={useGlobalServices}
                                        onClick={() => toggleAllServices(SERVICES, change, false)}
                                    >
                                        <Trans>unblock_all</Trans>
                                    </button>
                                </div>
                            </div>
                            <div className="services">
                                {SERVICES.map(service => (
                                    <Field
                                        key={service.id}
                                        icon={`service_${service.id}`}
                                        name={`blocked_services.${service.id}`}
                                        type="checkbox"
                                        component={renderServiceField}
                                        placeholder={service.name}
                                        disabled={useGlobalServices}
                                    />
                                ))}
                            </div>
                        </div>
                    </div>
                </Tabs>
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
    change: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    toggleClientModal: PropTypes.func.isRequired,
    clientIdentifier: PropTypes.string,
    useGlobalSettings: PropTypes.bool,
    useGlobalServices: PropTypes.bool,
    t: PropTypes.func.isRequired,
    processingAdding: PropTypes.bool.isRequired,
    processingUpdating: PropTypes.bool.isRequired,
};

const selector = formValueSelector('clientForm');

Form = connect((state) => {
    const clientIdentifier = selector(state, 'identifier');
    const useGlobalSettings = selector(state, 'use_global_settings');
    const useGlobalServices = selector(state, 'use_global_blocked_services');
    return {
        clientIdentifier,
        useGlobalSettings,
        useGlobalServices,
    };
})(Form);

export default flow([
    withNamespaces(),
    reduxForm({
        form: 'clientForm',
        enableReinitialize: true,
    }),
])(Form);
