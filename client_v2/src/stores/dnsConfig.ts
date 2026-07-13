import { createStore, reconcile } from 'solid-js/store';
import { untrack } from 'solid-js';
import { apiClient } from 'panel/api/Api';
import { addErrorToast, addSuccessToast } from './toasts';
import intl from 'panel/common/intl';
import { splitByNewLine } from 'panel/helpers/helpers';
import { DNS_REQUEST_OPTIONS } from 'panel/helpers/constants';

type DnsConfigState = {
    processingGetConfig: boolean;
    processingSetConfig: boolean;
    blocking_mode: string;
    ratelimit: number;
    blocking_ipv4: string;
    blocking_ipv6: string;
    blocked_response_ttl: number;
    upstream_timeout: number;
    edns_cs_enabled: boolean;
    disable_ipv6: boolean;
    dnssec_enabled: boolean;
    upstream_dns_file: string;
    upstream_dns: string;
    fallback_dns: string;
    bootstrap_dns: string;
    local_ptr_upstreams: string;
    ratelimit_whitelist: string;
    upstream_mode: string;
    resolve_clients: boolean;
    use_private_ptr_resolvers: boolean;
    default_local_ptr_upstreams: string[];
    ratelimit_subnet_len_ipv4?: number;
    ratelimit_subnet_len_ipv6?: number;
    edns_cs_use_custom?: boolean;
    edns_cs_custom_ip?: string;
    cache_size?: number;
    cache_ttl_max?: number;
    cache_ttl_min?: number;
    cache_optimistic?: boolean;
    cache_enabled?: boolean;
};

export const DEFAULT_BLOCKING_IPV4 = '0.0.0.0';
export const DEFAULT_BLOCKING_IPV6 = '::';
const BLOCKING_MODES = { default: 'default' };

const initialState: DnsConfigState = {
    processingGetConfig: true,
    processingSetConfig: false,
    blocking_mode: BLOCKING_MODES.default,
    ratelimit: 20,
    blocking_ipv4: DEFAULT_BLOCKING_IPV4,
    blocking_ipv6: DEFAULT_BLOCKING_IPV6,
    blocked_response_ttl: 10,
    upstream_timeout: 10,
    edns_cs_enabled: false,
    disable_ipv6: false,
    dnssec_enabled: false,
    upstream_dns_file: '',
    upstream_dns: '',
    fallback_dns: '',
    bootstrap_dns: '',
    local_ptr_upstreams: '',
    ratelimit_whitelist: '',
    upstream_mode: '',
    resolve_clients: false,
    use_private_ptr_resolvers: false,
    default_local_ptr_upstreams: [],
};

const [state, setState] = createStore<DnsConfigState>(initialState);

export const getDnsConfig = async () => {
    setState('processingGetConfig', true);
    try {
        const data = await apiClient.getDnsConfig();
        setState({
            ...data,
            blocking_ipv4: data.blocking_ipv4 || DEFAULT_BLOCKING_IPV4,
            blocking_ipv6: data.blocking_ipv6 || DEFAULT_BLOCKING_IPV6,
            upstream_mode:
                data.upstream_mode === '' ? DNS_REQUEST_OPTIONS.LOAD_BALANCING : data.upstream_mode,
            bootstrap_dns: data.bootstrap_dns?.join('\n') || '',
            fallback_dns: data.fallback_dns?.join('\n') || '',
            local_ptr_upstreams: data.local_ptr_upstreams?.join('\n') || '',
            upstream_dns: data.upstream_dns?.join('\n') || '',
            ratelimit_whitelist: data.ratelimit_whitelist?.join('\n') || '',
            processingGetConfig: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingGetConfig', false);
    }
};

export const clearDnsCache = async () => {
    try {
        await apiClient.clearCache();
    } catch (error) {
        addErrorToast({ error });
    }
};

/**
 * Toggles `use_private_ptr_resolvers`.
 * Called from the Private Reverse page's header switch.
 * NOT called from the main DNS settings page.
 */
export const togglePrivatePtrResolvers = () => {
    setDnsConfig({ use_private_ptr_resolvers: !state.use_private_ptr_resolvers }, { silent: true });
};

/** Toggles `resolve_clients`. Used by PrivateReverse page switch. */
export const toggleResolveClients = () => {
    setDnsConfig({ resolve_clients: !state.resolve_clients }, { silent: true });
};

/** Toggles `dnssec_enabled`. Used by ServerConfig Section 2, Row 7. */
export const toggleDnssecEnabled = () => {
    setDnsConfig({ dnssec_enabled: !state.dnssec_enabled }, { silent: true });
};

/** Toggles `disable_ipv6` (inverted in UI). Used by ServerConfig Section 2, Row 8. */
export const toggleDisableIPv6 = () => {
    setDnsConfig({ disable_ipv6: !state.disable_ipv6 }, { silent: true });
};

/** Toggles `cache_enabled`. Used by Cache Section 3 header switch. */
export const toggleCacheEnabled = () => {
    setDnsConfig({ cache_enabled: !state.cache_enabled }, { silent: true });
};

/** Toggles `cache_optimistic`. Used by Cache Section 3, Row 4. */
export const toggleOptimisticCaching = () => {
    setDnsConfig({ cache_optimistic: !state.cache_optimistic }, { silent: true });
};

/** Toggles `edns_cs_enabled`. Used by ServerConfig Section 2, Row 6 switch-link. */
export const toggleEdnsCsEnabled = () => {
    setDnsConfig({ edns_cs_enabled: !state.edns_cs_enabled }, { silent: true });
};

export const setDnsConfig = async (
    values: any,
    opts?: { toastMessage?: string; silent?: boolean },
) => {
    setState('processingSetConfig', true);
    try {
        const config = { ...values };

        if (Object.hasOwn(config, 'bootstrap_dns')) {
            config.bootstrap_dns = splitByNewLine(config.bootstrap_dns);
        }
        if (Object.hasOwn(config, 'fallback_dns')) {
            config.fallback_dns = splitByNewLine(config.fallback_dns);
        }
        if (Object.hasOwn(config, 'local_ptr_upstreams')) {
            config.local_ptr_upstreams = splitByNewLine(config.local_ptr_upstreams);
        }
        if (Object.hasOwn(config, 'upstream_dns')) {
            config.upstream_dns = splitByNewLine(config.upstream_dns);
        }
        if (Object.hasOwn(config, 'ratelimit_whitelist')) {
            config.ratelimit_whitelist = splitByNewLine(config.ratelimit_whitelist);
        }

        await apiClient.setDnsConfig(config);
        setState(reconcile({ ...untrack(() => state), ...values, processingSetConfig: false }));

        if (opts?.silent) {
            // Toggle switches — no toast needed
        } else if (opts?.toastMessage) {
            addSuccessToast(opts.toastMessage);
        } else {
            addSuccessToast(intl.getMessage('settings_notify_changes_saved'));
        }
    } catch (error) {
        addErrorToast({ error });
        setState('processingSetConfig', false);
    }
};

export const dnsConfigState = untrack(() => state);
