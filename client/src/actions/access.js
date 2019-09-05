import { createAction } from 'redux-actions';
import apiClient from '../api/Api';
import { addErrorToast, addSuccessToast } from './index';
import { normalizeTextarea } from '../helpers/helpers';

export const getAccessListRequest = createAction('GET_ACCESS_LIST_REQUEST');
export const getAccessListFailure = createAction('GET_ACCESS_LIST_FAILURE');
export const getAccessListSuccess = createAction('GET_ACCESS_LIST_SUCCESS');

export const getAccessList = () => async (dispatch) => {
    dispatch(getAccessListRequest());
    try {
        const data = await apiClient.getAccessList();
        dispatch(getAccessListSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getAccessListFailure());
    }
};

export const setAccessListRequest = createAction('SET_ACCESS_LIST_REQUEST');
export const setAccessListFailure = createAction('SET_ACCESS_LIST_FAILURE');
export const setAccessListSuccess = createAction('SET_ACCESS_LIST_SUCCESS');

export const setAccessList = config => async (dispatch) => {
    dispatch(setAccessListRequest());
    try {
        const { allowed_clients, disallowed_clients, blocked_hosts } = config;

        const values = {
            allowed_clients: (allowed_clients && normalizeTextarea(allowed_clients)) || [],
            disallowed_clients: (disallowed_clients && normalizeTextarea(disallowed_clients)) || [],
            blocked_hosts: (blocked_hosts && normalizeTextarea(blocked_hosts)) || [],
        };

        await apiClient.setAccessList(values);
        dispatch(setAccessListSuccess());
        dispatch(addSuccessToast('access_settings_saved'));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setAccessListFailure());
    }
};
