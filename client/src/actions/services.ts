import { createAction } from 'redux-actions';
import apiClient from '../api/Api';
import { addErrorToast, addSuccessToast } from './toasts';

export const getBlockedServicesRequest = createAction('GET_BLOCKED_SERVICES_REQUEST');
export const getBlockedServicesFailure = createAction('GET_BLOCKED_SERVICES_FAILURE');
export const getBlockedServicesSuccess = createAction('GET_BLOCKED_SERVICES_SUCCESS');

export const getBlockedServices = () => async (dispatch: any) => {
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

export const getAllBlockedServices = () => async (dispatch: any) => {
    dispatch(getAllBlockedServicesRequest());
    try {
        const data = await apiClient.getAllBlockedServices();
        dispatch(getAllBlockedServicesSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getAllBlockedServicesFailure());
    }
};

export const updateBlockedServicesRequest = createAction('UPDATE_BLOCKED_SERVICES_REQUEST');
export const updateBlockedServicesFailure = createAction('UPDATE_BLOCKED_SERVICES_FAILURE');
export const updateBlockedServicesSuccess = createAction('UPDATE_BLOCKED_SERVICES_SUCCESS');

export const updateBlockedServices = (values: any) => async (dispatch: any) => {
    dispatch(updateBlockedServicesRequest());
    try {
        await apiClient.updateBlockedServices(values);
        dispatch(updateBlockedServicesSuccess());
        dispatch(getBlockedServices());
        dispatch(addSuccessToast('blocked_services_saved'));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(updateBlockedServicesFailure());
    }
};
