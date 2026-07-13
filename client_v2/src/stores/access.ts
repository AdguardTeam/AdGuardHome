import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { apiClient } from 'panel/api/Api';
import { addErrorToast, addSuccessToast } from './toasts';
import { splitByNewLine } from 'panel/helpers/helpers';
import intl from 'panel/common/intl';

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
        addSuccessToast(intl.getMessage('settings_notify_changes_saved'));
    } catch (error) {
        addErrorToast({ error });
        setState('processingSet', false);
    }
};

type AccessList = {
    allowed_clients?: string[];
    disallowed_clients?: string[];
    blocked_hosts?: string[];
};

const addUnique = (items: string[], value: string) =>
    items.includes(value) ? items : items.concat(value);
const removeValue = (items: string[], value: string) => items.filter((i) => i !== value);

const getNextClientAccessList = ({
    accessList,
    ip,
    disallowed,
    disallowedRule,
}: {
    accessList: AccessList;
    ip: string;
    disallowed: boolean;
    disallowedRule: string;
}) => {
    const values = {
        blocked_hosts: accessList.blocked_hosts ?? [],
        allowed_clients: accessList.allowed_clients ?? [],
        disallowed_clients: accessList.disallowed_clients ?? [],
    };
    const isAllowlistMode = values.allowed_clients.length > 0;

    if (disallowed && isAllowlistMode) {
        return {
            ...values,
            allowed_clients: addUnique(values.allowed_clients, ip),
        };
    }
    if (disallowed) {
        return {
            ...values,
            disallowed_clients: removeValue(values.disallowed_clients, disallowedRule || ip),
        };
    }
    if (isAllowlistMode) {
        return {
            ...values,
            allowed_clients: removeValue(values.allowed_clients, ip),
        };
    }
    return {
        ...values,
        disallowed_clients: addUnique(values.disallowed_clients, ip),
    };
};

export const toggleClientBlock = async (
    ip: string,
    disallowed: boolean,
    disallowedRule: string,
) => {
    setState('processingSet', true);
    try {
        const accessList: AccessList = await apiClient.getAccessList();
        const values = getNextClientAccessList({
            accessList,
            ip,
            disallowed,
            disallowedRule,
        });
        await apiClient.setAccessList(values);
        setState({
            allowed_clients: values.allowed_clients.join('\n'),
            disallowed_clients: values.disallowed_clients.join('\n'),
            blocked_hosts: values.blocked_hosts.join('\n'),
            processingSet: false,
        });
        addSuccessToast(
            disallowed
                ? intl.getMessage('client_unblocked_flash')
                : intl.getMessage('client_blocked_flash'),
        );
    } catch (error) {
        addErrorToast({ error });
        setState('processingSet', false);
    }
};

export const accessState = untrack(() => state);
