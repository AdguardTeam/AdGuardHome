import { Show, createSignal } from 'solid-js';
import { Icon } from 'panel/common/ui/Icon';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import intl from 'panel/common/intl';
import {
    encryptionState,
    setTlsConfig,
    resetValidationStatus,
    applyTlsOptimistically,
} from 'panel/stores/encryption';
import { CertificateStatus, ValidationStatus } from '../Status';
import { defaultTlsValues, getSubmitValues } from './helpers';
import s from '../styles.module.pcss';
import theme from 'panel/lib/theme';

export const TlsCertSection = () => {
    const [showDeleteConfirm, setShowDeleteConfirm] = createSignal(false);

    const enc = () => encryptionState;

    /**
     * Removing the certificate takes the whole encryption setup down with it:
     * encryption is turned off and the TLS data goes back to its defaults, the
     * same payload "Reset DNS protocols" writes.  The server settings row is
     * disabled without a pair but keeps rendering what it knows, so a partial
     * reset would leave the hostname, the ports, and the redirect behind —
     * describing a server that no longer serves TLS, and seeding the next
     * wizard run with those values.
     */
    const handleRemoveCert = () => {
        const values = getSubmitValues(defaultTlsValues);

        applyTlsOptimistically(values);
        resetValidationStatus();
        setTlsConfig(values);
        setShowDeleteConfirm(false);
    };

    const renderStatus = () => {
        if (enc().valid_cert && enc().valid_key && !enc().valid_pair) {
            return (
                <ValidationStatus
                    type="error"
                    message={intl.getMessage('encryption_key_cert_mismatch')}
                />
            );
        }
        if (enc().warning_validation) {
            const isWarning = enc().valid_key && enc().valid_cert && enc().valid_pair;
            return (
                <ValidationStatus
                    type={isWarning ? 'warning' : 'error'}
                    message={enc().warning_validation}
                />
            );
        }
        if (!enc().certificate_chain && !enc().certificate_path) return null;
        return (
            <CertificateStatus
                validChain={enc().valid_chain}
                validCert={enc().valid_cert}
                subject={enc().subject}
                issuer={enc().issuer}
                notAfter={enc().not_after}
                dnsNames={enc().dns_names}
                keyType={enc().valid_key ? enc().key_type : undefined}
            />
        );
    };

    return (
        <div class={s.certSection} data-testid="tls-cert-section">
            <div class={s.certRow}>
                <span class={s.certTitle}>{intl.getMessage('tls_certificate')}</span>
                <button
                    type="button"
                    class={theme.form.action}
                    onClick={() => setShowDeleteConfirm(true)}
                    aria-label={intl.getMessage('encryption_certificates')}
                    data-testid="tls-cert-remove"
                >
                    <Icon icon="delete" color="red" />
                </button>
            </div>
            {renderStatus()}

            <Show when={showDeleteConfirm()}>
                <ConfirmDialog
                    title={intl.getMessage('remove_tls_certificate')}
                    text={intl.getMessage('remove_tls_certificate_desc')}
                    buttonText={intl.getMessage('yes_remove')}
                    cancelText={intl.getMessage('cancel')}
                    buttonVariant="danger"
                    onConfirm={handleRemoveCert}
                    onClose={() => setShowDeleteConfirm(false)}
                />
            </Show>
        </div>
    );
};
