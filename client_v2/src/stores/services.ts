import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { apiClient } from 'panel/api/Api';
import { addErrorToast } from './toasts';

type ServicesState = {
    processing: boolean;
    processingAll: boolean;
    processingSet: boolean;
    list: any;
    allServices: any[];
    allGroups: any[];
};

const initialState: ServicesState = {
    processing: true,
    processingAll: true,
    processingSet: false,
    list: {},
    allServices: [],
    allGroups: [],
};

const [state, setState] = createStore<ServicesState>(initialState);

export const getBlockedServices = async () => {
    setState('processing', true);
    try {
        const data = await apiClient.getBlockedServices();
        setState({ list: data, processing: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processing', false);
    }
};

export const getAllBlockedServices = async () => {
    setState('processingAll', true);
    try {
        const data = await apiClient.getAllBlockedServices();
        setState({
            allServices: data.blocked_services || [],
            allGroups: data.groups || [],
            processingAll: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingAll', false);
    }
};

export const updateBlockedServices = async (values: { ids: string[]; schedule?: unknown }) => {
    setState('processingSet', true);
    try {
        await apiClient.updateBlockedServices(values);
        setState('processingSet', false);
        await getBlockedServices();
    } catch (error) {
        addErrorToast({ error });
        setState('processingSet', false);
    }
};

export const allowBlockedService = async (serviceId: string) => {
    let list = untrack(() => state.list);
    if (!Array.isArray(list?.ids)) {
        await getBlockedServices();
        list = untrack(() => state.list);
    }
    const currentIds = Array.isArray(list?.ids) ? list.ids : [];
    if (!currentIds.includes(serviceId)) return;
    await updateBlockedServices({
        ids: currentIds.filter((id: string) => id !== serviceId),
        schedule: list?.schedule,
    });
};

export const servicesState = untrack(() => state);
