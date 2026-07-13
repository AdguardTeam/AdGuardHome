import { createMemo } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { dnsConfigState, togglePrivatePtrResolvers } from 'panel/stores/dnsConfig';
import { settingsState, testUpstreamWithFormValues } from 'panel/stores/settings';
import { Button } from 'panel/common/ui/Button';
import { SettingRow } from 'panel/common/ui/SettingRow';
import { useDialog } from 'panel/hooks/useDialog';
import { getUpstreamModeSummary, getUpstreamServersSummary, getTtlSummary } from '../helpers';
import theme from 'panel/lib/theme';

import { UpstreamModeDialog } from './blocks/UpstreamModeDialog';
import { ServerAddressesDialog } from './blocks/ServerAddressesDialog';
import { FallbackDnsDialog } from './blocks/FallbackDnsDialog';
import { BootstrapDnsDialog } from './blocks/BootstrapDnsDialog';
import { TimeoutDialog } from './blocks/TimeoutDialog';
import { Paths } from 'panel/components/Routes/Paths';

export const Upstream = () => {
    const navigate = useNavigate();

    const upstreamModeDialog = useDialog();
    const serverAddressesDialog = useDialog();
    const fallbackDialog = useDialog();
    const bootstrapDialog = useDialog();
    const timeoutDialog = useDialog();

    const upstreamModeValue = createMemo(() =>
        getUpstreamModeSummary(dnsConfigState.upstream_mode),
    );
    const serverAddressesValue = createMemo(() =>
        getUpstreamServersSummary(dnsConfigState.upstream_dns, dnsConfigState.upstream_dns_file),
    );
    const fallbackServersValue = createMemo(() =>
        getUpstreamServersSummary(dnsConfigState.fallback_dns),
    );
    const bootstrapServersValue = createMemo(() =>
        getUpstreamServersSummary(dnsConfigState.bootstrap_dns),
    );
    const timeoutValue = createMemo(() => getTtlSummary(dnsConfigState.upstream_timeout));

    const processing = () => dnsConfigState.processingSetConfig;

    return (
        <div>
            <h2 class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('upstream_dns')}
            </h2>

            <SettingRow
                variant="link"
                id="upstream_mode"
                title={intl.getMessage('dns_upstream_mode')}
                description={intl.getMessage('dns_upstream_mode_desc')}
                value={upstreamModeValue()}
                onClick={upstreamModeDialog.openDialog}
            />

            <SettingRow
                variant="link"
                id="server_addresses"
                title={intl.getMessage('dns_server_addresses')}
                description={intl.getMessage('dns_server_addresses_desc')}
                value={serverAddressesValue()}
                onClick={serverAddressesDialog.openDialog}
            />

            <SettingRow
                variant="link"
                id="fallback_servers"
                title={intl.getMessage('dns_fallback_servers')}
                description={intl.getMessage('dns_fallback_dns_desc')}
                value={fallbackServersValue()}
                onClick={fallbackDialog.openDialog}
            />

            <SettingRow
                variant="link"
                id="bootstrap_servers"
                title={intl.getMessage('dns_bootstrap_servers')}
                description={intl.getMessage('dns_bootstrap_dns_desc')}
                value={bootstrapServersValue()}
                onClick={bootstrapDialog.openDialog}
            />

            <SettingRow
                variant="switch-link"
                id="private_reverse"
                title={intl.getMessage('dns_private_reverse_resolvers')}
                description={
                    <>
                        <p>{intl.getMessage('dns_private_reverse_resolvers_desc')}</p>
                        <p>{intl.getMessage('dns_private_reverse_resolvers_disabled_desc')}</p>
                    </>
                }
                checked={dnsConfigState.use_private_ptr_resolvers}
                onChange={() => togglePrivatePtrResolvers()}
                onClick={() => navigate(Paths.DnsPrivateReverse)}
                divider
            />

            <SettingRow
                variant="link"
                id="upstream_timeout"
                title={intl.getMessage('dns_upstream_timeout')}
                description={intl.getMessage('dns_upstream_timeout_desc')}
                value={timeoutValue()}
                onClick={timeoutDialog.openDialog}
            />

            <div class={theme.form.actionRow}>
                <Button
                    variant="secondary"
                    disabled={settingsState.processingTestUpstream}
                    onClick={() =>
                        testUpstreamWithFormValues(
                            {
                                bootstrap_dns: dnsConfigState.bootstrap_dns,
                                upstream_dns: dnsConfigState.upstream_dns,
                                local_ptr_upstreams: dnsConfigState.local_ptr_upstreams,
                                fallback_dns: dnsConfigState.fallback_dns,
                            },
                            dnsConfigState.upstream_dns_file,
                        )
                    }
                    class={theme.form.actionButton}
                    compact
                >
                    {intl.getMessage('dns_test_upstreams')}
                </Button>
            </div>

            <UpstreamModeDialog
                open={upstreamModeDialog.open}
                onClose={upstreamModeDialog.closeDialog}
                processing={processing()}
            />
            <ServerAddressesDialog
                open={serverAddressesDialog.open}
                onClose={serverAddressesDialog.closeDialog}
                processing={processing()}
            />
            <FallbackDnsDialog
                open={fallbackDialog.open}
                onClose={fallbackDialog.closeDialog}
                processing={processing()}
            />
            <BootstrapDnsDialog
                open={bootstrapDialog.open}
                onClose={bootstrapDialog.closeDialog}
                processing={processing()}
            />
            <TimeoutDialog
                open={timeoutDialog.open}
                onClose={timeoutDialog.closeDialog}
                processing={processing()}
            />
        </div>
    );
};
