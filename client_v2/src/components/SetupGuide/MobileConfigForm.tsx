import React from 'react';
import { Controller, useForm } from 'react-hook-form';
import cn from 'clsx';

import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import intl from 'panel/common/intl';
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
            <Button className={s.configLink} variant="primary" disabled>
                {intl.getMessage('download_mobileconfig')}
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
            {intl.getMessage('download_mobileconfig')}
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
                                label={intl.getMessage('dhcp_table_hostname')}
                                placeholder={intl.getMessage('form_enter_hostname')}
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
                                    label={intl.getMessage('encryption_https')}
                                    placeholder={intl.getMessage('encryption_https')}
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
                        {intl.getMessage('client_id')}
                        <FaqTooltip
                            text={
                                intl.getMessage('client_id_faq', {
                                    a: (text: string) => (
                                        <a
                                            href={CLIENT_ID_LINK}
                                            target="_blank"
                                            rel="noreferrer"
                                            className={s.dnsLink}
                                        >
                                            {text}
                                        </a>
                                    ),
                                })
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
                                placeholder={intl.getMessage('client_id_placeholder')}
                                error={!!fieldState.error}
                                errorMessage={fieldState.error?.message}
                            />
                        )}
                    />
                </div>

                <div className={cn(s.formGroup, s.formGroupSettings)}>
                    <label className={s.formLabel}>
                        {intl.getMessage('protocol')}
                    </label>
                    <Controller
                        name="protocol"
                        control={control}
                        render={({ field }) => (
                            <Select
                                options={[
                                    { value: MOBILE_CONFIG_LINKS.DOT, label: intl.getMessage('dns_over_tls') },
                                    { value: MOBILE_CONFIG_LINKS.DOH, label: intl.getMessage('dns_over_https') }
                                ]}
                                value={{ value: field.value, label: field.value === MOBILE_CONFIG_LINKS.DOT ? intl.getMessage('dns_over_tls') : intl.getMessage('dns_over_https') }}
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
