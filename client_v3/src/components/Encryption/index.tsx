import { createMemo, Show, onCleanup } from 'solid-js';
import cn from 'clsx';

import { DEBOUNCE_TIMEOUT, ENCRYPTION_SOURCE } from 'panel/helpers/constants';
import { PageLoader } from 'panel/common/ui/Loader';
import { setTlsConfig, validateTlsConfig, encryptionState } from 'panel/stores/encryption';
import theme from 'panel/lib/theme';
import intl from 'panel/common/intl';

import { type EncryptionFormValues, Form } from './Form';
import s from './styles.module.pcss';

export const Encryption = () => {
    const initialValues = createMemo((): EncryptionFormValues => {
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
        } = encryptionState;
        const certificate_source = certificate_chain
            ? ENCRYPTION_SOURCE.CONTENT
            : ENCRYPTION_SOURCE.PATH;
        const key_source =
            private_key || private_key_saved ? ENCRYPTION_SOURCE.CONTENT : ENCRYPTION_SOURCE.PATH;

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
    });

    const getSubmitValues = (values: any) => {
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
    };

    const handleFormSubmit = (values: any) => {
        const submitValues = getSubmitValues(values);
        setTlsConfig(submitValues);
    };

    let debounceTimer: ReturnType<typeof setTimeout> | null = null;
    const debouncedConfigValidation = (values: any) => {
        if (debounceTimer) {
            clearTimeout(debounceTimer);
        }
        debounceTimer = setTimeout(() => {
            const submitValues = getSubmitValues(values);
            if (submitValues.enabled) {
                validateTlsConfig(submitValues);
            }
        }, DEBOUNCE_TIMEOUT);
    };

    onCleanup(() => {
        if (debounceTimer) {
            clearTimeout(debounceTimer);
        }
    });

    return (
        <div class={theme.layout.container}>
            <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <h1
                    class={cn(
                        theme.layout.title,
                        theme.title.h4,
                        theme.title.h3_tablet,
                        s.title,
                    )}
                >
                    {intl.getMessage('encryption_title')}
                </h1>

                <Show when={!encryptionState.processing} fallback={<PageLoader />}>
                    <Form
                        initialValues={initialValues()}
                        onSubmit={handleFormSubmit}
                        debouncedConfigValidation={debouncedConfigValidation}
                        encryption={encryptionState}
                    />
                </Show>
            </div>
        </div>
    );
};
