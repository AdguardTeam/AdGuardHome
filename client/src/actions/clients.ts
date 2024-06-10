import { createAction } from 'redux-actions';
import i18next from 'i18next';
import apiClient from '../api/Api';

import { getClients } from './index';
import { addErrorToast, addSuccessToast } from './toasts';

export const toggleClientModal = createAction('TOGGLE_CLIENT_MODAL');

export const addClientRequest = createAction('ADD_CLIENT_REQUEST');
export const addClientFailure = createAction('ADD_CLIENT_FAILURE');
export const addClientSuccess = createAction('ADD_CLIENT_SUCCESS');

export const addClient = (config: any) => async (dispatch: any) => {
    dispatch(addClientRequest());
    try {
        await apiClient.addClient(config);
        dispatch(addClientSuccess());
        dispatch(toggleClientModal());
        dispatch(addSuccessToast(i18next.t('client_added', { key: config.name })));
        dispatch(getClients());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(addClientFailure());
    }
};

export const deleteClientRequest = createAction('DELETE_CLIENT_REQUEST');
export const deleteClientFailure = createAction('DELETE_CLIENT_FAILURE');
export const deleteClientSuccess = createAction('DELETE_CLIENT_SUCCESS');

export const deleteClient = (config: any) => async (dispatch: any) => {
    dispatch(deleteClientRequest());
    try {
        await apiClient.deleteClient(config);
        dispatch(deleteClientSuccess());
        dispatch(addSuccessToast(i18next.t('client_deleted', { key: config.name })));
        dispatch(getClients());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(deleteClientFailure());
    }
};

export const updateClientRequest = createAction('UPDATE_CLIENT_REQUEST');
export const updateClientFailure = createAction('UPDATE_CLIENT_FAILURE');
export const updateClientSuccess = createAction('UPDATE_CLIENT_SUCCESS');

export const updateClient = (config: any, name: any) => async (dispatch: any) => {
    dispatch(updateClientRequest());
    try {
        const data = { name, data: { ...config } };

        await apiClient.updateClient(data);
        dispatch(updateClientSuccess());
        dispatch(toggleClientModal());
        dispatch(addSuccessToast(i18next.t('client_updated', { key: name })));
        dispatch(getClients());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(updateClientFailure());
    }
};
