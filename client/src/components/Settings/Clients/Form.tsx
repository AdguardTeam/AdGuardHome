import React, { useState } from 'react';
import { connect, useSelector } from 'react-redux';
import { Field, FieldArray, reduxForm, formValueSelector, FormErrors } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import Select from 'react-select';

import i18n from '../../../i18n';

import Tabs from '../../ui/Tabs';

import Examples from '../Dns/Upstream/Examples';

import { ScheduleForm } from '../../Filters/Services/ScheduleForm';
import { toggleAllServices, trimLinesAndRemoveEmpty, captitalizeWords } from '../../../helpers/helpers';
import {
    toNumber,
    renderInputField,
    renderGroupField,
    CheckboxField,
    renderServiceField,
    renderTextareaField,
} from '../../../helpers/form';
import { validateClientId, validateRequiredValue } from '../../../helpers/validators';
import { CLIENT_ID_LINK, FORM_NAME, UINT32_RANGE } from '../../../helpers/constants';
import './Service.css';
import { RootState } from '../../../initialState';

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
];

const logAndStatsCheckboxes = [
    {
        name: 'ignore_querylog',
        placeholder: 'ignore_query_log',
    },
    {
        name: 'ignore_statistics',
        placeholder: 'ignore_statistics',
    },
];
const validate = (values: any): FormErrors<any, string> => {
    const errors: {
        name?: string;
        ids?: string[];
    } = {};
    const { name, ids } = values;

    errors.name = validateRequiredValue(name);

    if (ids && ids.length) {
        const idArrayErrors: any = [];
        ids.forEach((id: any, idx: any) => {
            idArrayErrors[idx] = validateRequiredValue(id) || validateClientId(id);
        });

        if (idArrayErrors.length) {
            errors.ids = idArrayErrors;
        }
    }
    // @ts-expect-error FIXME: ts migration
    return errors;
};

const renderFieldsWrapper = (placeholder: any, buttonTitle: any) =>
    function cell(row: any) {
        const { fields } = row;
        return (
            <div className="form__group">
                {fields.map((ip: any, index: any) => (
                    <div key={index} className="mb-1">
                        <Field
                            name={ip}
                            component={renderGroupField}
                            type="text"
                            className="form-control"
                            placeholder={placeholder}
                            isActionAvailable={index !== 0}
                            removeField={() => fields.remove(index)}
                            normalizeOnBlur={(data: any) => data.trim()}
                        />
                    </div>
                ))}

                <button
                    type="button"
                    className="btn btn-link btn-block btn-sm"
                    onClick={() => fields.push()}
                    title={buttonTitle}>
                    <svg className="icon icon--24">
                        <use xlinkHref="#plus" />
                    </svg>
                </button>
            </div>
        );
    };

// Should create function outside of component to prevent component re-renders
const renderFields = renderFieldsWrapper(i18n.t('form_enter_id'), i18n.t('form_add_id'));

interface renderMultiselectProps {
    input: {
        name: string;
        value: string;
        checked: boolean;
        onChange: (...args: unknown[]) => unknown;
        onBlur: (...args: unknown[]) => unknown;
    };
    placeholder?: string;
    options?: unknown[];
}

const renderMultiselect = (props: renderMultiselectProps) => {
    const { input, placeholder, options } = props;

    return (
        <Select
            {...input}
            options={options}
            className="basic-multi-select"
            classNamePrefix="select"
            onChange={(value: any) => input.onChange(value)}
            onBlur={() => input.onBlur(input.value)}
            placeholder={placeholder}
            blurInputOnSelect={false}
            isMulti
        />
    );
};

interface FormProps {
    pristine: boolean;
    handleSubmit: (...args: unknown[]) => string;
    reset: (...args: unknown[]) => string;
    change: (...args: unknown[]) => unknown;
    submitting: boolean;
    handleClose: (...args: unknown[]) => unknown;
    useGlobalSettings?: boolean;
    useGlobalServices?: boolean;
    blockedServicesSchedule?: {
        time_zone: string;
    };
    t: (...args: unknown[]) => string;
    processingAdding: boolean;
    processingUpdating: boolean;
    invalid: boolean;
    tagsOptions: unknown[];
    initialValues?: {
        safe_search: any;
    };
}

let Form = (props: FormProps) => {
    const {
        t,
        handleSubmit,
        reset,
        change,
        submitting,
        useGlobalSettings,
        useGlobalServices,
        blockedServicesSchedule,
        handleClose,
        processingAdding,
        processingUpdating,
        invalid,
        tagsOptions,
        initialValues,
    } = props;

    const services = useSelector((store: RootState) => store?.services);
    const { safe_search } = initialValues;
    const safeSearchServices = { ...safe_search };
    delete safeSearchServices.enabled;

    const [activeTabLabel, setActiveTabLabel] = useState('settings');

    const handleScheduleSubmit = (values: any) => {
        change('blocked_services_schedule', { ...values });
    };

    const tabs = {
        settings: {
            title: 'settings',

            component: (
                <div title={props.t('main_settings')}>
                    <div className="form__label--bot form__label--bold">{t('protection_section_label')}</div>
                    {settingsCheckboxes.map((setting) => (
                        <div className="form__group" key={setting.name}>
                            <Field
                                name={setting.name}
                                type="checkbox"
                                component={CheckboxField}
                                placeholder={t(setting.placeholder)}
                                disabled={setting.name !== 'use_global_settings' ? useGlobalSettings : false}
                            />
                        </div>
                    ))}

                    <div className="form__group">
                        <Field
                            name="safe_search.enabled"
                            type="checkbox"
                            component={CheckboxField}
                            placeholder={t('enforce_safe_search')}
                            disabled={useGlobalSettings}
                        />
                    </div>

                    <div className="form__group--inner">
                        {Object.keys(safeSearchServices).map((searchKey) => (
                            <div key={searchKey}>
                                <Field
                                    name={`safe_search.${searchKey}`}
                                    type="checkbox"
                                    component={CheckboxField}
                                    placeholder={captitalizeWords(searchKey)}
                                    disabled={useGlobalSettings}
                                />
                            </div>
                        ))}
                    </div>

                    <div className="form__label--bold form__label--top form__label--bot">
                        {t('log_and_stats_section_label')}
                    </div>
                    {logAndStatsCheckboxes.map((setting) => (
                        <div className="form__group" key={setting.name}>
                            <Field
                                name={setting.name}
                                type="checkbox"
                                component={CheckboxField}
                                placeholder={t(setting.placeholder)}
                            />
                        </div>
                    ))}
                </div>
            ),
        },
        block_services: {
            title: 'block_services',

            component: (
                <div title={props.t('block_services')}>
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
                                    onClick={() => toggleAllServices(services.allServices, change, true)}>
                                    <Trans>block_all</Trans>
                                </button>
                            </div>

                            <div className="col-6">
                                <button
                                    type="button"
                                    className="btn btn-secondary btn-block"
                                    disabled={useGlobalServices}
                                    onClick={() => toggleAllServices(services.allServices, change, false)}>
                                    <Trans>unblock_all</Trans>
                                </button>
                            </div>
                        </div>
                        {services.allServices.length > 0 && (
                            <div className="services">
                                {services.allServices.map((service: any) => (
                                    <Field
                                        key={service.id}
                                        icon={service.icon_svg}
                                        name={`blocked_services.${service.id}`}
                                        type="checkbox"
                                        component={renderServiceField}
                                        placeholder={service.name}
                                        disabled={useGlobalServices}
                                    />
                                ))}
                            </div>
                        )}
                    </div>
                </div>
            ),
        },
        schedule_services: {
            title: 'schedule_services',
            component: (
                <>
                    <div className="form__desc mb-4">
                        <Trans>schedule_services_desc_client</Trans>
                    </div>

                    <ScheduleForm
                        schedule={blockedServicesSchedule}
                        onScheduleSubmit={handleScheduleSubmit}
                        clientForm
                    />
                </>
            ),
        },
        upstream_dns: {
            title: 'upstream_dns',

            component: (
                <div title={props.t('upstream_dns')}>
                    <div className="form__desc mb-3">
                        <Trans
                            components={[
                                <a href="#dns" key="0">
                                    link
                                </a>,
                            ]}>
                            upstream_dns_client_desc
                        </Trans>
                    </div>

                    <Field
                        id="upstreams"
                        name="upstreams"
                        component={renderTextareaField}
                        type="text"
                        className="form-control form-control--textarea mb-5"
                        placeholder={t('upstream_dns')}
                        normalizeOnBlur={trimLinesAndRemoveEmpty}
                    />

                    <Examples />

                    <div className="form__label--bold mt-5 mb-3">{t('upstream_dns_cache_configuration')}</div>

                    <div className="form__group mb-2">
                        <Field
                            name="upstreams_cache_enabled"
                            type="checkbox"
                            component={CheckboxField}
                            placeholder={t('enable_upstream_dns_cache')}
                        />
                    </div>

                    <div className="form__group form__group--settings">
                        <label htmlFor="upstreams_cache_size" className="form__label">
                            {t('dns_cache_size')}
                        </label>

                        <Field
                            name="upstreams_cache_size"
                            type="number"
                            component={renderInputField}
                            placeholder={t('enter_cache_size')}
                            className="form-control"
                            normalize={toNumber}
                            min={0}
                            max={UINT32_RANGE.MAX}
                        />
                    </div>
                </div>
            ),
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
                            normalizeOnBlur={(data: any) => data.trim()}
                        />
                    </div>

                    <div className="form__group mb-4">
                        <div className="form__label">
                            <strong className="mr-3">
                                <Trans>tags_title</Trans>
                            </strong>
                        </div>

                        <div className="form__desc mt-0 mb-2">
                            <Trans
                                components={[
                                    <a
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        href="https://link.adtidy.org/forward.html?action=dns_kb_filtering_syntax_ctag&from=ui&app=home"
                                        key="0">
                                        link
                                    </a>,
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
                                    <a href={CLIENT_ID_LINK} target="_blank" rel="noopener noreferrer" key="0">
                                        text
                                    </a>,
                                ]}>
                                client_identifier_desc
                            </Trans>
                        </div>
                    </div>

                    <div className="form__group">
                        <FieldArray name="ids" component={renderFields} />
                    </div>
                </div>

                <Tabs
                    controlClass="form"
                    tabs={tabs}
                    activeTabLabel={activeTabLabel}
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
                            handleClose();
                        }}>
                        <Trans>cancel_btn</Trans>
                    </button>

                    <button
                        type="submit"
                        className="btn btn-success btn-standard"
                        disabled={submitting || invalid || processingAdding || processingUpdating}>
                        <Trans>save_btn</Trans>
                    </button>
                </div>
            </div>
        </form>
    );
};

const selector = formValueSelector(FORM_NAME.CLIENT);

Form = connect((state) => {
    const useGlobalSettings = selector(state, 'use_global_settings');
    const useGlobalServices = selector(state, 'use_global_blocked_services');
    const blockedServicesSchedule = selector(state, 'blocked_services_schedule');
    return {
        useGlobalSettings,
        useGlobalServices,
        blockedServicesSchedule,
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
