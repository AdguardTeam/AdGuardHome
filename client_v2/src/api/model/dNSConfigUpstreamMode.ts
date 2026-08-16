/**
 * Upstream modes enumeration. The empty string value is deprecated; use `load_balance` instead.
 */
export type DNSConfigUpstreamMode = '' | 'fastest_addr' | 'load_balance' | 'parallel';
