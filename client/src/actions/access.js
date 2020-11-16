import { createAction } from 'redux-actions';
import i18next from 'i18next';

import apiClient from '../api/Api';
import { addErrorToast, addSuccessToast } from './toasts';
import { splitByNewLine } from '../helpers/helpers';

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
            allowed_clients: splitByNewLine(allowed_clients),
            disallowed_clients: splitByNewLine(disallowed_clients),
            blocked_hosts: splitByNewLine(blocked_hosts),
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

export const toggleClientBlock = (ip, disallowed, disallowed_rule) => async (dispatch) => {
    dispatch(toggleClientBlockRequest());
    try {
        const accessList = await apiClient.getAccessList();
        const allowed_clients = accessList.allowed_clients ?? [];
        const blocked_hosts = accessList.blocked_hosts ?? [];
        const disallowed_clients = accessList.disallowed_clients ?? [];

        const updatedDisallowedClients = disallowed
            ? disallowed_clients.filter((client) => client !== disallowed_rule)
            : disallowed_clients.concat(ip);

        const values = {
            allowed_clients,
            blocked_hosts,
            disallowed_clients: updatedDisallowedClients,
        };

        await apiClient.setAccessList(values);
        dispatch(toggleClientBlockSuccess(values));

        if (disallowed) {
            dispatch(addSuccessToast(i18next.t('client_unblocked', { ip: disallowed_rule })));
        } else {
            dispatch(addSuccessToast(i18next.t('client_blocked', { ip })));
        }
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(toggleClientBlockFailure());
    }
};
