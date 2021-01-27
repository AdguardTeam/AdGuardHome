import React from 'react';
import PropTypes from 'prop-types';
import { Trans } from 'react-i18next';
import { useSelector } from 'react-redux';
import { Field, reduxForm } from 'redux-form';
import i18next from 'i18next';
import cn from 'classnames';

import { getPathWithQueryString } from '../../../helpers/helpers';
import { FORM_NAME, MOBILE_CONFIG_LINKS } from '../../../helpers/constants';
import {
    renderInputField,
    renderSelectField,
} from '../../../helpers/form';
import {
    validateClientId,
    validateServerName,
} from '../../../helpers/validators';

const getDownloadLink = (host, clientId, protocol, invalid) => {
    if (!host || invalid) {
        return (
            <button
                type="button"
                className="btn btn-success btn-standard btn-large disabled"
            >
                <Trans>download_mobileconfig</Trans>
            </button>
        );
    }

    const linkParams = { host };

    if (clientId) {
        linkParams.client_id = clientId;
    }

    return (
        <a
            href={getPathWithQueryString(protocol, linkParams)}
            className={cn('btn btn-success btn-standard btn-large')}
            download
        >
            <Trans>download_mobileconfig</Trans>
        </a>
    );
};

const MobileConfigForm = ({ invalid }) => {
    const formValues = useSelector((state) => state.form[FORM_NAME.MOBILE_CONFIG]?.values);

    if (!formValues) {
        return null;
    }

    const { host, clientId, protocol } = formValues;

    const githubLink = (
        <a
            href="https://github.com/AdguardTeam/AdGuardHome/wiki/Clients#idclient"
            target="_blank"
            rel="noopener noreferrer"
        >
            text
        </a>
    );

    return (
        <form onSubmit={(e) => e.preventDefault()}>
            <div>
                <div className="form__group form__group--settings">
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
                <div className="form__group form__group--settings">
                    <label htmlFor="clientId" className="form__label form__label--with-desc">
                        {i18next.t('client_id')}
                    </label>
                    <div className="form__desc form__desc--top">
                        <Trans components={{ a: githubLink }}>
                            client_id_desc
                        </Trans>
                    </div>
                    <Field
                        name="clientId"
                        type="text"
                        component={renderInputField}
                        className="form-control"
                        placeholder={i18next.t('client_id_placeholder')}
                        validate={validateClientId}
                    />
                </div>
                <div className="form__group form__group--settings">
                    <label htmlFor="protocol" className="form__label">
                        {i18next.t('protocol')}
                    </label>
                    <Field
                        name="protocol"
                        type="text"
                        component={renderSelectField}
                        className="form-control"
                    >
                        <option value={MOBILE_CONFIG_LINKS.DOT}>
                            {i18next.t('dns_over_tls')}
                        </option>
                        <option value={MOBILE_CONFIG_LINKS.DOH}>
                            {i18next.t('dns_over_https')}
                        </option>
                    </Field>
                </div>
            </div>

            {getDownloadLink(host, clientId, protocol, invalid)}
        </form>
    );
};

MobileConfigForm.propTypes = {
    invalid: PropTypes.bool.isRequired,
};

export default reduxForm({ form: FORM_NAME.MOBILE_CONFIG })(MobileConfigForm);
