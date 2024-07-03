import { createAction } from 'redux-actions';
import i18next from 'i18next';

import apiClient from '../api/Api';

import { splitByNewLine } from '../helpers/helpers';
import { addErrorToast, addSuccessToast } from './toasts';

export const getDnsConfigRequest = createAction('GET_DNS_CONFIG_REQUEST');
export const getDnsConfigFailure = createAction('GET_DNS_CONFIG_FAILURE');
export const getDnsConfigSuccess = createAction('GET_DNS_CONFIG_SUCCESS');

export const getDnsConfig = () => async (dispatch: any) => {
    dispatch(getDnsConfigRequest());
    try {
        const data = await apiClient.getDnsConfig();
        dispatch(getDnsConfigSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getDnsConfigFailure());
    }
};

export const clearDnsCacheRequest = createAction('CLEAR_DNS_CACHE_REQUEST');
export const clearDnsCacheFailure = createAction('CLEAR_DNS_CACHE_FAILURE');
export const clearDnsCacheSuccess = createAction('CLEAR_DNS_CACHE_SUCCESS');

export const clearDnsCache = () => async (dispatch: any) => {
    dispatch(clearDnsCacheRequest());
    try {
        const data = await apiClient.clearCache();
        dispatch(clearDnsCacheSuccess(data));
        dispatch(addSuccessToast(i18next.t('cache_cleared')));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(clearDnsCacheFailure());
    }
};

export const setDnsConfigRequest = createAction('SET_DNS_CONFIG_REQUEST');
export const setDnsConfigFailure = createAction('SET_DNS_CONFIG_FAILURE');
export const setDnsConfigSuccess = createAction('SET_DNS_CONFIG_SUCCESS');

export const setDnsConfig = (config: any) => async (dispatch: any) => {
    dispatch(setDnsConfigRequest());
    try {
        const data = { ...config };

        let hasDnsSettings = false;
        if (Object.prototype.hasOwnProperty.call(data, 'bootstrap_dns')) {
            data.bootstrap_dns = splitByNewLine(config.bootstrap_dns);
            hasDnsSettings = true;
        }
        if (Object.prototype.hasOwnProperty.call(data, 'fallback_dns')) {
            data.fallback_dns = splitByNewLine(config.fallback_dns);
            hasDnsSettings = true;
        }
        if (Object.prototype.hasOwnProperty.call(data, 'local_ptr_upstreams')) {
            data.local_ptr_upstreams = splitByNewLine(config.local_ptr_upstreams);
            hasDnsSettings = true;
        }
        if (Object.prototype.hasOwnProperty.call(data, 'upstream_dns')) {
            data.upstream_dns = splitByNewLine(config.upstream_dns);
            hasDnsSettings = true;
        }
        if (Object.prototype.hasOwnProperty.call(data, 'ratelimit_whitelist')) {
            data.ratelimit_whitelist = splitByNewLine(config.ratelimit_whitelist);
            hasDnsSettings = true;
        }

        await apiClient.setDnsConfig(data);

        if (hasDnsSettings) {
            dispatch(addSuccessToast('updated_upstream_dns_toast'));
        } else {
            dispatch(addSuccessToast('config_successfully_saved'));
        }

        dispatch(setDnsConfigSuccess(config));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setDnsConfigFailure());
    }
};
