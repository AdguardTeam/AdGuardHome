import { createAction } from 'redux-actions';
import type { AppDispatch } from 'panel/store/types';
import { apiClient } from '../api/Api';
import { addErrorToast } from './toasts';

type BlockedServicesUpdate = {
    ids: string[];
    schedule?: unknown;
};

export const getBlockedServicesRequest = createAction('GET_BLOCKED_SERVICES_REQUEST');
export const getBlockedServicesFailure = createAction('GET_BLOCKED_SERVICES_FAILURE');
export const getBlockedServicesSuccess = createAction('GET_BLOCKED_SERVICES_SUCCESS');

export const getBlockedServices = () => async (dispatch: AppDispatch) => {
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

export const getAllBlockedServices = () => async (dispatch: AppDispatch) => {
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

export const updateBlockedServices =
    (values: BlockedServicesUpdate) => async (dispatch: AppDispatch) => {
        dispatch(updateBlockedServicesRequest());
        try {
            await apiClient.updateBlockedServices(values);
            dispatch(updateBlockedServicesSuccess());
            dispatch(getBlockedServices());
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(updateBlockedServicesFailure());
        }
    };

export const allowBlockedService =
    (serviceId: string) => async (dispatch: AppDispatch, getState: any) => {
        let list = getState().services?.list;

        if (!Array.isArray(list?.ids)) {
            await dispatch(getBlockedServices());
            list = getState().services?.list;
        }

        const currentIds = Array.isArray(list?.ids) ? list.ids : [];

        if (!currentIds.includes(serviceId)) {
            return;
        }

        await dispatch(
            updateBlockedServices({
                ids: currentIds.filter((id: string) => id !== serviceId),
                schedule: list?.schedule,
            }),
        );
    };
