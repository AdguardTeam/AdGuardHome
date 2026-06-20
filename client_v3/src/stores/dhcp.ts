import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { apiClient } from 'panel/api/Api';
import { addErrorToast } from './toasts';

type Lease = { hostname: string; ip: string; mac: string };

type DhcpState = {
    processing: boolean;
    processingStatus: boolean;
    processingInterfaces: boolean;
    processingDhcp: boolean;
    processingConfig: boolean;
    processingAdding: boolean;
    processingDeleting: boolean;
    processingUpdating: boolean;
    enabled: boolean;
    interface_name: string;
    check: any;
    v4: {
        gateway_ip: string;
        subnet_mask: string;
        range_start: string;
        range_end: string;
        lease_duration: number;
    };
    v6: {
        range_start: string;
        lease_duration: number;
    };
    leases: Lease[];
    staticLeases: Lease[];
    isModalOpen: boolean;
    leaseModalConfig: Lease | undefined;
    modalType: string;
    dhcp_available: boolean;
    staticIpError: boolean;
    interfaces?: Record<string, any>;
};

const initialState: DhcpState = {
    processing: true,
    processingStatus: false,
    processingInterfaces: false,
    processingDhcp: false,
    processingConfig: false,
    processingAdding: false,
    processingDeleting: false,
    processingUpdating: false,
    enabled: false,
    interface_name: '',
    check: null,
    v4: {
        gateway_ip: '',
        subnet_mask: '',
        range_start: '',
        range_end: '',
        lease_duration: 0,
    },
    v6: {
        range_start: '',
        lease_duration: 0,
    },
    leases: [],
    staticLeases: [],
    isModalOpen: false,
    leaseModalConfig: undefined,
    modalType: '',
    dhcp_available: false,
    staticIpError: false,
};

const [state, setState] = createStore<DhcpState>(initialState);

export const getDhcpStatus = async () => {
    setState('processingStatus', true);
    try {
        const globalStatus = await apiClient.getGlobalStatus();
        if (globalStatus.dhcp_available) {
            const status = await apiClient.getDhcpStatus();
            setState({
                ...status,
                dhcp_available: true,
                processingStatus: false,
                processing: false,
            });
        } else {
            setState({ dhcp_available: false, processingStatus: false, processing: false });
        }
    } catch (error) {
        addErrorToast({ error });
        setState({ processingStatus: false, processing: false });
    }
};

export const getDhcpInterfaces = async () => {
    setState('processingInterfaces', true);
    try {
        const data = await apiClient.getDhcpInterfaces();
        setState({ interfaces: data, processingInterfaces: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processingInterfaces', false);
    }
};

export const findActiveDhcp = async (interfaceName: string) => {
    setState('processingDhcp', true);
    try {
        const data = await apiClient.findActiveDhcp(interfaceName);
        setState({ check: data, processingDhcp: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processingDhcp', false);
    }
};

export const setDhcpConfig = async (values: any) => {
    setState('processingConfig', true);
    try {
        await apiClient.setDhcpConfig(values);
        setState({ ...values, processingConfig: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processingConfig', false);
    }
};

export const toggleDhcp = async (config?: any) => {
    const newEnabled = !untrack(() => state.enabled);
    setState('processingConfig', true);
    try {
        const payload = config ? { ...config, enabled: newEnabled } : { enabled: newEnabled };
        await apiClient.setDhcpConfig(payload);
        setState({ enabled: newEnabled, processingConfig: false });
        if (config && !newEnabled) {
            await getDhcpStatus();
        }
    } catch (error) {
        addErrorToast({ error });
        setState('processingConfig', false);
    }
};

export const resetDhcp = async () => {
    setState('processingDhcp', true);
    try {
        await apiClient.resetDhcp();
        await getDhcpStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingDhcp', false);
    }
};

export const resetDhcpLeases = async () => {
    setState('processingDhcp', true);
    try {
        await apiClient.resetDhcpLeases();
        await getDhcpStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingDhcp', false);
    }
};

export const toggleLeaseModal = (modalType?: string, leaseConfig?: Lease) => {
    setState({
        isModalOpen: !state.isModalOpen,
        modalType: modalType || '',
        leaseModalConfig: leaseConfig,
    });
};

export const addStaticLease = async (lease: Lease) => {
    setState('processingAdding', true);
    try {
        await apiClient.addStaticLease(lease);
        setState('processingAdding', false);
        toggleLeaseModal();
        await getDhcpStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingAdding', false);
    }
};

export const removeStaticLease = async (lease: Lease) => {
    setState('processingDeleting', true);
    try {
        await apiClient.removeStaticLease(lease);
        setState('processingDeleting', false);
        await getDhcpStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingDeleting', false);
    }
};

export const updateStaticLease = async (config: { target: Lease; update: Lease }) => {
    setState('processingUpdating', true);
    try {
        await apiClient.updateStaticLease(config);
        setState('processingUpdating', false);
        toggleLeaseModal();
        await getDhcpStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingUpdating', false);
    }
};

export const dhcpState = untrack(() => state);
