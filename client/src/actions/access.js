import { createAction } from 'redux-actions';
import i18next from 'i18next';

import apiClient from '../api/Api';
import { normalizeTextarea } from '../helpers/helpers';
import { ACTION } from '../helpers/constants';
import { addErrorToast, addSuccessToast } from './toasts';

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

export const setAccessList = (config) => async (dispatch) => {
    dispatch(setAccessListRequest());
    try {
        const { allowed_clients, disallowed_clients, blocked_hosts } = config;

        const values = {
            allowed_clients: normalizeTextarea(allowed_clients),
            disallowed_clients: normalizeTextarea(disallowed_clients),
            blocked_hosts: normalizeTextarea(blocked_hosts),
        };

        await apiClient.setAccessList(values);
        dispatch(setAccessListSuccess());
        dispatch(addSuccessToast('access_settings_saved'));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setAccessListFailure());
    }
};

export const toggleClientBlockRequest = createAction('TOGGLE_CLIENT_BLOCK_REQUEST');
export const toggleClientBlockFailure = createAction('TOGGLE_CLIENT_BLOCK_FAILURE');
export const toggleClientBlockSuccess = createAction('TOGGLE_CLIENT_BLOCK_SUCCESS');

export const toggleClientBlock = (type, ip) => async (dispatch) => {
    dispatch(toggleClientBlockRequest());
    try {
        const {
            allowed_clients, disallowed_clients, blocked_hosts,
        } = await apiClient.getAccessList();
        let updatedDisallowedClients = disallowed_clients || [];

        if (type === ACTION.unblock && updatedDisallowedClients.includes(ip)) {
            updatedDisallowedClients = updatedDisallowedClients.filter((client) => client !== ip);
        } else if (type === ACTION.block && !updatedDisallowedClients.includes(ip)) {
            updatedDisallowedClients.push(ip);
        }

        const values = {
            allowed_clients,
            blocked_hosts,
            disallowed_clients: updatedDisallowedClients,
        };

        await apiClient.setAccessList(values);
        dispatch(toggleClientBlockSuccess(values));

        if (type === ACTION.unblock) {
            dispatch(addSuccessToast(i18next.t('client_unblocked', { ip })));
        } else if (type === ACTION.block) {
            dispatch(addSuccessToast(i18next.t('client_blocked', { ip })));
        }
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(toggleClientBlockFailure());
    }
};
