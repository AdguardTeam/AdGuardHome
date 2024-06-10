import React from 'react';
import { Trans } from 'react-i18next';
import { useSelector } from 'react-redux';

import { Field, reduxForm } from 'redux-form';
import i18next from 'i18next';
import cn from 'classnames';

import { getPathWithQueryString } from '../../../helpers/helpers';
import { CLIENT_ID_LINK, FORM_NAME, MOBILE_CONFIG_LINKS, STANDARD_HTTPS_PORT } from '../../../helpers/constants';
import { renderInputField, renderSelectField, toNumber } from '../../../helpers/form';
import {
    validateConfigClientId,
    validateServerName,
    validatePort,
    validateIsSafePort,
} from '../../../helpers/validators';
import { RootState } from '../../../initialState';

const getDownloadLink = (host: any, clientId: any, protocol: any, invalid: any) => {
    if (!host || invalid) {
        return (
            <button type="button" className="btn btn-success btn-standard btn-large disabled">
                <Trans>download_mobileconfig</Trans>
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
            download>
            <Trans>download_mobileconfig</Trans>
        </a>
    );
};

interface MobileConfigFormProps {
    invalid: boolean;
}

const MobileConfigForm = ({ invalid }: MobileConfigFormProps) => {
    const formValues = useSelector((state: RootState) => state.form[FORM_NAME.MOBILE_CONFIG]?.values);

    if (!formValues) {
        return null;
    }

    const { host, clientId, protocol, port } = formValues;

    const githubLink = (
        <a href={CLIENT_ID_LINK} target="_blank" rel="noopener noreferrer">
            text
        </a>
    );

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

                            <Field
                                name="host"
                                type="text"
                                component={renderInputField}
                                className="form-control"
                                placeholder={i18next.t('form_enter_hostname')}
                                validate={validateServerName}
                            />
                        </div>
                        {protocol === MOBILE_CONFIG_LINKS.DOH && (
                            <div className="col">
                                <label htmlFor="port" className="form__label">
                                    {i18next.t('encryption_https')}
                                </label>

                                <Field
                                    name="port"
                                    type="number"
                                    component={renderInputField}
                                    className="form-control"
                                    placeholder={i18next.t('encryption_https')}
                                    validate={[validatePort, validateIsSafePort]}
                                    normalize={toNumber}
                                />
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

                    <Field
                        name="clientId"
                        type="text"
                        component={renderInputField}
                        className="form-control"
                        placeholder={i18next.t('client_id_placeholder')}
                        validate={validateConfigClientId}
                    />
                </div>

                <div className="form__group form__group--settings">
                    <label htmlFor="protocol" className="form__label">
                        {i18next.t('protocol')}
                    </label>

                    <Field name="protocol" type="text" component={renderSelectField} className="form-control">
                        <option value={MOBILE_CONFIG_LINKS.DOT}>{i18next.t('dns_over_tls')}</option>

                        <option value={MOBILE_CONFIG_LINKS.DOH}>{i18next.t('dns_over_https')}</option>
                    </Field>
                </div>
            </div>

            {getDownloadLink(getHostName(), clientId, protocol, invalid)}
        </form>
    );
};

export default reduxForm({ form: FORM_NAME.MOBILE_CONFIG })(MobileConfigForm);
