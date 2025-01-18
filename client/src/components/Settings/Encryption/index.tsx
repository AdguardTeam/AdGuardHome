import React, { useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { debounce } from 'lodash';
import { DEBOUNCE_TIMEOUT, ENCRYPTION_SOURCE } from '../../../helpers/constants';

import { Form, FormValues } from './Form';
import Card from '../../ui/Card';
import PageTitle from '../../ui/PageTitle';
import Loading from '../../ui/Loading';
import { EncryptionData } from '../../../initialState';

type Props = {
    encryption: EncryptionData;
    setTlsConfig: (values: EncryptionData) => void;
    validateTlsConfig: (values: EncryptionData) => void;
};

export const Encryption = ({ encryption, setTlsConfig, validateTlsConfig }: Props) => {
    const { t } = useTranslation();

    useEffect(() => {
        if (encryption.enabled) {
            validateTlsConfig(encryption);
        }
    }, [encryption, validateTlsConfig]);

    const getInitialValues = useCallback((data: any): FormValues => {
        const { certificate_chain, private_key, private_key_saved } = data;
        const certificate_source = certificate_chain ? ENCRYPTION_SOURCE.CONTENT : ENCRYPTION_SOURCE.PATH;
        const key_source = private_key || private_key_saved ? ENCRYPTION_SOURCE.CONTENT : ENCRYPTION_SOURCE.PATH;

        return {
            ...data,
            certificate_source,
            key_source,
        };
    }, []);

    const getSubmitValues = useCallback((values: any) => {
        const { certificate_source, key_source, private_key_saved, ...config } = values;

        if (certificate_source === ENCRYPTION_SOURCE.PATH) {
            config.certificate_chain = '';
        } else {
            config.certificate_path = '';
        }

        if (key_source === ENCRYPTION_SOURCE.PATH) {
            config.private_key = '';
        } else {
            config.private_key_path = '';

            if (private_key_saved) {
                config.private_key = '';
                config.private_key_saved = private_key_saved;
            }
        }

        return config;
    }, []);

    const handleFormSubmit = useCallback(
        (values: any) => {
            const submitValues = getSubmitValues(values);
            setTlsConfig(submitValues);
        },
        [getSubmitValues, setTlsConfig],
    );

    const handleChange = useCallback(
        debounce((values) => {
            const submitValues = getSubmitValues(values);

            if (submitValues.enabled) {
                validateTlsConfig(submitValues);
            }
        }, DEBOUNCE_TIMEOUT),
        [getSubmitValues, validateTlsConfig],
    );

    const initialValues = getInitialValues(encryption);

    return (
        <div className="encryption">
            <PageTitle title={t('encryption_settings')} />

            {encryption.processing ? (
                <Loading />
            ) : (
                <Card
                    title={t('encryption_title')}
                    subtitle={t('encryption_desc')}
                    bodyType="card-body box-body--settings">
                    <Form
                        initialValues={initialValues}
                        onSubmit={handleFormSubmit}
                        onChange={handleChange}
                        setTlsConfig={setTlsConfig}
                        validateTlsConfig={validateTlsConfig}
                        {...encryption}
                    />
                </Card>
            )}
        </div>
    );
};
