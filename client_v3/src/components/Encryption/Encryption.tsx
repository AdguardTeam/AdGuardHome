import { createSignal, createEffect, Show, onMount, onCleanup } from 'solid-js';
import cn from 'clsx';

import { SettingRow } from 'panel/common/ui/SettingRow';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Icon } from 'panel/common/ui/Icon';
import { PageLoader } from 'panel/common/ui/Loader';
import { PlusButton } from 'panel/common/ui/PlusButton';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { getTlsStatus, encryptionState, setTlsConfig } from 'panel/stores/encryption';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';

import { createDebouncedValidator } from './blocks/helpers';
import { PlainDnsToggle } from './blocks/PlainDnsToggle';
import { TlsCertSection } from './blocks/TlsCertSection';
import { ServerSettingsRow } from './blocks/ServerSettingsRow';
import { RedirectToggle } from './blocks/RedirectToggle';
import { ResetDnsModal } from './blocks/ResetDnsModal';
import { ServerSettingsModal } from './blocks/ServerSettingsModal';
import { AddTlsCertModal } from './blocks/AddTlsCert';
import s from './styles.module.pcss';

export const Encryption = () => {
    const [resetOpen, setResetOpen] = createSignal(false);
    const [serverSettingsOpen, setServerSettingsOpen] = createSignal(false);
    const [addCertOpen, setAddCertOpen] = createSignal(false);
    const [menuOpen, setMenuOpen] = createSignal(false);

    const [tlsStatusLoaded, setTlsStatusLoaded] = createSignal(false);

    /**
     * Shadows encryptionState.enabled with {@code equals: false} so we can
     * force a DOM re-sync even when the value is unchanged (e.g. reverting
     * after a modal opens without saving).  Synced from store only when
     * {@code processingConfig} is false to avoid flashing during async save.
     */
    const [encryptionEnabled, setEncryptionEnabled] = createSignal(false, {
        equals: false,
    });

    createEffect(() => {
        if (!encryptionState.processingConfig) {
            setEncryptionEnabled(encryptionState.enabled);
        }
    });

    const [validateConfig, cancelValidation] = createDebouncedValidator();

    onMount(async () => {
        await getTlsStatus();
        setTlsStatusLoaded(true);
    });

    onCleanup(() => {
        cancelValidation();
    });

    const certConfigured = () =>
        !!(encryptionState.certificate_chain || encryptionState.certificate_path);

    const handleEncryptedDnsChange = (checked: boolean) => {
        if (!checked) {
            setEncryptionEnabled(false);
            setTlsConfig(
                {
                    enabled: false,
                    serve_plain_dns: true,
                },
                { silent: true },
            );
            return;
        }

        // Enabling: check if everything is configured before saving.
        const hasCert = !!(encryptionState.certificate_chain || encryptionState.certificate_path);
        const hasKey = !!(
            encryptionState.private_key ||
            encryptionState.private_key_path ||
            encryptionState.private_key_saved
        );
        const hasServerName = !!encryptionState.server_name;

        // Everything is set up — save the change.
        // Native input already shows ON from the click; sync effect
        // confirms on success or reverts on failure.
        if (hasCert && hasKey && hasServerName) {
            setTlsConfig(
                {
                    enabled: true,
                },
                { silent: true },
            );
            return;
        }

        // Not saving — force DOM back to unchecked so the switch
        // doesn't appear ON while encryption is actually OFF.
        setEncryptionEnabled(false);

        // Certificate or key is missing — open the TLS cert wizard (don't save yet).
        if (!hasCert || !hasKey) {
            setAddCertOpen(true);
            return;
        }

        // Cert and key are present, but server name isn't set — open server settings.
        if (!hasServerName) {
            setServerSettingsOpen(true);
        }
    };

    /**
     * Centralised debounced validation trigger.
     * Watches the encryption state and fires debounced backend validation
     * whenever encryption is enabled and cert/key values are present.
     * Replaces the createEffect that was previously inside Form.tsx.
     */
    createEffect(() => {
        if (!tlsStatusLoaded()) return;
        if (!encryptionState.enabled) return;
        const hasCert = !!(encryptionState.certificate_chain || encryptionState.certificate_path);
        const hasKey = !!(
            encryptionState.private_key ||
            encryptionState.private_key_path ||
            encryptionState.private_key_saved
        );
        if (!hasCert || !hasKey) return;

        validateConfig({
            enabled: encryptionState.enabled,
            serve_plain_dns: encryptionState.serve_plain_dns,
            server_name: encryptionState.server_name,
            force_https: encryptionState.force_https,
            port_https: encryptionState.port_https,
            port_dns_over_tls: encryptionState.port_dns_over_tls,
            port_dns_over_quic: encryptionState.port_dns_over_quic,
            certificate_chain: encryptionState.certificate_chain,
            private_key: encryptionState.private_key,
            certificate_path: encryptionState.certificate_path,
            private_key_path: encryptionState.private_key_path,
            certificate_source: encryptionState.certificate_chain
                ? ENCRYPTION_SOURCE.CONTENT
                : ENCRYPTION_SOURCE.PATH,
            key_source:
                encryptionState.private_key || encryptionState.private_key_saved
                    ? ENCRYPTION_SOURCE.CONTENT
                    : ENCRYPTION_SOURCE.PATH,
            private_key_saved: encryptionState.private_key_saved,
        });
    });

    const handleResetClick = () => {
        setMenuOpen(false);
        setResetOpen(true);
    };

    const resetMenu = (
        <div
            class={cn(theme.dropdown.item, theme.dropdown.item_danger, theme.dropdown.item_large)}
            onClick={handleResetClick}
        >
            {intl.getMessage('reset_dns_protocols')}
        </div>
    );

    return (
        <div class={theme.layout.container}>
            <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <div class={s.header}>
                    <h1 class={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                        {intl.getMessage('dns_protocols_title')}
                    </h1>
                    <Dropdown
                        trigger="click"
                        position="bottomRight"
                        noIcon
                        open={menuOpen()}
                        onOpenChange={setMenuOpen}
                        menu={resetMenu}
                    >
                        <button
                            type="button"
                            class={s.menuButton}
                            aria-label={intl.getMessage('reset_dns_protocols')}
                        >
                            <Icon icon="bullets" />
                        </button>
                    </Dropdown>
                </div>

                <Show when={tlsStatusLoaded()} fallback={<PageLoader />}>
                    <PlainDnsToggle />

                    <h2
                        class={cn(
                            theme.layout.subtitle,
                            theme.title.h5,
                            theme.title.h4_tablet,
                            s.section,
                        )}
                    >
                        {intl.getMessage('encryption_title')}
                    </h2>

                    <SettingRow
                        id="encrypted_dns"
                        variant="switch"
                        title={intl.getMessage('encryption_encrypted_dns')}
                        description={intl.getMessage('encryption_encrypted_dns_desc')}
                        checked={encryptionEnabled()}
                        disabled={encryptionState.processingConfig}
                        onChange={handleEncryptedDnsChange}
                    />

                    <Show when={!certConfigured()}>
                        <div class={s.plusButton}>
                            <PlusButton onClick={() => setAddCertOpen(true)} weight="semi">
                                {intl.getMessage('add_tls_certificate')}
                            </PlusButton>
                        </div>
                    </Show>

                    <Show when={certConfigured()}>
                        <TlsCertSection />
                    </Show>

                    <ServerSettingsRow onOpen={() => setServerSettingsOpen(true)} />

                    <RedirectToggle />
                </Show>
            </div>

            <ResetDnsModal open={resetOpen()} onClose={() => setResetOpen(false)} />

            <ServerSettingsModal
                open={serverSettingsOpen()}
                onClose={() => setServerSettingsOpen(false)}
            />

            <AddTlsCertModal open={addCertOpen()} onClose={() => setAddCertOpen(false)} />
        </div>
    );
};
