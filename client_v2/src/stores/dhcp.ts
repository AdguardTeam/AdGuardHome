import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import {
    status,
    dhcpStatus,
    dhcpInterfaces,
    checkActiveDhcp,
    dhcpSetConfig,
    dhcpReset,
    dhcpResetLeases,
    dhcpAddStaticLease,
    dhcpRemoveStaticLease,
    dhcpUpdateStaticLease,
} from 'panel/api/generated';
import { addErrorToast, addSuccessToast } from './toasts';
import intl from 'panel/common/intl';
import { STATUS_RESPONSE } from 'panel/helpers/constants';
import { Paths } from 'panel/components/Routes/Paths';

import type { DhcpStaticLease } from 'panel/api/model/dhcpStaticLease';
import type { DhcpSearchResult } from 'panel/api/model/dhcpSearchResult';
import type { DHCPNetInterfaces } from 'panel/api/model/dHCPNetInterfaces';
import type { DhcpConfig } from 'panel/api/model/dhcpConfig';

type Lease = { hostname: string; ip: string; mac: string };

export type LeaseModalType = 'ADD_LEASE' | 'EDIT_LEASE' | 'MAKE_STATIC';

type DhcpState = {
    processing: boolean;
    processingStatus: boolean;
    processingInterfaces: boolean;
    processingDhcp: boolean;
    processingConfig: boolean;
    processingAdding: boolean;
    processingDeleting: boolean;
    processingUpdating: boolean;
    processingReset: boolean;
    enabled: boolean;
    interface_name: string;
    check: DhcpSearchResult | null;
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
    staticLeases: DhcpStaticLease[];
    isModalOpen: boolean;
    leaseModalConfig: Lease | undefined;
    modalType: LeaseModalType | '';
    dhcp_available: boolean;
    staticIpError: boolean;
    interfaces?: DHCPNetInterfaces;
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
    processingReset: false,
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
    interfaces: {},
};

const [state, setState] = createStore<DhcpState>(initialState);

export const getDhcpStatus = async () => {
    setState('processingStatus', true);
    try {
        const globalStatus = await status();
        if (globalStatus.dhcp_available) {
            const statusData = await dhcpStatus();
            const { static_leases: staticLeases, ...values } = statusData;
            setState({
                enabled: values.enabled ?? false,
                interface_name: values.interface_name ?? '',
                v4: {
                    gateway_ip: values.v4?.gateway_ip ?? '',
                    subnet_mask: values.v4?.subnet_mask ?? '',
                    range_start: values.v4?.range_start ?? '',
                    range_end: values.v4?.range_end ?? '',
                    lease_duration: values.v4?.lease_duration ?? 0,
                },
                v6: {
                    range_start: values.v6?.range_start ?? '',
                    lease_duration: values.v6?.lease_duration ?? 0,
                },
                leases: values.leases.map((l) => ({ hostname: l.hostname, ip: l.ip, mac: l.mac })),
                staticLeases: (staticLeases ?? []).map((l) => ({
                    hostname: l.hostname,
                    ip: l.ip,
                    mac: l.mac,
                })),
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
        const data = await dhcpInterfaces();
        setState({
            interfaces: data,
            processingInterfaces: false,
        });
    } catch (error) {
        addErrorToast({ error });
        setState('processingInterfaces', false);
    }
};

export const findActiveDhcp = async (interfaceName: string, navigate?: (path: string) => void) => {
    setState('processingDhcp', true);
    try {
        const data = await checkActiveDhcp({ interface: interfaceName });
        setState({ check: data, processingDhcp: false });

        const cur = untrack(() => state);
        const v4 = cur.check?.v4 ?? { static_ip: {}, other_server: {} };
        const v6 = cur.check?.v6 ?? { other_server: {} };
        const interfaces = cur.interfaces ?? {};
        const interfaceNameState = cur.interface_name;

        let isError = false;
        let isStaticIPError = false;
        const hasV4 = !!interfaces[interfaceName]?.ipv4_addresses;
        const hasV6 = !!interfaces[interfaceName]?.ipv6_addresses;

        if (hasV4 && v4.other_server.found === STATUS_RESPONSE.ERROR) {
            isError = true;
            if (v4.other_server.error) addErrorToast({ error: v4.other_server.error });
        }
        if (hasV6 && v6.other_server.found === STATUS_RESPONSE.ERROR) {
            isError = true;
            if (v6.other_server.error) addErrorToast({ error: v6.other_server.error });
        }
        if (hasV4 && v4.static_ip.static === STATUS_RESPONSE.ERROR) {
            isStaticIPError = true;
            addErrorToast({
                error: intl.getMessage('dhcp_static_ip_error'),
                action: {
                    text: intl.getMessage('set_static_ip_manually'),
                    callback: () => navigate?.(Paths.DhcpLeases),
                },
            });
        }
        if (isError) {
            addErrorToast({
                error: intl.getMessage('dhcp_error'),
                action: {
                    text: intl.getMessage('try_again'),
                    callback: () => findActiveDhcp(interfaceName, navigate),
                },
            });
        }
        if (isStaticIPError || isError) return;

        if (
            (hasV4 && v4.other_server.found === STATUS_RESPONSE.YES) ||
            (hasV6 && v6.other_server.found === STATUS_RESPONSE.YES)
        ) {
            addErrorToast({
                error: intl.getMessage('dhcp_found'),
                action: {
                    text: intl.getMessage('try_again'),
                    callback: () => findActiveDhcp(interfaceName, navigate),
                },
            });
        } else if (
            hasV4 &&
            v4.static_ip.static === STATUS_RESPONSE.NO &&
            v4.static_ip.ip &&
            interfaceNameState
        ) {
            addErrorToast({
                error: intl.getMessage('dhcp_dynamic_ip_found', {
                    interface_name: interfaceName,
                    ip: v4.static_ip.ip,
                }),
                action: {
                    text: intl.getMessage('try_again'),
                    callback: () => findActiveDhcp(interfaceName, navigate),
                },
            });
        } else {
            addSuccessToast(intl.getMessage('dhcp_not_found'));
        }
    } catch (error) {
        addErrorToast({ error });
        setState('processingDhcp', false);
    }
};

export const setDhcpConfig = async (values: DhcpConfig) => {
    setState('processingConfig', true);
    try {
        await dhcpSetConfig(values);
        const cur = untrack(() => state);
        setState({
            v4: { ...cur.v4, ...values.v4 },
            v6: { ...cur.v6, ...values.v6 },
            interface_name: values.interface_name ?? cur.interface_name,
            processingConfig: false,
        });
        addSuccessToast(intl.getMessage('dhcp_config_saved'));
    } catch (error) {
        addErrorToast({ error });
        setState('processingConfig', false);
    }
};

export const toggleDhcp = async (config?: DhcpConfig) => {
    setState('processingConfig', true);
    try {
        const values = config || {};
        const enabled = !values.enabled;
        const payload = { ...values, enabled };
        await dhcpSetConfig(payload);
        setState({ enabled, check: null, processingConfig: false });
    } catch (error) {
        addErrorToast({ error });
        setState('processingConfig', false);
    }
};

export const resetDhcp = async () => {
    setState('processingReset', true);
    try {
        await dhcpReset();
        await getDhcpStatus();
        setState('processingReset', false);
        addSuccessToast(intl.getMessage('dhcp_config_saved'));
    } catch (error) {
        addErrorToast({ error });
        setState('processingReset', false);
    }
};

export const resetDhcpLeases = async () => {
    setState('processingReset', true);
    try {
        await dhcpResetLeases();
        await getDhcpStatus();
        setState('processingReset', false);
        addSuccessToast(intl.getMessage('dhcp_reset_leases_success'));
    } catch (error) {
        addErrorToast({ error });
        setState('processingReset', false);
    }
};

export const toggleLeaseModal = (modalType?: LeaseModalType, leaseConfig?: Lease) => {
    setState({
        isModalOpen: !state.isModalOpen,
        modalType: modalType || '',
        leaseModalConfig: leaseConfig,
    });
};

export const addStaticLease = async (lease: Lease) => {
    setState('processingAdding', true);
    try {
        const name = lease.hostname || lease.ip;
        await dhcpAddStaticLease(lease);
        setState('processingAdding', false);
        toggleLeaseModal();
        addSuccessToast(intl.getMessage('dhcp_lease_added', { key: name }));
        await getDhcpStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingAdding', false);
    }
};

export const removeStaticLease = async (lease: Lease) => {
    setState('processingDeleting', true);
    try {
        const name = lease.hostname || lease.ip;
        await dhcpRemoveStaticLease(lease);
        setState('processingDeleting', false);
        addSuccessToast(intl.getMessage('dhcp_lease_deleted', { key: name }));
        await getDhcpStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingDeleting', false);
    }
};

export const updateStaticLease = async (lease: Lease) => {
    setState('processingUpdating', true);
    try {
        await dhcpUpdateStaticLease(lease);
        setState('processingUpdating', false);
        toggleLeaseModal();
        addSuccessToast(
            intl.getMessage('dhcp_lease_updated', {
                key: lease.hostname || lease.ip,
            }),
        );
        await getDhcpStatus();
    } catch (error) {
        addErrorToast({ error });
        setState('processingUpdating', false);
    }
};

export const dhcpState = untrack(() => state);
