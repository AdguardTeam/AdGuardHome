import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { apiClient } from 'panel/api/Api';
import { addErrorToast, addSuccessToast } from './toasts';
import intl from 'panel/common/intl';
import { getClients } from './dashboard';

type ClientsState = {
    processing: boolean;
    processingAdding: boolean;
    processingDeleting: boolean;
    processingUpdating: boolean;
    isModalOpen: boolean;
    modalClientName: string;
    modalType: string;
};

const initialState: ClientsState = {
    processing: true,
    processingAdding: false,
    processingDeleting: false,
    processingUpdating: false,
    isModalOpen: false,
    modalClientName: '',
    modalType: '',
};

const [state, setState] = createStore<ClientsState>(initialState);

export const toggleClientModal = (payload?: { type?: string; name?: string }) => {
    if (payload) {
        setState({
            modalType: payload.type || '',
            modalClientName: payload.name || '',
            isModalOpen: !untrack(() => state.isModalOpen),
        });
    } else {
        setState('isModalOpen', (prev) => !prev);
    }
};

export const addClient = async (config: any) => {
    setState('processingAdding', true);
    try {
        await apiClient.addClient(config);
        setState('processingAdding', false);
        toggleClientModal();
        addSuccessToast(intl.getMessage('client_added'));
        await getClients();
    } catch (error) {
        addErrorToast({ error });
        setState('processingAdding', false);
    }
};

export const deleteClient = async (name: string) => {
    setState('processingDeleting', true);
    try {
        await apiClient.deleteClient({ name });
        setState('processingDeleting', false);
        addSuccessToast(intl.getMessage('client_removed'));
        await getClients();
    } catch (error) {
        addErrorToast({ error });
        setState('processingDeleting', false);
    }
};

export const updateClient = async (name: string, data: any): Promise<boolean> => {
    setState('processingUpdating', true);
    try {
        await apiClient.updateClient({ name, data });
        setState('processingUpdating', false);
        toggleClientModal();
        await getClients();
        return true;
    } catch (error) {
        addErrorToast({ error });
        setState('processingUpdating', false);
        return false;
    }
};

export const clientsState = untrack(() => state);
