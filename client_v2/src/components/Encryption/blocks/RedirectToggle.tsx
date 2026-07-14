import { SettingRow } from 'panel/common/ui/SettingRow';
import intl from 'panel/common/intl';
import { encryptionState, setTlsConfig } from 'panel/stores/encryption';

export const RedirectToggle = () => {
    const enc = () => encryptionState;

    const fullyConfigured = () => {
        const hasCert = !!(enc().certificate_chain || enc().certificate_path);
        const hasKey = !!(enc().private_key || enc().private_key_path || enc().private_key_saved);
        return enc().enabled && hasCert && hasKey;
    };

    const onChange = (checked: boolean) => {
        setTlsConfig({ force_https: checked });
    };

    return (
        <SettingRow
            id="force_https"
            variant="switch"
            title={intl.getMessage('encryption_force_redirect')}
            checked={enc().force_https}
            disabled={!fullyConfigured()}
            onChange={onChange}
        />
    );
};
