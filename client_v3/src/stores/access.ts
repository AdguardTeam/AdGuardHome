import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { apiClient } from 'panel/api/Api';
import { addErrorToast } from './toasts';
import { splitByNewLine } from 'panel/helpers/helpers';

type AccessState = {
    processing: boolean;
    processingSet: boolean;
    allowed_clients: string;
    disallowed_clients: string;
    blocked_hosts: string;
};

const initialState: AccessState = {
    processing: true,
    processingSet: false,
    allowed_clients: '',
    disallowed_clients: '',
    blocked_hosts: '',
};

const [state, setState] = createStore<AccessState>(initialState);

export const getAccessList = async () => {
    setState('processing', true);
    try {
        const data = await apiClient.getAccessList();
        setState({
            allowed_clients: data.allowed_clients?.join('\n') || '',
            disallowed_clients: data.disallowed_clients?.join('\n') || '',
            blocked_hosts: data.blocked_hosts?.join('\n') || '',
            processing: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processing', false);
    }
};

export const setAccessList = async (values: any) => {
    setState('processingSet', true);
    try {
        const config = { ...values };

        if (Object.hasOwn(config, 'allowed_clients')) {
            config.allowed_clients = splitByNewLine(config.allowed_clients);
        }
        if (Object.hasOwn(config, 'disallowed_clients')) {
            config.disallowed_clients = splitByNewLine(config.disallowed_clients);
        }
        if (Object.hasOwn(config, 'blocked_hosts')) {
            config.blocked_hosts = splitByNewLine(config.blocked_hosts);
        }

        await apiClient.setAccessList(config);
        setState({ ...values, processingSet: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processingSet', false);
    }
};

export const toggleClientBlock = async (clientName: string) => {
    setState('processingSet', true);
    try {
        const accessList = await apiClient.getAccessList();
        const isDisallowed = accessList.disallowed_clients?.includes(clientName);
        const newDisallowed = isDisallowed
            ? accessList.disallowed_clients.filter((c: string) => c !== clientName)
            : [...(accessList.disallowed_clients || []), clientName];
        const config = {
            allowed_clients: accessList.allowed_clients || [],
            disallowed_clients: newDisallowed,
            blocked_hosts: accessList.blocked_hosts || [],
        };
        await apiClient.setAccessList(config);
        setState({
            allowed_clients: config.allowed_clients.join('\n'),
            disallowed_clients: config.disallowed_clients.join('\n'),
            blocked_hosts: config.blocked_hosts.join('\n'),
            processingSet: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingSet', false);
    }
};

export const accessState = untrack(() => state);
