import intl from 'panel/common/intl';
import {
    MAX_PORT,
    R_CIDR,
    R_CIDR_IPV6,
    R_HOST,
    R_IPV4,
    R_IPV6,
    R_MAC,
    R_URL_REQUIRES_PROTOCOL,
    STANDARD_WEB_PORT,
    UNSAFE_PORTS,
    R_CLIENT_ID,
    R_DOMAIN,
    MAX_PASSWORD_LENGTH,
    MIN_PASSWORD_LENGTH,
    R_HOSTNAME,
    UINT32_RANGE,
} from './constants';

import { ip4ToInt, isValidAbsolutePath } from './form';

import { isIpInCidr, parseSubnetMask } from './helpers';

/** Return type for all validators: `undefined` means valid, string is the i18n error message. */
type ValidationResult = string | undefined;

// Validation functions
// If the value is valid, the validation function should return undefined.

/**
 * Validates that a value is non-empty.
 *
 * @example validateRequiredValue("hello")  // undefined (valid)
 * @example validateRequiredValue("")       // "Field is required"
 * @example validateRequiredValue(0)        // undefined (0 is valid)
 */
export const validateRequiredValue = (value?: string | number | boolean): ValidationResult => {
    const formattedValue = typeof value === 'string' ? value.trim() : value;
    if (
        formattedValue ||
        formattedValue === 0 ||
        (typeof formattedValue === 'string' && formattedValue.length !== 0)
    ) {
        return undefined;
    }
    return intl.getMessage('form_error_required');
};

/**
 * Context object passed to DHCP v4 validators.
 * Mirrors the shape of `allValues.v4` in v1's FormDHCPv4.
 */
interface DhcpV4Context {
    v4?: {
        gateway_ip?: string;
        subnet_mask?: string;
        range_start?: string;
        range_end?: string;
    };
}

/**
 * Validates that the DHCP range end is greater than the range start.
 *
 * @example validateIpv4RangeEnd(undefined, { v4: { range_start: "192.168.1.1", range_end: "192.168.1.254" } })
 *          // undefined (valid)
 * @example validateIpv4RangeEnd(undefined, { v4: { range_start: "192.168.1.254", range_end: "192.168.1.1" } })
 *          // "Must be greater than range start"
 */
export const validateIpv4RangeEnd = (_: undefined, allValues?: DhcpV4Context): ValidationResult => {
    if (!allValues || !allValues.v4 || !allValues.v4.range_end || !allValues.v4.range_start) {
        return undefined;
    }

    const { range_end, range_start } = allValues.v4;

    if (ip4ToInt(range_end) <= ip4ToInt(range_start)) {
        return intl.getMessage('greater_range_start_error');
    }

    return undefined;
};

/**
 * Validates an IPv4 address format.
 *
 * @example validateIpv4("192.168.1.1")         // undefined (valid)
 * @example validateIpv4("999.999.999.999")     // "Invalid IPv4 address"
 */
export const validateIpv4 = (value?: string): ValidationResult => {
    if (value && !R_IPV4.test(value)) {
        return intl.getMessage('form_error_ip4_format');
    }
    return undefined;
};

/**
 * Validates that a gateway IP is outside the DHCP range.
 *
 * @example validateNotInRange("192.168.1.1", { v4: { range_start: "192.168.1.2", range_end: "192.168.1.254" } })
 *          // undefined (not in range)
 * @example validateNotInRange("192.168.1.100", { v4: { range_start: "192.168.1.2", range_end: "192.168.1.254" } })
 *          // "Must be out of range 192.168.1.2-192.168.1.254"
 */
export const validateNotInRange = (value: string, allValues?: DhcpV4Context): ValidationResult => {
    if (!allValues.v4) {
        return undefined;
    }

    const { range_start, range_end } = allValues.v4;

    if (range_start && validateIpv4(range_start)) {
        return undefined;
    }

    if (range_end && validateIpv4(range_end)) {
        return undefined;
    }

    const isAboveMin = range_start && ip4ToInt(value) >= ip4ToInt(range_start);
    const isBelowMax = range_end && ip4ToInt(value) <= ip4ToInt(range_end);

    if (isAboveMin && isBelowMax) {
        return intl.getMessage('out_of_range_error', {
            start: range_start,
            end: range_end,
        });
    }

    return undefined;
};

/**
 * Validates the gateway IP + subnet mask combination.
 *
 * @example validateGatewaySubnetMask(undefined, { v4: { gateway_ip: "192.168.1.1", subnet_mask: "255.255.255.0" } })
 *          // undefined (valid)
 * @example validateGatewaySubnetMask(undefined, { v4: { gateway_ip: "192.168.1.1", subnet_mask: "bad" } })
 *          // "Invalid subnet mask"
 */
export const validateGatewaySubnetMask = (
    _: undefined,
    allValues?: DhcpV4Context,
): ValidationResult => {
    if (!allValues || !allValues.v4 || !allValues.v4.subnet_mask || !allValues.v4.gateway_ip) {
        return intl.getMessage('gateway_or_subnet_invalid');
    }

    const { subnet_mask, gateway_ip } = allValues.v4;

    if (validateIpv4(gateway_ip) || validateIpv4(subnet_mask)) {
        return intl.getMessage('gateway_or_subnet_invalid');
    }

    return parseSubnetMask(subnet_mask) ? undefined : intl.getMessage('gateway_or_subnet_invalid');
};

/**
 * Validates that an IP address belongs to the gateway's subnet.
 *
 * @example
 * validateIpForGatewaySubnetMask(
 *   "192.168.1.5",
 *   { v4: { gateway_ip: "192.168.1.1", subnet_mask: "255.255.255.0" } },
 * ) // undefined (in subnet)
 * @example
 * validateIpForGatewaySubnetMask(
 *   "10.0.0.1",
 *   { v4: { gateway_ip: "192.168.1.1", subnet_mask: "255.255.255.0" } },
 * ) // "Addresses must be in one subnet"
 */
export const validateIpForGatewaySubnetMask = (
    value: string,
    allValues?: DhcpV4Context,
): ValidationResult => {
    if (
        !allValues ||
        !allValues.v4 ||
        !value ||
        !allValues.v4.gateway_ip ||
        !allValues.v4.subnet_mask
    ) {
        return undefined;
    }

    const { gateway_ip, subnet_mask } = allValues.v4;

    if ((gateway_ip && validateIpv4(gateway_ip)) || (subnet_mask && validateIpv4(subnet_mask))) {
        return undefined;
    }

    const subnetPrefix = parseSubnetMask(subnet_mask);

    if (!isIpInCidr(value, `${gateway_ip}/${subnetPrefix}`)) {
        return intl.getMessage('subnet_error');
    }

    return undefined;
};

/**
 * Validates a client ID format.
 *
 * @example validateConfigClientId("my-device")    // undefined (valid)
 * @example validateConfigClientId("")             // undefined (empty = valid)
 */
export const validateConfigClientId = (value?: string): ValidationResult => {
    if (!value) {
        return undefined;
    }
    const formattedValue = value.trim();
    if (formattedValue && !R_CLIENT_ID.test(formattedValue)) {
        return intl.getMessage('form_error_client_id_format');
    }
    return undefined;
};

/**
 * Validates a server name (domain) for encryption config.
 *
 * @example validateServerName("dns.example.com")  // undefined (valid)
 * @example validateServerName("")                 // undefined (empty = valid)
 */
export const validateServerName = (value?: string): ValidationResult => {
    if (!value) {
        return undefined;
    }
    const formattedValue = value ? value.trim() : value;
    if (formattedValue && !R_DOMAIN.test(formattedValue)) {
        return intl.getMessage('form_error_server_name');
    }
    return undefined;
};

/**
 * Validates an IPv6 address format.
 *
 * @example validateIpv6("::1")                     // undefined (valid)
 * @example validateIpv6("not-an-ip")               // "Invalid IPv6 address"
 */
export const validateIpv6 = (value?: string): ValidationResult => {
    if (value && !R_IPV6.test(value)) {
        return intl.getMessage('form_error_ip6_format');
    }
    return undefined;
};

/**
 * Validates an IP address (v4 or v6) format.
 *
 * @example validateIp("192.168.1.1")     // undefined (valid)
 * @example validateIp("::1")             // undefined (valid)
 * @example validateIp("bad")             // "Invalid IP address"
 */
export const validateIp = (value?: string): ValidationResult => {
    if (value && !R_IPV4.test(value) && !R_IPV6.test(value)) {
        return intl.getMessage('form_error_ip_format');
    }
    return undefined;
};

/**
 * Validates that each non-empty line is a valid IP address.
 *
 * @example validateIpPerLine("192.168.1.1\n10.0.0.1")  // undefined (valid)
 * @example validateIpPerLine("bad")                      // "Invalid IP address"
 */
export const validateIpPerLine = (value: string): string | undefined => {
    if (!value) return undefined;
    const lines = value.split('\n');
    for (let i = 0; i < lines.length; i += 1) {
        const line = lines[i].trim();
        if (line) {
            const error = validateIp(line);
            if (error) return error;
        }
    }
    return undefined;
};

/**
 * Validates a MAC address format.
 *
 * @example validateMac("00:11:22:33:44:55")        // undefined (valid)
 * @example validateMac("not-a-mac")                 // "Invalid MAC address"
 */
export const validateMac = (value?: string): ValidationResult => {
    if (value && !R_MAC.test(value)) {
        return intl.getMessage('form_error_mac_format');
    }
    return undefined;
};

/**
 * Validates a port number is within the standard web range (80–65535).
 *
 * @example validatePort(8080)      // undefined (valid)
 * @example validatePort(80)        // undefined (valid)
 * @example validatePort(79)        // "Enter port number in the range of 80-65535"
 */
export const validatePort = (value?: number): ValidationResult => {
    if ((value || value === 0) && (value < STANDARD_WEB_PORT || value > MAX_PORT)) {
        return intl.getMessage('form_error_port_range');
    }
    return undefined;
};

/**
 * Validates a port number is in the full range (1–65535) for install wizard.
 * Unlike `validatePort`, 0 is INVALID (install requires a real port).
 *
 * @example validateInstallPort(80)    // undefined (valid)
 * @example validateInstallPort(0)     // "Invalid port number"
 */
export const validateInstallPort = (value?: number): ValidationResult => {
    if (value < 1 || value > MAX_PORT) {
        return intl.getMessage('form_error_port');
    }
    return undefined;
};

/**
 * Validates a TLS port number. 0 = disabled (valid).
 *
 * @example validatePortTLS(0)         // undefined (valid, disabled)
 * @example validatePortTLS(853)       // undefined (valid)
 * @example validatePortTLS(79)        // "Enter port number in the range of 80-65535"
 */
export const validatePortTLS = (value?: number): ValidationResult => {
    if (value === 0) {
        return undefined;
    }
    if (value && (value < STANDARD_WEB_PORT || value > MAX_PORT)) {
        return intl.getMessage('form_error_port_range');
    }
    return undefined;
};

export const validatePortQuic = validatePortTLS; // inherits type

/**
 * Checks a port against the unsafe ports list.
 *
 * @example validateIsSafePort(443)     // undefined (safe)
 * @example validateIsSafePort(53)      // "Port is unsafe"
 */
export const validateIsSafePort = (value?: number): ValidationResult => {
    if (UNSAFE_PORTS.includes(value)) {
        return intl.getMessage('form_error_port_unsafe');
    }
    return undefined;
};

/**
 * Validates a domain name format.
 *
 * @example validateDomain("example.com")    // undefined (valid)
 * @example validateDomain("not a domain")   // "Invalid domain name"
 */
export const validateDomain = (value?: string): ValidationResult => {
    if (value && !R_HOST.test(value)) {
        return intl.getMessage('form_error_domain_format');
    }
    return undefined;
};

/**
 * Validates a DNS answer (IP or domain).
 *
 * @example validateAnswer("192.168.1.1")      // undefined (valid)
 * @example validateAnswer("example.com")      // undefined (valid)
 * @example validateAnswer("bad")              // "Invalid answer"
 */
export const validateAnswer = (value?: string): ValidationResult => {
    if (value && !R_IPV4.test(value) && !R_IPV6.test(value) && !R_HOST.test(value)) {
        return intl.getMessage('form_error_answer_format');
    }
    return undefined;
};

/**
 * Validates an absolute file path or URL format.
 *
 * @example validatePath("/usr/local/bin")      // undefined (valid)
 * @example validatePath("http://example.com")  // undefined (valid)
 */
export const validatePath = (value?: string): ValidationResult => {
    if (value && !isValidAbsolutePath(value) && !R_URL_REQUIRES_PROTOCOL.test(value)) {
        return intl.getMessage('form_error_url_format');
    }
    return undefined;
};

/**
 * Validates that an IP is contained within a CIDR range.
 *
 * @example validateIpv4InCidr("192.168.1.5", { cidr: "192.168.1.0/24" })
 *          // undefined (within CIDR)
 * @example validateIpv4InCidr("10.0.0.1", { cidr: "192.168.1.0/24" })
 *          // "192.168.1.0/24 does not contain 10.0.0.1"
 */
export const validateIpv4InCidr = (
    valueIp: string,
    allValues: { cidr: string },
): ValidationResult => {
    if (!isIpInCidr(valueIp, allValues.cidr)) {
        return intl.getMessage('form_error_subnet', { ip: valueIp, cidr: allValues.cidr });
    }
    return undefined;
};

const utf8StringLength = (value: string): number => {
    const encoder = new TextEncoder();
    const view = encoder.encode(value);

    return view.length;
};

/**
 * Validates password length. Returns "true" (react-hook-form convention)
 * if invalid, NOT an i18n string.
 *
 * @example validatePasswordLength("short")      // true (too short)
 * @example validatePasswordLength("longenough") // undefined (valid)
 */
export const validatePasswordLength = (value?: string): boolean | undefined => {
    if (value) {
        const length = utf8StringLength(value);
        if (length < MIN_PASSWORD_LENGTH || length > MAX_PASSWORD_LENGTH) {
            return true;
        }
    }

    return undefined;
};

/**
 * Validates that an IP is not the same as the gateway IP.
 *
 * @example validateIpGateway("192.168.1.2", { gatewayIp: "192.168.1.1" })
 *          // undefined (different)
 * @example validateIpGateway("192.168.1.1", { gatewayIp: "192.168.1.1" })
 *          // "IP address cannot be the same as gateway"
 */
export const validateIpGateway = (
    value: string,
    allValues: { gatewayIp: string },
): ValidationResult => {
    if (value === allValues.gatewayIp) {
        return intl.getMessage('form_error_gateway_ip');
    }
    return undefined;
};

/**
 * Validates that plain DNS is configured when encryption is disabled.
 * At least one of encryption or plain DNS must be enabled.
 *
 * @example validatePlainDns(true, { enabled: true })   // undefined (both on)
 * @example validatePlainDns(true, { enabled: false })  // undefined (plain DNS on)
 * @example validatePlainDns(false, { enabled: true })  // undefined (encryption on)
 * @example validatePlainDns(false, { enabled: false }) // error string (neither on)
 */
export const validatePlainDns = (
    value: boolean,
    allValues: { enabled?: boolean },
): ValidationResult => {
    const { enabled } = allValues;

    if (!enabled && !value) {
        return intl.getMessage('encryption_plain_dns_error');
    }

    return undefined;
};

/**
 * Validates a single client identifier value.
 * Returns undefined if valid, or an i18n error message string if invalid.
 *
 * @param value - The identifier string to validate
 * @param allValues - All identifier values in the form (for duplicate checking)
 * @param currentIndex - The index of this identifier in the array
 * @param existingIds - Identifiers from all other persistent clients (for
 *   cross-client duplicate checking, excluding the client being edited)
 *
 * @example validateIdentifier("192.168.1.1", [], 0)        // undefined (valid IPv4)
 * @example validateIdentifier("", [], 0)                   // "Field is required"
 */
export const validateIdentifier = (
    value: string,
    allValues: string[],
    currentIndex: number,
    existingIds?: string[],
): string | undefined => {
    const trimmed = (value || '').trim();

    if (!trimmed) {
        return intl.getMessage('form_error_required');
    }

    const isValidFormat =
        R_IPV4.test(trimmed) ||
        R_IPV6.test(trimmed) ||
        R_MAC.test(trimmed) ||
        R_CIDR.test(trimmed) ||
        R_CIDR_IPV6.test(trimmed) ||
        R_CLIENT_ID.test(trimmed);

    if (!isValidFormat) {
        return intl.getMessage('clients_identifier_format_error');
    }

    const duplicateIndex = allValues.findIndex(
        (v, i) => i !== currentIndex && v.trim() === trimmed,
    );
    if (duplicateIndex !== -1) {
        return intl.getMessage('clients_identifier_already_used');
    }

    if (existingIds && existingIds.includes(trimmed)) {
        return intl.getMessage('clients_identifier_already_used');
    }

    return undefined;
};

/**
 * Regex matching a hostname that consists only of digits.
 */
const R_ALL_DIGITS = /^[0-9]+$/;

/**
 * Validates a hostname format.  Empty values are considered valid.
 *
 * @example validateHostname("my-router")          // undefined (valid)
 * @example validateHostname("")                   // undefined (empty = valid)
 * @example validateHostname("123")                // "Use numbers …" (all-numeric)
 * @example validateHostname("-")                  // "Use numbers …" (only hyphens)
 * @example validateHostname("-host")              // "Use numbers …" (starts with hyphen)
 */
export const validateHostname = (value?: string): ValidationResult => {
    if (!value) {
        return undefined;
    }

    if (
        !R_HOSTNAME.test(value)
        || R_ALL_DIGITS.test(value)
        || value.startsWith('-')
        || value.endsWith('-')
    ) {
        return intl.getMessage('form_error_hostname_format');
    }

    return undefined;
};

// A valid upstream line contains at least one dot or colon.
const R_COMMENT = /^\s*#/;
const R_HAS_ADDRESS = /[.:]/;

/**
 * Validates upstream server lines. Each line must contain a dot or colon.
 *
 * @example validateUpstreams("https://dns.example.com")    // undefined (valid)
 * @example validateUpstreams("# comment\n1.1.1.1")        // undefined (comments ok)
 * @example validateUpstreams("not-a-server")               // "Invalid upstream"
 */
export const validateUpstreams = (value: string): string | undefined => {
    if (!value) {
        return undefined;
    }
    const lines = value.split('\n');
    const invalidLines: number[] = [];
    for (let i = 0; i < lines.length; i += 1) {
        const line = lines[i].trim();
        if (line && !R_COMMENT.test(line) && !R_HAS_ADDRESS.test(line)) {
            invalidLines.push(i + 1);
        }
    }

    if (invalidLines.length === 0) {
        return undefined;
    }

    if (invalidLines.length === 1) {
        return intl.getMessage('form_error_upstream_format', { line: invalidLines[0] });
    }

    return intl.getMessage('form_error_upstream_format_multi', {
        lines: invalidLines.join(', '),
    });
};

interface LeaseEntry {
    ip: string;
    mac?: string;
}

export const validateIpNotDuplicate =
    (existingLeases: LeaseEntry[], editIp?: string): ((value?: string) => ValidationResult) =>
    (value) => {
        if (value && value !== editIp && existingLeases.some((lease) => lease.ip === value)) {
            return intl.getMessage('form_error_ip_already_added');
        }
        return undefined;
    };

export const validateMacNotDuplicate =
    (existingLeases: LeaseEntry[], editMac?: string): ((value?: string) => ValidationResult) =>
    (value) => {
        if (value && value !== editMac && existingLeases.some((lease) => lease.mac === value)) {
            return intl.getMessage('dhcp_mac_address_already_added');
        }
        return undefined;
    };

export const validateMaxValue = (value: number, max: number): string | undefined => {
    if (value > max) {
        return intl.getMessage('form_value_length_common_error', {
            max_length: String(max),
        });
    }
    return undefined;
};

export const validateBetween = (value: number, min: number, max: number): string | undefined => {
    if (value < min || value > max) {
        return intl.getMessage('form_value_value_between_error', {
            min_value: String(min),
            max_value: String(max),
        });
    }
    return undefined;
};

export const validateMinValue = (value: number, min: number): string | undefined => {
    if (value < min) {
        return intl.getMessage('form_value_value_min_error', {
            min_value: String(min),
        });
    }
    return undefined;
};

/**
 * Validates the client DNS cache size.
 * Callers should gate on `upstreams_cache_enabled` before invoking.
 * Returns undefined if valid, or an i18n error message string if invalid.
 *
 * @param size - The cache size in bytes
 * @param enabled - Whether per-client upstream caching is enabled
 *
 * @example validateCacheSize(0, true)         // error ("must be greater than zero")
 * @example validateCacheSize(1000, true)      // undefined (valid)
 * @example validateCacheSize(4294967296, true) // error (exceeds UINT32_MAX)
 * @example validateCacheSize(0, false)        // undefined (no validation when disabled)
 */
export const validateCacheSize = (size: number, enabled: boolean): ValidationResult => {
    if (!enabled) {
        return undefined;
    }
    if (size === 0) {
        return intl.getMessage('cache_config_size_validation');
    }
    return validateMaxValue(size, UINT32_RANGE.MAX);
};
