import { createSignal, createEffect, Show } from 'solid-js';
import { SettingRow } from 'panel/common/ui/SettingRow';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import intl from 'panel/common/intl';
import { encryptionState, setTlsConfig } from 'panel/stores/encryption';

export const PlainDnsToggle = () => {
    const [confirmDisable, setConfirmDisable] = createSignal(false);
    const [localChecked, setLocalChecked] = createSignal(encryptionState.serve_plain_dns, {
        equals: false,
    });
    const enc = () => encryptionState;

    // Sync localChecked from store when the dialog is not open,
    // so external changes (e.g. reset, removeCert) are reflected.
    createEffect(() => {
        if (!confirmDisable()) {
            setLocalChecked(enc().serve_plain_dns);
        }
    });

    const fullyConfigured = () => {
        const hasCert = !!(enc().certificate_chain || enc().certificate_path);
        const hasKey = !!(enc().private_key || enc().private_key_path || enc().private_key_saved);
        return enc().enabled && hasCert && hasKey;
    };

    const locked = () => !fullyConfigured();

    const onChange = (checked: boolean) => {
        if (locked()) return;
        if (checked) {
            setTlsConfig({ serve_plain_dns: true });
            return;
        }
        // Force DOM re-sync before opening the modal.  With equals:false
        // this re-applies checked={true} to the native input.
        setLocalChecked((v) => v);
        setConfirmDisable(true);
    };

    return (
        <>
            <SettingRow
                id="serve_plain_dns"
                variant="switch"
                title={intl.getMessage('encryption_plain_dns')}
                description={intl.getMessage('encryption_plain_dns_desc')}
                checked={localChecked()}
                disabled={locked()}
                onChange={onChange}
            />
            <Show when={confirmDisable()}>
                <ConfirmDialog
                    title={intl.getMessage('encryption_disable_plain_dns')}
                    text={intl.getMessage('encryption_disable_plain_dns_desc')}
                    buttonText={intl.getMessage('yes_disable')}
                    cancelText={intl.getMessage('cancel')}
                    buttonVariant="danger"
                    onClose={() => setConfirmDisable(false)}
                    onConfirm={() => {
                        setTlsConfig({ serve_plain_dns: false });
                        setConfirmDisable(false);
                    }}
                />
            </Show>
        </>
    );
};
