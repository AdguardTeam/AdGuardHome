import { createEffect, createSignal, on } from 'solid-js';
import { createStore } from 'solid-js/store';

import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import intl from 'panel/common/intl';
import { normalizeServerName, toNumber } from 'panel/helpers/form';
import { validateServerName } from 'panel/helpers/validators';
import { encryptionState, setTlsConfig } from 'panel/stores/encryption';
import {
    validatePortField,
    validateServerSettings,
    type ServerSettingsField,
    type ServerSettingsValues,
} from '../validate';
import { getExternalPorts } from './helpers';
import { ServerSettingsFields } from './ServerSettingsFields';

type Props = {
    open: boolean;
    onClose: () => void;
};

const defaults: ServerSettingsValues = {
    server_name: '',
    port_https: 0,
    port_dns_over_tls: 0,
    port_dns_over_quic: 0,
};

export const ServerSettingsModal = (props: Props) => {
    const [values, setValues] = createStore<ServerSettingsValues>({ ...defaults });
    const [errors, setErrors] = createSignal<Record<string, string>>({});

    createEffect(
        on(
            () => props.open,
            (open) => {
                if (!open) return;

                setValues({
                    server_name: encryptionState.server_name || '',
                    port_https: Number(encryptionState.port_https) || 0,
                    port_dns_over_tls: Number(encryptionState.port_dns_over_tls) || 0,
                    port_dns_over_quic: Number(encryptionState.port_dns_over_quic) || 0,
                });
                setErrors({});
            },
        ),
    );

    /** Errors that follow from the current values, e.g. a port conflict. */
    const clientErrors = () => validateServerSettings(values, getExternalPorts());

    /**
     * Error to render under a field: blur/submit errors win, then the live
     * client-side rules.  Only the ports have live rules — the server name is
     * validated on blur and by the backend, so typing does not flash an error
     * mid-word.
     */
    const fieldError = (field: ServerSettingsField) =>
        errors()[field] ?? (field === 'server_name' ? undefined : clientErrors()[field]);

    const setFieldError = (field: ServerSettingsField, message?: string) => {
        setErrors((prev) => {
            const next = { ...prev };
            if (message) {
                next[field] = message;
            } else {
                delete next[field];
            }
            return next;
        });
    };

    const handleFieldChange = (field: ServerSettingsField, raw: string) => {
        setFieldError(field);
        if (field === 'server_name') {
            setValues('server_name', raw);
            return;
        }
        setValues(field, toNumber(raw));
    };

    const handleFieldBlur = (field: ServerSettingsField) => {
        if (field === 'server_name') {
            const normalized = normalizeServerName(String(values.server_name ?? ''));
            setValues('server_name', normalized);
            setFieldError(field, validateServerName(normalized));
            return;
        }

        const port = Number(values[field]) || 0;
        setFieldError(field, validatePortField(field, port));
    };

    const save = () => {
        const errs = clientErrors();
        if (Object.values(errs).some(Boolean)) {
            setErrors(errs);
            return;
        }

        setTlsConfig({ ...values });
        props.onClose();
    };

    const processing = () => encryptionState.processingConfig || encryptionState.processingValidate;
    const hasErrors = () =>
        Object.values(errors()).some(Boolean) || Object.values(clientErrors()).some(Boolean);

    return (
        <ConfigDialog
            open={props.open}
            title={intl.getMessage('encrypted_dns_settings')}
            onClose={props.onClose}
            onSubmit={save}
            processing={processing()}
            submitDisabled={processing() || hasErrors()}
        >
            <ServerSettingsFields
                values={values}
                onFieldChange={handleFieldChange}
                onFieldBlur={handleFieldBlur}
                errorFor={fieldError}
                clearable
            />
        </ConfigDialog>
    );
};
