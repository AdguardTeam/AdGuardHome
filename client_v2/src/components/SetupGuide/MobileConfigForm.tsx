import React from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { Controller, useForm } from 'react-hook-form';
import i18next from 'i18next';
import cn from 'clsx';

import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import { getPathWithQueryString } from '../../helpers/helpers';
import { CLIENT_ID_LINK, MOBILE_CONFIG_LINKS, STANDARD_HTTPS_PORT } from '../../helpers/constants';
import { toNumber } from '../../helpers/form';
import {
    validateConfigClientId,
    validateServerName,
    validatePort,
    validateIsSafePort,
} from '../../helpers/validators';
import { Button } from '../../common/ui/Button';

import s from './MobileConfigForm.module.pcss';

const getDownloadLink = (host: string, clientId: string, protocol: string, invalid: boolean) => {
    if (!host || invalid) {
        return (
            <Button variant="primary" disabled>
                {i18next.t('download_mobileconfig')}
            </Button>
        );
    }

    const linkParams: { host: string; client_id?: string } = { host };

    if (clientId) {
        linkParams.client_id = clientId;
    }

    const handleDownload = () => {
        const link = document.createElement('a');
        link.href = getPathWithQueryString(protocol, linkParams);
        link.download = '';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    };

    return (
        <Button variant="primary" onClick={handleDownload}>
            {i18next.t('download_mobileconfig')}
        </Button>
    );
};

type FormValues = {
    host: string;
    clientId: string;
    protocol: string;
    port?: number;
};

type Props = {
    initialValues?: FormValues;
};

const defaultFormValues = {
    host: '',
    clientId: '',
    protocol: MOBILE_CONFIG_LINKS.DOT,
    port: undefined,
};

export const MobileConfigForm = ({ initialValues }: Props) => {
    const { t } = useTranslation();

    const {
        watch,
        control,
        formState: { isValid },
    } = useForm<FormValues>({
        mode: 'onBlur',
        defaultValues: {
            ...defaultFormValues,
            ...initialValues,
        },
    });

    const protocol = watch('protocol');
    const host = watch('host');
    const clientId = watch('clientId');
    const port = watch('port');

    const getHostName = () => {
        if (port && port !== STANDARD_HTTPS_PORT && protocol === MOBILE_CONFIG_LINKS.DOH) {
            return `${host}:${port}`;
        }

        return host;
    };


    return (
        <form onSubmit={(e) => e.preventDefault()}>
            <div className={s.form}>
                <div className={cn(s.formGroup, s.formGroupSettings)}>
                    <Controller
                        name="host"
                        control={control}
                        rules={{ validate: validateServerName }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                type="text"
                                data-testid="mobile_config_host"
                                label={t('dhcp_table_hostname')}
                                placeholder={t('form_enter_hostname')}
                                error={!!fieldState.error}
                                errorMessage={fieldState.error?.message}
                            />
                        )}
                    />
                </div>
                {protocol === MOBILE_CONFIG_LINKS.DOH && (
                    <div className={cn(s.formGroup, s.formGroupSettings)}>
                        <Controller
                            name="port"
                            control={control}
                            rules={{
                                validate: {
                                    range: (value) => validatePort(value) || true,
                                    safety: (value) => validateIsSafePort(value) || true,
                                },
                            }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    type="number"
                                    data-testid="mobile_config_port"
                                    label={t('encryption_https')}
                                    placeholder={t('encryption_https')}
                                    error={!!fieldState.error}
                                    errorMessage={fieldState.error?.message}
                                    onChange={(e) => {
                                        const { value } = e.target;
                                        field.onChange(toNumber(value));
                                    }}
                                />
                            )}
                        />
                    </div>
                )}


                <div className={cn(s.formGroup, s.formGroupSettings)}>
                    <label htmlFor="clientId" className={cn(s.formLabel, s.formLabelWithDesc)}>
                        {t('client_id')}
                        <FaqTooltip
                            text={
                                <Trans
                                    i18nKey="client_id_faq"
                                    components={{
                                        0: <a href={CLIENT_ID_LINK} target="_blank" rel="noopener noreferrer" />
                                    }}
                                />
                            }
                        />
                    </label>

                    <Controller
                        name="clientId"
                        control={control}
                        rules={{
                            validate: validateConfigClientId,
                        }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                type="text"
                                data-testid="mobile_config_client_id"
                                placeholder={t('client_id_placeholder')}
                                error={!!fieldState.error}
                                errorMessage={fieldState.error?.message}
                            />
                        )}
                    />
                </div>

                <div className={cn(s.formGroup, s.formGroupSettings)}>
                    <label className={s.formLabel}>
                        {t('protocol')}
                    </label>
                    <Controller
                        name="protocol"
                        control={control}
                        render={({ field }) => (
                            <Select
                                options={[
                                    { value: MOBILE_CONFIG_LINKS.DOT, label: t('dns_over_tls') },
                                    { value: MOBILE_CONFIG_LINKS.DOH, label: t('dns_over_https') }
                                ]}
                                value={{ value: field.value, label: field.value === MOBILE_CONFIG_LINKS.DOT ? t('dns_over_tls') : t('dns_over_https') }}
                                onChange={(option) => field.onChange(option?.value)}
                                isSearchable={false}
                                size="responsive"
                                height="big"
                            />
                        )}
                    />
                </div>
            </div>

            {getDownloadLink(getHostName(), clientId, protocol, !isValid)}
        </form>
    );
};
