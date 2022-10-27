import { createAction } from 'redux-actions';
import apiClient from '../api/Api';
import { addErrorToast, addSuccessToast } from './toasts';

export const getBlockedServicesAvailableServicesRequest = createAction('GET_BLOCKED_SERVICES_AVAILABLE_SERVICES_REQUEST');
export const getBlockedServicesAvailableServicesFailure = createAction('GET_BLOCKED_SERVICES_AVAILABLE_SERVICES_FAILURE');
export const getBlockedServicesAvailableServicesSuccess = createAction('GET_BLOCKED_SERVICES_AVAILABLE_SERVICES_SUCCESS');

export const getBlockedServicesAvailableServices = () => async (dispatch) => {
    dispatch(getBlockedServicesAvailableServicesRequest());
    try {
        const data = await apiClient.getBlockedServicesAvailableServices();
        dispatch(getBlockedServicesAvailableServicesSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getBlockedServicesAvailableServicesFailure());
    }
};

export const getBlockedServicesRequest = createAction('GET_BLOCKED_SERVICES_REQUEST');
export const getBlockedServicesFailure = createAction('GET_BLOCKED_SERVICES_FAILURE');
export const getBlockedServicesSuccess = createAction('GET_BLOCKED_SERVICES_SUCCESS');

export const getBlockedServices = () => async (dispatch) => {
    dispatch(getBlockedServicesRequest());
    try {
        const data = await apiClient.getBlockedServices();
        dispatch(getBlockedServicesSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getBlockedServicesFailure());
    }
};

export const getAllBlockedServicesRequest = createAction('GET_ALL_BLOCKED_SERVICES_REQUEST');
export const getAllBlockedServicesFailure = createAction('GET_ALL_BLOCKED_SERVICES_FAILURE');
export const getAllBlockedServicesSuccess = createAction('GET_ALL_BLOCKED_SERVICES_SUCCESS');

export const getAllBlockedServices = () => async (dispatch) => {
    dispatch(getAllBlockedServicesRequest());
    try {
        const data = await apiClient.getAllBlockedServices();
        dispatch(getAllBlockedServicesSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getAllBlockedServicesFailure());
    }
};

export const setBlockedServicesRequest = createAction('SET_BLOCKED_SERVICES_REQUEST');
export const setBlockedServicesFailure = createAction('SET_BLOCKED_SERVICES_FAILURE');
export const setBlockedServicesSuccess = createAction('SET_BLOCKED_SERVICES_SUCCESS');

export const setBlockedServices = (values) => async (dispatch) => {
    dispatch(setBlockedServicesRequest());
    try {
        await apiClient.setBlockedServices(values);
        dispatch(setBlockedServicesSuccess());
        dispatch(getBlockedServices());
        dispatch(addSuccessToast('blocked_services_saved'));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setBlockedServicesFailure());
    }
};
