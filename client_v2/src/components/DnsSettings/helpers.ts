import { DNS_REQUEST_OPTIONS, BLOCKING_MODES, EDNS_MODES } from 'panel/helpers/constants';
import intl from 'panel/common/intl';

export const getUpstreamModeSummary = (mode: string): string => {
    switch (mode) {
        case DNS_REQUEST_OPTIONS.PARALLEL:
            return intl.getMessage('upstream_dns_parallel_requests');
        case DNS_REQUEST_OPTIONS.FASTEST_ADDR:
            return intl.getMessage('upstream_dns_fastest_addr');
        case DNS_REQUEST_OPTIONS.LOAD_BALANCING:
        default:
            return intl.getMessage('upstream_dns_load_balancing');
    }
};

export const getUpstreamServersSummary = (servers: string, upstreamDnsFile?: string): string => {
    if (upstreamDnsFile) {
        return intl.getMessage('upstream_dns_configured_in_file', { path: upstreamDnsFile });
    }
    if (!servers) return '';
    const lines = servers.split('\n').filter(Boolean);
    return lines.join(', ');
};

export const getRateLimitSummary = (ratelimit: number): string => {
    if (ratelimit === 0) return intl.getMessage('dns_rate_limit_no_limit');
    return intl.getMessage('dns_rate_limit_value', { value: ratelimit });
};

export const getBlockingModeSummary = (mode: string): string => {
    switch (mode) {
        case BLOCKING_MODES.refused:
            return 'REFUSED';
        case BLOCKING_MODES.nxdomain:
            return 'NXDOMAIN';
        case BLOCKING_MODES.null_ip:
            return intl.getMessage('dns_blocking_mode_null_ip');
        case BLOCKING_MODES.custom_ip:
            return intl.getMessage('dns_blocking_mode_custom_ip');
        default:
            return intl.getMessage('dns_blocking_mode_default');
    }
};

export const getCacheSizeSummary = (bytes?: number): string => {
    if (!bytes && bytes !== 0) return '';
    if (bytes < 1024) return `${bytes} bytes`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
    return `${Number((bytes / (1024 * 1024)).toFixed(1))} MB`;
};

export const getTtlSummary = (seconds?: number): string => {
    if (seconds === undefined || seconds === null) return '';
    return intl.getMessage('dns_ttl_value', { value: seconds });
};

export const getListSummary = (list: string): string => {
    if (!list || !list.trim()) return '';
    return list.split('\n').filter(Boolean).join(', ');
};

/**
 * Returns blocking mode options with i18n text.
 * Factory function — re-executes on each render for locale correctness.
 */
export const getBlockingModeOptions = () => {
    return [
        {
            text: intl.getMessage('dns_blocking_mode_default'),
            value: BLOCKING_MODES.default,
            description: intl.getMessage('dns_blocking_mode_default_desc'),
        },
        {
            text: intl.getMessage('dns_blocking_mode_refused'),
            value: BLOCKING_MODES.refused,
            description: intl.getMessage('dns_blocking_mode_refused_desc'),
        },
        {
            text: intl.getMessage('dns_blocking_mode_nxdomain'),
            value: BLOCKING_MODES.nxdomain,
            description: intl.getMessage('dns_blocking_mode_nxdomain_desc'),
        },
        {
            text: intl.getMessage('dns_blocking_mode_null_ip'),
            value: BLOCKING_MODES.null_ip,
            description: intl.getMessage('dns_blocking_mode_null_ip_desc'),
        },
        {
            text: intl.getMessage('dns_blocking_mode_custom_ip'),
            value: BLOCKING_MODES.custom_ip,
            description: intl.getMessage('dns_blocking_mode_custom_ip_desc'),
        },
    ];
};

/**
 * Returns EDNS mode options with i18n text.
 * Factory function — re-executes on each render for locale correctness.
 */
export const getEdnsOptions = () => {
    return [
        {
            text: intl.getMessage('dns_edns_option_default'),
            value: EDNS_MODES.default,
        },
        {
            text: intl.getMessage('dns_edns_option_custom'),
            value: EDNS_MODES.custom,
        },
    ];
};

/**
 * Returns upstream mode options with i18n text.
 * Factory function — re-executes on each call for locale correctness.
 * When used inside `createMemo`, tracks the locale signal and updates
 * option labels on language change.
 */
export const getUpstreamModeOptions = () => {
    return [
        {
            text: intl.getMessage('upstream_dns_load_balancing'),
            value: DNS_REQUEST_OPTIONS.LOAD_BALANCING,
            description: intl.getMessage('upstream_dns_load_balancing_desc'),
        },
        {
            text: intl.getMessage('upstream_dns_parallel_requests'),
            value: DNS_REQUEST_OPTIONS.PARALLEL,
            description: intl.getMessage('upstream_dns_parallel_requests_desc'),
        },
        {
            text: intl.getMessage('upstream_dns_fastest_addr'),
            value: DNS_REQUEST_OPTIONS.FASTEST_ADDR,
            description: intl.getMessage('upstream_dns_fastest_addr_desc'),
            warning: intl.getMessage('upstream_dns_fastest_addr_warning'),
        },
    ];
};
