import { createMemo } from 'solid-js';
import cn from 'clsx';

import {
    dnsConfigState,
    toggleDnssecEnabled,
    toggleDisableIPv6,
    toggleEdnsCsEnabled,
} from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { SettingRow } from 'panel/common/ui/SettingRow';
import { getRateLimitSummary, getBlockingModeSummary, getUpstreamServersSummary } from '../helpers';
import theme from 'panel/lib/theme';
import { IP_VERSION, IPV4_SUBNET_PREFIX, IPV6_SUBNET_PREFIX } from 'panel/helpers/constants';
import { useDialog } from 'panel/hooks/useDialog';

import { RateLimitDialog } from './blocks/RateLimitDialog';
import { SubnetPrefixV4Dialog } from './blocks/SubnetPrefixV4Dialog';
import { SubnetPrefixV6Dialog } from './blocks/SubnetPrefixV6Dialog';
import { BlockingModeDialog } from './blocks/BlockingModeDialog';
import { RateLimitAllowlistDialog } from './blocks/RateLimitAllowlistDialog';
import { EdnsDialog } from './blocks/EdnsDialog';

export const ServerConfig = () => {
    const rateLimitDialog = useDialog();
    const subnetV4Dialog = useDialog();
    const subnetV6Dialog = useDialog();
    const blockingModeDialog = useDialog();
    const allowlistDialog = useDialog();
    const ednsDialog = useDialog();

    const rateLimitValue = createMemo(() => getRateLimitSummary(dnsConfigState.ratelimit));
    const blockingModeValue = createMemo(() =>
        getBlockingModeSummary(dnsConfigState.blocking_mode),
    );

    const processing = () => dnsConfigState.processingSetConfig;

    return (
        <div>
            <h2 class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('dns_config')}
            </h2>

            <SettingRow
                variant="link"
                id="rate_limit"
                title={intl.getMessage('dns_rate_limit')}
                description={intl.getMessage('dns_rate_limit_desc')}
                value={rateLimitValue()}
                onClick={rateLimitDialog.openDialog}
            />

            <SettingRow
                variant="link"
                id="ratelimit_allowlist"
                title={intl.getMessage('dns_rate_limit_allowlist')}
                description={intl.getMessage('dns_rate_limit_allowlist_desc')}
                value={getUpstreamServersSummary(dnsConfigState.ratelimit_whitelist)}
                onClick={allowlistDialog.openDialog}
            />

            <SettingRow
                variant="link"
                id="subnet_ipv4"
                title={intl.getMessage('dns_subnet_prefix', { value: IP_VERSION.V4 })}
                description={intl.getMessage('dns_subnet_prefix_desc', { value: IP_VERSION.V4 })}
                value={String(
                    dnsConfigState.ratelimit_subnet_len_ipv4 ?? IPV4_SUBNET_PREFIX.DEFAULT,
                )}
                onClick={subnetV4Dialog.openDialog}
            />

            <SettingRow
                variant="link"
                id="subnet_ipv6"
                title={intl.getMessage('dns_subnet_prefix', { value: IP_VERSION.V6 })}
                description={intl.getMessage('dns_subnet_prefix_desc', { value: IP_VERSION.V6 })}
                value={String(
                    dnsConfigState.ratelimit_subnet_len_ipv6 ?? IPV6_SUBNET_PREFIX.DEFAULT,
                )}
                onClick={subnetV6Dialog.openDialog}
            />

            <SettingRow
                variant="link"
                id="blocking_mode"
                title={intl.getMessage('dns_blocking_mode')}
                description={intl.getMessage('dns_blocking_mode_desc')}
                value={blockingModeValue()}
                onClick={blockingModeDialog.openDialog}
            />

            <SettingRow
                variant="switch-link"
                id="edns_client_subnet"
                title={intl.getMessage('dns_edns_client_subnet')}
                description={intl.getMessage('dns_edns_client_subnet_desc')}
                checked={dnsConfigState.edns_cs_enabled}
                onChange={() => toggleEdnsCsEnabled()}
                onClick={ednsDialog.openDialog}
                divider
            />

            <SettingRow
                variant="switch"
                id="dnssec"
                title={intl.getMessage('dns_dnssec')}
                description={intl.getMessage('dns_dnssec_desc')}
                checked={dnsConfigState.dnssec_enabled}
                onChange={() => toggleDnssecEnabled()}
            />

            <SettingRow
                variant="switch"
                id="ipv6_resolution"
                title={intl.getMessage('dns_ipv6_resolution')}
                description={intl.getMessage('dns_ipv6_resolution_desc')}
                checked={!dnsConfigState.disable_ipv6}
                onChange={() => toggleDisableIPv6()}
            />

            <RateLimitDialog
                open={rateLimitDialog.open}
                onClose={rateLimitDialog.closeDialog}
                processing={processing()}
            />

            <SubnetPrefixV4Dialog
                open={subnetV4Dialog.open}
                onClose={subnetV4Dialog.closeDialog}
                processing={processing()}
            />

            <SubnetPrefixV6Dialog
                open={subnetV6Dialog.open}
                onClose={subnetV6Dialog.closeDialog}
                processing={processing()}
            />

            <BlockingModeDialog
                open={blockingModeDialog.open}
                onClose={blockingModeDialog.closeDialog}
                processing={processing()}
            />

            <RateLimitAllowlistDialog
                open={allowlistDialog.open}
                onClose={allowlistDialog.closeDialog}
                processing={processing()}
            />

            <EdnsDialog
                open={ednsDialog.open}
                onClose={ednsDialog.closeDialog}
                processing={processing()}
            />
        </div>
    );
};
