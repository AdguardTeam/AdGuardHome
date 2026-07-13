import { R_MAC_WITHOUT_COLON, R_UNIX_ABSOLUTE_PATH, R_WIN_ABSOLUTE_PATH } from './constants';

/**
 *
 * @param {string} ip
 * @returns {*}
 */
export const ip4ToInt = (ip: any) => {
    const intIp = ip.split('.').reduce((int: any, oct: any) => int * 256 + parseInt(oct, 10), 0);
    return Number.isNaN(intIp) ? 0 : intIp;
};

/**
 * @param value {string}
 * @returns {*|number}
 */
export const toNumber = (value: any) => value && parseInt(value, 10);

/**
 * @param value {string}
 * @returns {*|number}
 */

export const toFloatNumber = (value: any) => value && parseFloat(value);

/**
 * @param value {string}
 * @returns {boolean}
 */
export const isValidAbsolutePath = (value: any) =>
    R_WIN_ABSOLUTE_PATH.test(value) || R_UNIX_ABSOLUTE_PATH.test(value);

/**
 * Normalizes a MAC address to colon-separated uppercase format.
 * Handles bare hex (12 or 16 chars), dash-separated, and colon-separated formats.
 *
 * @example normalizeMac("aabbccddeeff")   // "AA:BB:CC:DD:EE:FF"
 * @example normalizeMac("AA-BB-CC-DD-EE-FF") // "AA:BB:CC:DD:EE:FF"
 * @example normalizeMac("aa:bb:cc:dd:ee:ff") // "AA:BB:CC:DD:EE:FF"
 */
export const normalizeMac = (value: any) => {
    if (!value || typeof value !== 'string') return value;

    // Handle separator-less bare hex (12 or 16 chars)
    if (R_MAC_WITHOUT_COLON.test(value)) {
        return value.match(/.{2}/g).join(':').toUpperCase();
    }

    // Handle dash-separated MACs
    if (value.includes('-')) {
        return value.replace(/-/g, ':').toUpperCase();
    }

    // Already colon-separated or other format — just uppercase
    return value.toUpperCase();
};
