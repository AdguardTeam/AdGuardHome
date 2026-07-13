import { Show } from 'solid-js';
import { Icon } from 'panel/common/ui/Icon';
import intl from 'panel/common/intl';
import {
    encryptionState,
    setTlsConfig,
    resetValidationStatus,
    clearCertOptimistically,
} from 'panel/stores/encryption';
import { CertificateStatus, KeyStatus, ValidationStatus } from '../Status';
import s from '../styles.module.pcss';
import theme from 'panel/lib/theme';

export const TlsCertSection = () => {
    const enc = () => encryptionState;

    const removeCert = () => {
        clearCertOptimistically();
        resetValidationStatus();
        setTlsConfig({
            enabled: false,
            serve_plain_dns: true,
            certificate_chain: '',
            private_key: '',
            certificate_path: '',
            private_key_path: '',
            private_key_saved: false,
        });
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
            <>
                <CertificateStatus
                    validChain={enc().valid_chain}
                    validCert={enc().valid_cert}
                    subject={enc().subject}
                    issuer={enc().issuer}
                    notAfter={enc().not_after}
                    dnsNames={enc().dns_names}
                />
                <Show when={enc().private_key || enc().private_key_path}>
                    <KeyStatus validKey={enc().valid_key} keyType={enc().key_type} />
                </Show>
            </>
        );
    };

    return (
        <div class={s.certSection} data-testid="tls-cert-section">
            <div class={s.certRow}>
                <span class={s.certTitle}>{intl.getMessage('tls_certificate')}</span>
                <button
                    type="button"
                    class={theme.form.action}
                    onClick={removeCert}
                    aria-label={intl.getMessage('encryption_certificates')}
                >
                    <Icon icon="delete" color="red" />
                </button>
            </div>
            {renderStatus()}
        </div>
    );
};
