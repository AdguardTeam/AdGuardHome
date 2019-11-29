import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, FieldArray, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import i18n from '../../../i18n';
import Tabs from '../../ui/Tabs';
import { toggleAllServices } from '../../../helpers/helpers';
import {
    renderField,
    renderGroupField,
    renderSelectField,
    renderServiceField,
} from '../../../helpers/form';
import { SERVICES } from '../../../helpers/constants';
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

const validate = (values) => {
    const errors = {};
    const { name, ids } = values;

    if (!name || !name.length) {
        errors.name = i18n.t('form_error_required');
    }

    if (ids && ids.length) {
        const idArrayErrors = [];
        ids.forEach((id, idx) => {
            if (!id || !id.length) {
                idArrayErrors[idx] = i18n.t('form_error_required');
            }
        });

        if (idArrayErrors.length) {
            errors.ids = idArrayErrors;
        }
    }

    return errors;
};

const renderFields = (placeholder, buttonTitle) =>
    function cell(row) {
        const {
            fields,
            meta: { error },
        } = row;

        return (
            <div className="form__group">
                {fields.map((ip, index) => (
                    <div key={index} className="mb-1">
                        <Field
                            name={ip}
                            component={renderGroupField}
                            type="text"
                            className="form-control"
                            placeholder={placeholder}
                            isActionAvailable={index !== 0}
                            removeField={() => fields.remove(index)}
                        />
                    </div>
                ))}
                <button
                    type="button"
                    className="btn btn-link btn-block btn-sm"
                    onClick={() => fields.push()}
                    title={buttonTitle}
                >
                    <svg className="icon icon--close">
                        <use xlinkHref="#plus" />
                    </svg>
                </button>
                {error && <div className="error">{error}</div>}
            </div>
        );
    };

let Form = (props) => {
    const {
        t,
        handleSubmit,
        reset,
        change,
        pristine,
        submitting,
        useGlobalSettings,
        useGlobalServices,
        toggleClientModal,
        processingAdding,
        processingUpdating,
        invalid,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="modal-body">
                <div className="form__group mb-0">
                    <div className="form__group">
                        <Field
                            id="name"
                            name="name"
                            component={renderField}
                            type="text"
                            className="form-control"
                            placeholder={t('form_client_name')}
                        />
                    </div>

                    <div className="form__group">
                        <div className="form__label">
                            <strong className="mr-3">
                                <Trans>client_identifier</Trans>
                            </strong>
                        </div>
                        <div className="form__desc mt-0">
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

                    <div className="form__group">
                        <FieldArray
                            name="ids"
                            component={renderFields(t('form_enter_id'), t('form_add_id'))}
                        />
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
                                    disabled={
                                        setting.name !== 'use_global_settings'
                                            ? useGlobalSettings
                                            : false
                                    }
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
                        disabled={
                            submitting ||
                            invalid ||
                            pristine ||
                            processingAdding ||
                            processingUpdating
                        }
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
    useGlobalSettings: PropTypes.bool,
    useGlobalServices: PropTypes.bool,
    t: PropTypes.func.isRequired,
    processingAdding: PropTypes.bool.isRequired,
    processingUpdating: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
};

const selector = formValueSelector('clientForm');

Form = connect((state) => {
    const useGlobalSettings = selector(state, 'use_global_settings');
    const useGlobalServices = selector(state, 'use_global_blocked_services');
    return {
        useGlobalSettings,
        useGlobalServices,
    };
})(Form);

export default flow([
    withNamespaces(),
    reduxForm({
        form: 'clientForm',
        enableReinitialize: true,
        validate,
    }),
])(Form);
