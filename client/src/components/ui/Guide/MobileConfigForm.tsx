import React from 'react';
import { Trans } from 'react-i18next';
import { useForm } from 'react-hook-form';
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

const getDownloadLink = (host: string, clientId: string, protocol: string, invalid: boolean) => {
    if (!host || invalid) {
        return (
            <button type="button" className="btn btn-success btn-standard btn-large disabled">
                {i18next.t('download_mobileconfig')}
            </button>
        );
    }

    const linkParams: { host: string, client_id?: string } = { host };

    if (clientId) {
        linkParams.client_id = clientId;
    }

    return (
        <a
            href={getPathWithQueryString(protocol, linkParams)}
            className={cn('btn btn-success btn-standard btn-large')}
            download
        >
            {i18next.t('download_mobileconfig')}
        </a>
    );
};

const githubLink = (
    <a href={CLIENT_ID_LINK} target="_blank" rel="noopener noreferrer">
        text
    </a>
);

type FormValues = {
    host: string;
    clientId: string;
    protocol: string;
    port?: number;
}

type Props = {
    initialValues?: FormValues;
}

export const MobileConfigForm = ({ initialValues }: Props) => {
    const {
        register,
        watch,
        formState: { errors, isValid },
      } = useForm<FormValues>({
        mode: 'onChange',
        defaultValues: {
          host: initialValues?.host || '',
          clientId: initialValues?.clientId || '',
          protocol: initialValues?.protocol || MOBILE_CONFIG_LINKS.DOT,
          port: initialValues?.port || undefined,
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
                            <label htmlFor="host" className="form__label">
                                {i18next.t('dhcp_table_hostname')}
                            </label>

                            <input
                                id="host"
                                type="text"
                                className="form-control"
                                placeholder={i18next.t('form_enter_hostname')}
                                {...register('host', {
                                    validate: validateServerName,
                                })}
                            />
                            {errors.host && (
                                <div className="form__message form__message--error">
                                    {errors.host.message}
                                </div>
                            )}
                        </div>
                        {protocol === MOBILE_CONFIG_LINKS.DOH && (
                            <div className="col">
                                <label htmlFor="port" className="form__label">
                                    {i18next.t('encryption_https')}
                                </label>

                                <input
                                    id="port"
                                    type="number"
                                    className="form-control"
                                    placeholder={i18next.t('encryption_https')}
                                    {...register('port', {
                                        setValueAs: (val) => toNumber(val),
                                        validate: {
                                            range: (value) => validatePort(value) || true,
                                            safety: (value) => validateIsSafePort(value) || true,
                                        },
                                    })}
                                />

                                {errors.port && (
                                    <div className="form__message form__message--error">
                                        {errors.port.message}
                                    </div>
                                )}
                            </div>
                        )}
                    </div>
                </div>

                <div className="form__group form__group--settings">
                    <label htmlFor="clientId" className="form__label form__label--with-desc">
                        {i18next.t('client_id')}
                    </label>

                    <div className="form__desc form__desc--top">
                        <Trans components={{ a: githubLink }}>client_id_desc</Trans>
                    </div>

                    <input
                        id="clientId"
                        type="text"
                        className="form-control"
                        placeholder={i18next.t('client_id_placeholder')}
                        {...register('clientId', {
                            validate: validateConfigClientId,
                        })}
                    />
                    
                    {errors.clientId && (
                        <div className="form__message form__message--error">
                            {errors.clientId.message}
                        </div>
                    )}
                </div>

                <div className="form__group form__group--settings">
                    <label htmlFor="protocol" className="form__label">
                        {i18next.t('protocol')}
                    </label>

                    <select
                        id="protocol"
                        className="form-control"
                        {...register('protocol')}
                    >
                        <option value={MOBILE_CONFIG_LINKS.DOT}>{i18next.t('dns_over_tls')}</option>
                        <option value={MOBILE_CONFIG_LINKS.DOH}>{i18next.t('dns_over_https')}</option>
                    </select>
                </div>
            </div>

            {getDownloadLink(getHostName(), clientId, protocol, !isValid)}
        </form>
    );
};
