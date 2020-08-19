import React, { useState } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import {
    Field, FieldArray, reduxForm, formValueSelector,
} from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';
import Select from 'react-select';

import i18n from '../../../i18n';
import Tabs from '../../ui/Tabs';
import Examples from '../Dns/Upstream/Examples';
import { toggleAllServices } from '../../../helpers/helpers';
import {
    renderInputField,
    renderGroupField,
    renderCheckboxField,
    renderServiceField,
} from '../../../helpers/form';
import { validateClientId, validateRequiredValue } from '../../../helpers/validators';
import { FORM_NAME, SERVICES } from '../../../helpers/constants';
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
    errors.name = validateRequiredValue(name);

    if (ids && ids.length) {
        const idArrayErrors = [];
        ids.forEach((id, idx) => {
            idArrayErrors[idx] = validateRequiredValue(id) || validateClientId(id);
        });

        if (idArrayErrors.length) {
            errors.ids = idArrayErrors;
        }
    }
    return errors;
};

const renderFieldsWrapper = (placeholder, buttonTitle) => function cell(row) {
    const {
        fields,
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
                        normalizeOnBlur={(data) => data.trim()}
                    />
                </div>
            ))}
            <button
                type="button"
                className="btn btn-link btn-block btn-sm"
                onClick={() => fields.push()}
                title={buttonTitle}
            >
                <svg className="icon icon--24">
                    <use xlinkHref="#plus" />
                </svg>
            </button>
        </div>
    );
};

// Should create function outside of component to prevent component re-renders
const renderFields = renderFieldsWrapper(i18n.t('form_enter_id'), i18n.t('form_add_id'));

const renderMultiselect = (props) => {
    const { input, placeholder, options } = props;

    return (
        <Select
            {...input}
            options={options}
            className="basic-multi-select"
            classNamePrefix="select"
            onChange={(value) => input.onChange(value)}
            onBlur={() => input.onBlur(input.value)}
            placeholder={placeholder}
            blurInputOnSelect={false}
            isMulti
        />
    );
};

renderMultiselect.propTypes = {
    input: PropTypes.object.isRequired,
    placeholder: PropTypes.string,
    options: PropTypes.array,
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
        tagsOptions,
    } = props;

    const [activeTabLabel, setActiveTabLabel] = useState('settings');

    const tabs = {
        settings: {
            title: 'settings',
            component: <div label="settings" title={props.t('main_settings')}>
                {settingsCheckboxes.map((setting) => (
                    <div className="form__group" key={setting.name}>
                        <Field
                            name={setting.name}
                            type="checkbox"
                            component={renderCheckboxField}
                            placeholder={t(setting.placeholder)}
                            disabled={
                                setting.name !== 'use_global_settings'
                                    ? useGlobalSettings
                                    : false
                            }
                        />
                    </div>
                ))}
            </div>,
        },
        block_services: {
            title: 'block_services',
            component: <div label="services" title={props.t('block_services')}>
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
                        {SERVICES.map((service) => (
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
            </div>,
        },
        upstream_dns: {
            title: 'upstream_dns',
            component: <div label="upstream" title={props.t('upstream_dns')}>
                <div className="form__desc mb-3">
                    <Trans components={[<a href="#dns" key="0">link</a>]}>
                        upstream_dns_client_desc
                    </Trans>
                </div>
                <Field
                    id="upstreams"
                    name="upstreams"
                    component="textarea"
                    type="text"
                    className="form-control form-control--textarea mb-5"
                    placeholder={t('upstream_dns')}
                />
                <Examples />
            </div>,
        },
    };

    const activeTab = tabs[activeTabLabel].component;

    return (
        <form onSubmit={handleSubmit}>
            <div className="modal-body">
                <div className="form__group mb-0">
                    <div className="form__group">
                        <Field
                            id="name"
                            name="name"
                            component={renderInputField}
                            type="text"
                            className="form-control"
                            placeholder={t('form_client_name')}
                            normalizeOnBlur={(data) => data.trim()}
                        />
                    </div>

                    <div className="form__group mb-4">
                        <div className="form__label">
                            <strong className="mr-3">
                                <Trans>tags_title</Trans>
                            </strong>
                        </div>
                        <div className="form__desc mt-0 mb-2">
                            <Trans components={[
                                <a href="https://github.com/AdguardTeam/AdGuardHome/wiki/Hosts-Blocklists#ctag"
                                   key="0">link</a>,
                            ]}>
                                tags_desc
                            </Trans>
                        </div>
                        <Field
                            name="tags"
                            component={renderMultiselect}
                            placeholder={t('form_select_tags')}
                            options={tagsOptions}
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
                            component={renderFields}
                        />
                    </div>
                </div>

                <Tabs controlClass="form" tabs={tabs} activeTabLabel={activeTabLabel}
                      setActiveTabLabel={setActiveTabLabel}>
                    {activeTab}
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
                            submitting
                            || invalid
                            || pristine
                            || processingAdding
                            || processingUpdating
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
    tagsOptions: PropTypes.array.isRequired,
};

const selector = formValueSelector(FORM_NAME.CLIENT);

Form = connect((state) => {
    const useGlobalSettings = selector(state, 'use_global_settings');
    const useGlobalServices = selector(state, 'use_global_blocked_services');
    return {
        useGlobalSettings,
        useGlobalServices,
    };
})(Form);

export default flow([
    withTranslation(),
    reduxForm({
        form: FORM_NAME.CLIENT,
        enableReinitialize: true,
        validate,
    }),
])(Form);
