import React, { useState } from 'react';
import { useSelector } from 'react-redux';
import { Trans, useTranslation } from 'react-i18next';
import { Controller, FormProvider, useForm } from 'react-hook-form';
import Select from 'react-select';

import Tabs from '../../../ui/Tabs';
import { CLIENT_ID_LINK, LOCAL_TIMEZONE_VALUE } from '../../../../helpers/constants';
import { RootState } from '../../../../initialState';
import { Input } from '../../../ui/Controls/Input';
import { validateRequiredValue } from '../../../../helpers/validators';
import { ClientForm } from './types';
import { BlockedServices, ClientIds, MainSettings, ScheduleServices, UpstreamDns } from './components';

import '../Service.css';

const defaultFormValues: ClientForm = {
    ids: [{ name: '' }],
    name: '',
    tags: [],
    use_global_settings: false,
    filtering_enabled: false,
    safebrowsing_enabled: false,
    parental_enabled: false,
    ignore_querylog: false,
    ignore_statistics: false,
    blocked_services: {},
    safe_search: { enabled: false },
    upstreams: '',
    upstreams_cache_enabled: false,
    upstreams_cache_size: 0,
    use_global_blocked_services: false,
    blocked_services_schedule: {
        time_zone: LOCAL_TIMEZONE_VALUE,
    },
};

type Props = {
    onSubmit: (values: ClientForm) => void;
    onClose: () => void;
    useGlobalSettings?: boolean;
    useGlobalServices?: boolean;
    blockedServicesSchedule?: {
        time_zone: string;
    };
    processingAdding: boolean;
    processingUpdating: boolean;
    tagsOptions: { label: string; value: string }[];
    initialValues?: ClientForm;
};

export const Form = ({
    onSubmit,
    onClose,
    processingAdding,
    processingUpdating,
    tagsOptions,
    initialValues,
}: Props) => {
    const { t } = useTranslation();
    const methods = useForm<ClientForm>({
        defaultValues: {
            ...defaultFormValues,
            ...initialValues,
        },
        mode: 'onBlur',
    });

    const {
        handleSubmit,
        reset,
        control,
        formState: { isSubmitting, isValid },
    } = methods;

    const services = useSelector((store: RootState) => store?.services);
    const { safe_search } = initialValues;
    const safeSearchServices = { ...safe_search };
    delete safeSearchServices.enabled;

    const [activeTabLabel, setActiveTabLabel] = useState('settings');

    const tabs = {
        settings: {
            title: 'settings',
            component: <MainSettings safeSearchServices={safeSearchServices} />,
        },
        block_services: {
            title: 'block_services',
            component: <BlockedServices services={services?.allServices} />,
        },
        schedule_services: {
            title: 'schedule_services',
            component: <ScheduleServices />,
        },
        upstream_dns: {
            title: 'upstream_dns',
            component: <UpstreamDns />,
        },
    };

    const activeTab = tabs[activeTabLabel].component;

    return (
        <FormProvider {...methods}>
            <form onSubmit={handleSubmit(onSubmit)}>
                <div className="modal-body">
                    <div className="form__group mb-0">
                        <div className="form__group">
                            <Controller
                                name="name"
                                control={control}
                                rules={{ validate: validateRequiredValue }}
                                render={({ field, fieldState }) => (
                                    <Input
                                        {...field}
                                        type="text"
                                        data-testid="clients_name"
                                        placeholder={t('form_client_name')}
                                        error={fieldState.error?.message}
                                        onBlur={(event) => {
                                            const trimmedValue = event.target.value.trim();
                                            field.onBlur();
                                            field.onChange(trimmedValue);
                                        }}
                                    />
                                )}
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
                                            key="0"
                                        />,
                                    ]}>
                                    tags_desc
                                </Trans>
                            </div>

                            <Controller
                                name="tags"
                                control={control}
                                render={({ field }) => (
                                    <Select
                                        {...field}
                                        data-testid="clients_tags"
                                        options={tagsOptions}
                                        className="basic-multi-select"
                                        classNamePrefix="select"
                                        isMulti
                                    />
                                )}
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
                                        <a href={CLIENT_ID_LINK} target="_blank" rel="noopener noreferrer" key="0" />,
                                    ]}>
                                    client_identifier_desc
                                </Trans>
                            </div>
                        </div>

                        <div className="form__group">
                            <ClientIds />
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
                            disabled={isSubmitting}
                            onClick={() => {
                                reset();
                                onClose();
                            }}>
                            <Trans>cancel_btn</Trans>
                        </button>

                        <button
                            type="submit"
                            className="btn btn-success btn-standard"
                            disabled={isSubmitting || !isValid || processingAdding || processingUpdating}>
                            <Trans>save_btn</Trans>
                        </button>
                    </div>
                </div>
            </form>
        </FormProvider>
    );
};
