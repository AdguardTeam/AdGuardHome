import React from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { Controller, useForm } from 'react-hook-form';
import i18next from 'i18next';
import cn from 'classnames';

import { getPathWithQueryString } from '../../../helpers/helpers';
import { CLIENT_ID_LINK, MOBILE_CONFIG_LINKS, STANDARD_HTTPS_PORT } from '../../../helpers/constants';
import { toNumber } from '../../../helpers/form';
import {
    validateConfigClientId,
    validateServerName,
    validatePort,
    validateIsSafePort,
} from '../../../helpers/validators';
import { Input } from '../Controls/Input';
import { Select } from '../Controls/Select';

const getDownloadLink = (host: string, clientId: string, protocol: string, invalid: boolean) => {
    if (!host || invalid) {
        return (
            <button type="button" className="btn btn-success btn-standard btn-large disabled">
                {i18next.t('download_mobileconfig')}
            </button>
        );
    }

    const linkParams: { host: string; client_id?: string } = { host };

    if (clientId) {
        linkParams.client_id = clientId;
    }

    return (
        <a
            href={getPathWithQueryString(protocol, linkParams)}
            className={cn('btn btn-success btn-standard btn-large')}
            download>
            {i18next.t('download_mobileconfig')}
        </a>
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
            <div>
                <div className="form__group form__group--settings">
                    <div className="row">
                        <div className="col">
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
                                        error={fieldState.error?.message}
                                    />
                                )}
                            />
                        </div>
                        {protocol === MOBILE_CONFIG_LINKS.DOH && (
                            <div className="col">
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
                                            error={fieldState.error?.message}
                                            onChange={(e) => {
                                                const { value } = e.target;
                                                field.onChange(toNumber(value));
                                            }}
                                        />
                                    )}
                                />
                            </div>
                        )}
                    </div>
                </div>

                <div className="form__group form__group--settings">
                    <label htmlFor="clientId" className="form__label form__label--with-desc">
                        {t('client_id')}
                    </label>

                    <div className="form__desc form__desc--top">
                        <Trans
                            components={{ a: <a href={CLIENT_ID_LINK} target="_blank" rel="noopener noreferrer" /> }}>
                            client_id_desc
                        </Trans>
                    </div>

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
                                error={fieldState.error?.message}
                            />
                        )}
                    />
                </div>

                <div className="form__group form__group--settings">
                    <Controller
                        name="protocol"
                        control={control}
                        render={({ field }) => (
                            <Select {...field} label={t('protocol')} data-testid="mobile_config_protocol">
                                <option value={MOBILE_CONFIG_LINKS.DOT}>{t('dns_over_tls')}</option>
                                <option value={MOBILE_CONFIG_LINKS.DOH}>{t('dns_over_https')}</option>
                            </Select>
                        )}
                    />
                </div>
            </div>

            {getDownloadLink(getHostName(), clientId, protocol, !isValid)}
        </form>
    );
};
