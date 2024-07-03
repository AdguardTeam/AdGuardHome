import { createAction } from 'redux-actions';
import apiClient from '../api/Api';
import { addErrorToast, addSuccessToast } from './toasts';

export const nextStep = createAction('NEXT_STEP');
export const prevStep = createAction('PREV_STEP');

export const getDefaultAddressesRequest = createAction('GET_DEFAULT_ADDRESSES_REQUEST');
export const getDefaultAddressesFailure = createAction('GET_DEFAULT_ADDRESSES_FAILURE');
export const getDefaultAddressesSuccess = createAction('GET_DEFAULT_ADDRESSES_SUCCESS');

export const getDefaultAddresses = () => async (dispatch: any) => {
    dispatch(getDefaultAddressesRequest());
    try {
        const addresses = await apiClient.getDefaultAddresses();
        dispatch(getDefaultAddressesSuccess(addresses));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getDefaultAddressesFailure());
    }
};

export const setAllSettingsRequest = createAction('SET_ALL_SETTINGS_REQUEST');
export const setAllSettingsFailure = createAction('SET_ALL_SETTINGS_FAILURE');
export const setAllSettingsSuccess = createAction('SET_ALL_SETTINGS_SUCCESS');

export const setAllSettings = (values: any) => async (dispatch: any) => {
    dispatch(setAllSettingsRequest());
    try {
        const { confirm_password, ...config } = values;

        await apiClient.setAllSettings(config);
        dispatch(setAllSettingsSuccess());
        dispatch(addSuccessToast('install_saved'));
        dispatch(nextStep());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setAllSettingsFailure());
        dispatch(prevStep());
    }
};

export const checkConfigRequest = createAction('CHECK_CONFIG_REQUEST');
export const checkConfigFailure = createAction('CHECK_CONFIG_FAILURE');
export const checkConfigSuccess = createAction('CHECK_CONFIG_SUCCESS');

export const checkConfig = (values: any) => async (dispatch: any) => {
    dispatch(checkConfigRequest());
    try {
        const check = await apiClient.checkConfig(values);
        dispatch(checkConfigSuccess(check));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(checkConfigFailure());
    }
};
