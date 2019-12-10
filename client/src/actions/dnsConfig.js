import { createAction } from 'redux-actions';

import apiClient from '../api/Api';
import { addErrorToast, addSuccessToast } from './index';

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
        await apiClient.setDnsConfig(config);
        dispatch(addSuccessToast('config_successfully_saved'));
        dispatch(setDnsConfigSuccess(config));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setDnsConfigFailure());
    }
};
