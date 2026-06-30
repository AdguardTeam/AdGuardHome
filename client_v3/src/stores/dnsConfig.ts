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

export const setDnsConfig = async (values: any, toastMessage?: string) => {
    setState('processingSetConfig', true);
    try {
        const config = { ...values };
        let hasDnsSettings = false;

        if (Object.hasOwn(config, 'bootstrap_dns')) {
            config.bootstrap_dns = splitByNewLine(config.bootstrap_dns);
            hasDnsSettings = true;
        }
        if (Object.hasOwn(config, 'fallback_dns')) {
            config.fallback_dns = splitByNewLine(config.fallback_dns);
            hasDnsSettings = true;
        }
        if (Object.hasOwn(config, 'local_ptr_upstreams')) {
            config.local_ptr_upstreams = splitByNewLine(config.local_ptr_upstreams);
            hasDnsSettings = true;
        }
        if (Object.hasOwn(config, 'upstream_dns')) {
            config.upstream_dns = splitByNewLine(config.upstream_dns);
            hasDnsSettings = true;
        }
        if (Object.hasOwn(config, 'ratelimit_whitelist')) {
            config.ratelimit_whitelist = splitByNewLine(config.ratelimit_whitelist);
            hasDnsSettings = true;
        }

        await apiClient.setDnsConfig(config);
        setState(reconcile({ ...untrack(() => state), ...values, processingSetConfig: false }));

        if (toastMessage) {
            addSuccessToast(toastMessage);
        } else if (hasDnsSettings) {
            addSuccessToast(intl.getMessage('updated_upstream_dns_toast'));
        } else {
            addSuccessToast(intl.getMessage('settings_notify_changes_saved'));
        }
    } catch (error) {
        addErrorToast({ error });
        setState('processingSetConfig', false);
    }
};

export const dnsConfigState = untrack(() => state);
