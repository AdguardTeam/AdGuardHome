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
export const isValidAbsolutePath = (value: any) => R_WIN_ABSOLUTE_PATH.test(value) || R_UNIX_ABSOLUTE_PATH.test(value);

/**
 * @param value {string}
 * @returns {*|string}
 */
export const normalizeMac = (value: any) => {
    if (value && R_MAC_WITHOUT_COLON.test(value)) {
        return value.match(/.{2}/g).join(':');
    }

    return value;
};
