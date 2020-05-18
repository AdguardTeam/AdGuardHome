import { createAction } from 'redux-actions';

import apiClient from '../api/Api';
import { addErrorToast, addSuccessToast } from './index';
import { normalizeTextarea } from '../helpers/helpers';

export const getDnsConfigRequest = createAction('GET_DNS_CONFIG_REQUEST');
export const getDnsConfigFailure = createAction('GET_DNS_CONFIG_FAILURE');
export const getDnsConfigSuccess = createAction('GET_DNS_CONFIG_SUCCESS');

export const getDnsConfig = () => async (dispatch) => {
    dispatch(getDnsConfigRequest());
    try {
        const data = await apiClient.getDnsConfig();
        dispatch(getDnsConfigSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getDnsConfigFailure());
    }
};

export const setDnsConfigRequest = createAction('SET_DNS_CONFIG_REQUEST');
export const setDnsConfigFailure = createAction('SET_DNS_CONFIG_FAILURE');
export const setDnsConfigSuccess = createAction('SET_DNS_CONFIG_SUCCESS');

export const setDnsConfig = config => async (dispatch) => {
    dispatch(setDnsConfigRequest());
    try {
        const data = { ...config };

        let hasDnsSettings = false;
        if (Object.prototype.hasOwnProperty.call(data, 'bootstrap_dns')) {
            data.bootstrap_dns = normalizeTextarea(config.bootstrap_dns);
            hasDnsSettings = true;
        }
        if (Object.prototype.hasOwnProperty.call(data, 'upstream_dns')) {
            data.upstream_dns = normalizeTextarea(config.upstream_dns);
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
