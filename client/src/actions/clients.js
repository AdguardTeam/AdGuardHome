import { createAction } from 'redux-actions';
import { t } from 'i18next';
import Api from '../api/Api';
import { addErrorToast, addSuccessToast, getClients } from './index';
import { CLIENT_ID } from '../helpers/constants';

const apiClient = new Api();

export const toggleClientModal = createAction('TOGGLE_CLIENT_MODAL');

export const addClientRequest = createAction('ADD_CLIENT_REQUEST');
export const addClientFailure = createAction('ADD_CLIENT_FAILURE');
export const addClientSuccess = createAction('ADD_CLIENT_SUCCESS');

export const addClient = config => async (dispatch) => {
    dispatch(addClientRequest());
    try {
        let data;
        if (config.identifier === CLIENT_ID.MAC) {
            const { ip, identifier, ...values } = config;

            data = { ...values };
        } else {
            const { mac, identifier, ...values } = config;

            data = { ...values };
        }

        await apiClient.addClient(data);
        dispatch(addClientSuccess());
        dispatch(toggleClientModal());
        dispatch(addSuccessToast(t('client_added', { key: config.name })));
        dispatch(getClients());
    } catch (error) {
        dispatch(toggleClientModal());
        dispatch(addErrorToast({ error }));
        dispatch(addClientFailure());
    }
};

export const deleteClientRequest = createAction('DELETE_CLIENT_REQUEST');
export const deleteClientFailure = createAction('DELETE_CLIENT_FAILURE');
export const deleteClientSuccess = createAction('DELETE_CLIENT_SUCCESS');

export const deleteClient = config => async (dispatch) => {
    dispatch(deleteClientRequest());
    try {
        await apiClient.deleteClient(config);
        dispatch(deleteClientSuccess());
        dispatch(addSuccessToast(t('client_deleted', { key: config.name })));
        dispatch(getClients());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(deleteClientFailure());
    }
};

export const updateClientRequest = createAction('UPDATE_CLIENT_REQUEST');
export const updateClientFailure = createAction('UPDATE_CLIENT_FAILURE');
export const updateClientSuccess = createAction('UPDATE_CLIENT_SUCCESS');

export const updateClient = (config, name) => async (dispatch) => {
    dispatch(updateClientRequest());
    try {
        let data;
        if (config.identifier === CLIENT_ID.MAC) {
            const { ip, identifier, ...values } = config;

            data = { name, data: { ...values } };
        } else {
            const { mac, identifier, ...values } = config;

            data = { name, data: { ...values } };
        }

        await apiClient.updateClient(data);
        dispatch(updateClientSuccess());
        dispatch(toggleClientModal());
        dispatch(addSuccessToast(t('client_updated', { key: name })));
        dispatch(getClients());
    } catch (error) {
        dispatch(toggleClientModal());
        dispatch(addErrorToast({ error }));
        dispatch(updateClientFailure());
    }
};
