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

        // Reload configuration from server to ensure UI reflects saved state
        const updatedConfig = await apiClient.getDnsConfig();

        if (hasDnsSettings) {
            dispatch(addSuccessToast('updated_upstream_dns_toast'));
        } else {
            dispatch(addSuccessToast('config_successfully_saved'));
        }

        dispatch(setDnsConfigSuccess(updatedConfig));
    } catch (error) {
        // Parse error message to provide better user feedback
        const errorMessage = error instanceof Error ? error.message : String(error);

        // Check if error is related to IPSet
        if (errorMessage.includes('ipset') || errorMessage.includes('unknown ipset')) {
            const customError = new Error(i18next.t('ipset_error_save_failed'));
            dispatch(addErrorToast({ error: customError }));
        } else {
            dispatch(addErrorToast({ error }));
        }

        dispatch(setDnsConfigFailure());
    }
};
