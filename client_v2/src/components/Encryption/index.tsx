import React, { useCallback, useMemo } from 'react';
import { debounce } from 'lodash';
import { useDispatch, useSelector } from 'react-redux';
import cn from 'clsx';

import { DEBOUNCE_TIMEOUT, ENCRYPTION_SOURCE } from 'panel/helpers/constants';
import { RootState } from 'panel/initialState';
import { PageLoader } from 'panel/common/ui/Loader';
import { setTlsConfig, validateTlsConfig } from 'panel/actions/encryption';
import theme from 'panel/lib/theme';
import intl from 'panel/common/intl';

import { EncryptionFormValues, Form } from './Form';
import s from './styles.module.pcss';

export const Encryption = () => {
    const dispatch = useDispatch();
    const encryption = useSelector((state: RootState) => state.encryption);

    const initialValues = useMemo((): EncryptionFormValues => {
        const {
            enabled,
            serve_plain_dns,
            server_name,
            force_https,
            port_https,
            port_dns_over_tls,
            port_dns_over_quic,
            certificate_chain,
            private_key,
            certificate_path,
            private_key_path,
            private_key_saved,
        } = encryption;
        const certificate_source = certificate_chain ? ENCRYPTION_SOURCE.CONTENT : ENCRYPTION_SOURCE.PATH;
        const key_source = private_key || private_key_saved ? ENCRYPTION_SOURCE.CONTENT : ENCRYPTION_SOURCE.PATH;

        return {
            enabled,
            serve_plain_dns,
            server_name,
            force_https,
            port_https,
            port_dns_over_tls,
            port_dns_over_quic,
            certificate_chain,
            private_key,
            certificate_path,
            private_key_path,
            private_key_saved,
            certificate_source,
            key_source,
        };
    }, [encryption]);

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
            dispatch(setTlsConfig(submitValues));
        },
        [getSubmitValues, setTlsConfig],
    );

    const validateConfig = useCallback((values) => {
        const submitValues = getSubmitValues(values);

        if (submitValues.enabled) {
            dispatch(validateTlsConfig(submitValues));
        }
    }, []);

    const debouncedConfigValidation = useMemo(() => debounce(validateConfig, DEBOUNCE_TIMEOUT), [validateConfig]);

    return (
        <div className={theme.layout.container}>
            <div className={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <h1 className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet, s.title)}>
                    {intl.getMessage('encryption_title')}
                </h1>

                {encryption.processing ? (
                    <PageLoader />
                ) : (
                    <Form
                        initialValues={initialValues}
                        onSubmit={handleFormSubmit}
                        debouncedConfigValidation={debouncedConfigValidation}
                        encryption={encryption}
                    />
                )}
            </div>
        </div>
    );
};
