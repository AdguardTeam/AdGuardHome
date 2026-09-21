import { createSignal, createEffect, Show, onMount, onCleanup } from 'solid-js';
import cn from 'clsx';
import { useSearchParams } from '@solidjs/router';

import { SettingRow } from 'panel/common/ui/SettingRow';
import { DangerLink } from 'panel/common/ui/DangerLink';
import { Icon } from 'panel/common/ui/Icon';
import { PageLoader } from 'panel/common/ui/Loader';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { getTlsStatus, encryptionState, setTlsConfig } from 'panel/stores/encryption';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';
import { TLS_WIZARD_QUERY_KEY } from 'panel/components/Routes/Paths';

import { createDebouncedValidator } from './blocks/helpers';
import { PlainDnsToggle } from './blocks/PlainDnsToggle';
import { TlsCertSection } from './blocks/TlsCertSection';
import { ServerSettingsRow } from './blocks/ServerSettingsRow';
import { RedirectToggle } from './blocks/RedirectToggle';
import { ResetDnsModal } from './blocks/ResetDnsModal';
import { ServerSettingsModal } from './blocks/ServerSettingsModal';
import { TlsSetupWizard } from './blocks/SetupWizard';
import s from './styles.module.pcss';

export const Encryption = () => {
    const [resetOpen, setResetOpen] = createSignal(false);
    const [serverSettingsOpen, setServerSettingsOpen] = createSignal(false);
    const [addCertOpen, setAddCertOpen] = createSignal(false);

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

    const [searchParams, setSearchParams] = useSearchParams<{
        [TLS_WIZARD_QUERY_KEY]?: string;
    }>();

    createEffect(() => {
        if (!tlsStatusLoaded() || !searchParams[TLS_WIZARD_QUERY_KEY]) return;

        setAddCertOpen(true);
        setSearchParams({ [TLS_WIZARD_QUERY_KEY]: undefined }, { replace: true });
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
                    force_https: false,
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
            setTlsConfig({
                enabled: true,
            });
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
     * Centralised debounced validation trigger: fires a backend validation
     * whenever encryption is enabled and cert/key values are present.
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
            port_https: Number(encryptionState.port_https) || 0,
            port_dns_over_tls: Number(encryptionState.port_dns_over_tls) || 0,
            port_dns_over_quic: Number(encryptionState.port_dns_over_quic) || 0,
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

    return (
        <div class={theme.layout.container}>
            <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <div class={s.header}>
                    <h1 class={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                        {intl.getMessage('protocols')}
                    </h1>
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
                        <SettingRow
                            id="tls_cert_setup"
                            variant="link"
                            prefixIcon={<Icon icon="plus" color="green" />}
                            titleLink
                            hideArrow
                            title={intl.getMessage('tls_setup_row_title')}
                            description={intl.getMessage('tls_setup_row_description')}
                            onClick={() => setAddCertOpen(true)}
                            rowClass={s.setupRow}
                        />
                    </Show>

                    <Show when={certConfigured()}>
                        <TlsCertSection />
                    </Show>

                    <ServerSettingsRow onOpen={() => setServerSettingsOpen(true)} />

                    <RedirectToggle />

                    <DangerLink onClick={() => setResetOpen(true)}>
                        {intl.getMessage('reset_dns_protocols')}
                    </DangerLink>
                </Show>
            </div>

            <ResetDnsModal open={resetOpen()} onClose={() => setResetOpen(false)} />

            <ServerSettingsModal
                open={serverSettingsOpen()}
                onClose={() => setServerSettingsOpen(false)}
            />

            <TlsSetupWizard open={addCertOpen()} onClose={() => setAddCertOpen(false)} />
        </div>
    );
};
