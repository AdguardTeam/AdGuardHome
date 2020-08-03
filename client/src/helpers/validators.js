import { Trans } from 'react-i18next';
import React from 'react';
import i18next from 'i18next';
import {
    R_CIDR,
    R_CIDR_IPV6,
    R_HOST,
    R_IPV4,
    R_IPV6,
    R_MAC,
    R_URL_REQUIRES_PROTOCOL,
    UNSAFE_PORTS,
} from './constants';
import { isValidAbsolutePath } from './form';


// Validation functions
// https://redux-form.com/8.3.0/examples/fieldlevelvalidation/
// If the value is valid, the validation function should return undefined.
/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validateRequiredValue = (value) => {
    const formattedValue = typeof value === 'string' ? value.trim() : value;
    if (formattedValue || formattedValue === 0 || (formattedValue && formattedValue.length !== 0)) {
        return undefined;
    }
    return <Trans>form_error_required</Trans>;
};

/**
 * @param maximum {number}
 * @returns {(value:number) => undefined|string}
 */
export const getMaxValueValidator = (maximum) => (value) => {
    if (value && value > maximum) {
        i18next.t('value_not_larger_than', { maximum });
    }
    return undefined;
};


/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validateIpv4 = (value) => {
    if (value && !R_IPV4.test(value)) {
        return <Trans>form_error_ip4_format</Trans>;
    }
    return undefined;
};

/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validateClientId = (value) => {
    if (!value) {
        return undefined;
    }
    const formattedValue = value ? value.trim() : value;
    if (formattedValue && !(
        R_IPV4.test(formattedValue)
        || R_IPV6.test(formattedValue)
        || R_MAC.test(formattedValue)
        || R_CIDR.test(formattedValue)
        || R_CIDR_IPV6.test(formattedValue)
    )) {
        return <Trans>form_error_client_id_format</Trans>;
    }
    return undefined;
};

/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validateIpv6 = (value) => {
    if (value && !R_IPV6.test(value)) {
        return <Trans>form_error_ip6_format</Trans>;
    }
    return undefined;
};

/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validateIp = (value) => {
    if (value && !R_IPV4.test(value) && !R_IPV6.test(value)) {
        return <Trans>form_error_ip_format</Trans>;
    }
    return undefined;
};

/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validateMac = (value) => {
    if (value && !R_MAC.test(value)) {
        return <Trans>form_error_mac_format</Trans>;
    }
    return undefined;
};

/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validateIsPositiveValue = (value) => {
    if ((value || value === 0) && value <= 0) {
        return <Trans>form_error_positive</Trans>;
    }
    return undefined;
};

/**
 * @param value {string}
 * @returns {boolean|*}
 */
export const validateBiggerOrEqualZeroValue = (value) => {
    if (value < 0) {
        return <Trans>form_error_negative</Trans>;
    }
    return false;
};

/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validatePort = (value) => {
    if ((value || value === 0) && (value < 80 || value > 65535)) {
        return <Trans>form_error_port_range</Trans>;
    }
    return undefined;
};

/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validateInstallPort = (value) => {
    if (value < 1 || value > 65535) {
        return <Trans>form_error_port</Trans>;
    }
    return undefined;
};

/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validatePortTLS = (value) => {
    if (value === 0) {
        return undefined;
    }
    if (value && (value < 80 || value > 65535)) {
        return <Trans>form_error_port_range</Trans>;
    }
    return undefined;
};

/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validateIsSafePort = (value) => {
    if (UNSAFE_PORTS.includes(value)) {
        return <Trans>form_error_port_unsafe</Trans>;
    }
    return undefined;
};

/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validateDomain = (value) => {
    if (value && !R_HOST.test(value)) {
        return <Trans>form_error_domain_format</Trans>;
    }
    return undefined;
};

/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validateAnswer = (value) => {
    if (value && (!R_IPV4.test(value) && !R_IPV6.test(value) && !R_HOST.test(value))) {
        return <Trans>form_error_answer_format</Trans>;
    }
    return undefined;
};

/**
 * @param value {string}
 * @returns {undefined|string}
 */
export const validatePath = (value) => {
    if (value && !isValidAbsolutePath(value) && !R_URL_REQUIRES_PROTOCOL.test(value)) {
        return <Trans>form_error_url_or_path_format</Trans>;
    }
    return undefined;
};
