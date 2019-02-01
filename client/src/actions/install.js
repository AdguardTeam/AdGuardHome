import { createAction } from 'redux-actions';
import Api from '../api/Api';

const apiClient = new Api();

export const addErrorToast = createAction('ADD_ERROR_TOAST');
export const addSuccessToast = createAction('ADD_SUCCESS_TOAST');
export const removeToast = createAction('REMOVE_TOAST');
export const nextStep = createAction('NEXT_STEP');
export const prevStep = createAction('PREV_STEP');

export const getDefaultAddressesRequest = createAction('GET_DEFAULT_ADDRESSES_REQUEST');
export const getDefaultAddressesFailure = createAction('GET_DEFAULT_ADDRESSES_FAILURE');
export const getDefaultAddressesSuccess = createAction('GET_DEFAULT_ADDRESSES_SUCCESS');

export const getDefaultAddresses = () => async (dispatch) => {
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

export const setAllSettings = values => async (dispatch) => {
    dispatch(setAllSettingsRequest());
    try {
        const {
            web,
            dns,
            username,
            password,
        } = values;

        const config = {
            web,
            dns,
            username,
            password,
        };

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
