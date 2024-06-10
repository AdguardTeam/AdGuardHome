import { handleActions } from 'redux-actions';

import * as actions from '../actions';

import { enrichWithConcatenatedIpAddresses } from '../helpers/helpers';

const dhcp = handleActions(
    {
        [actions.getDhcpStatusRequest.toString()]: (state: any) => ({
            ...state,
            processing: true,
        }),
        [actions.getDhcpStatusFailure.toString()]: (state: any) => ({
            ...state,
            processing: false,
        }),
        [actions.getDhcpStatusSuccess.toString()]: (state: any, { payload }: any) => {
            const { static_leases: staticLeases, ...values } = payload;

            const newState = {
                ...state,
                staticLeases,
                processing: false,
                ...values,
            };

            return newState;
        },

        [actions.getDhcpInterfacesRequest.toString()]: (state: any) => ({
            ...state,
            processingInterfaces: true,
        }),
        [actions.getDhcpInterfacesFailure.toString()]: (state: any) => ({
            ...state,
            processingInterfaces: false,
        }),
        [actions.getDhcpInterfacesSuccess.toString()]: (state: any, { payload }: any) => {
            const newState = {
                ...state,
                interfaces: enrichWithConcatenatedIpAddresses(payload),
                processingInterfaces: false,
            };
            return newState;
        },

        [actions.findActiveDhcpRequest.toString()]: (state: any) => ({
            ...state,
            processingStatus: true,
        }),
        [actions.findActiveDhcpFailure.toString()]: (state: any) => ({
            ...state,
            processingStatus: false,
        }),
        [actions.findActiveDhcpSuccess.toString()]: (state: any, { payload }: any) => {
            const newState = {
                ...state,
                check: payload,
                processingStatus: false,
            };
            return newState;
        },

        [actions.toggleDhcpRequest.toString()]: (state: any) => ({
            ...state,
            processingDhcp: true,
        }),
        [actions.toggleDhcpFailure.toString()]: (state: any) => ({
            ...state,
            processingDhcp: false,
        }),
        [actions.toggleDhcpSuccess.toString()]: (state: any) => {
            const { enabled } = state;
            const newState = {
                ...state,
                enabled: !enabled,
                check: null,
                processingDhcp: false,
            };
            return newState;
        },

        [actions.setDhcpConfigRequest.toString()]: (state: any) => ({
            ...state,
            processingConfig: true,
        }),
        [actions.setDhcpConfigFailure.toString()]: (state: any) => ({
            ...state,
            processingConfig: false,
        }),
        [actions.setDhcpConfigSuccess.toString()]: (state: any, { payload }: any) => {
            const { v4, v6 } = state;
            const newConfigV4 = { ...v4, ...payload.v4 };
            const newConfigV6 = { ...v6, ...payload.v6 };

            const newState = {
                ...state,
                v4: newConfigV4,
                v6: newConfigV6,
                interface_name: payload.interface_name,
                processingConfig: false,
            };

            return newState;
        },

        [actions.resetDhcpRequest.toString()]: (state: any) => ({
            ...state,
            processingReset: true,
        }),
        [actions.resetDhcpFailure.toString()]: (state: any) => ({
            ...state,
            processingReset: false,
        }),
        [actions.resetDhcpSuccess.toString()]: (state: any) => ({
            ...state,
            processingReset: false,
            enabled: false,
            v4: {},
            v6: {},
            interface_name: '',
        }),
        [actions.resetDhcpLeasesSuccess.toString()]: (state: any) => ({
            ...state,
            leases: [],
            staticLeases: [],
        }),

        [actions.toggleLeaseModal.toString()]: (state: any, { payload }: any) => {
            const newState = {
                ...state,
                isModalOpen: !state.isModalOpen,
                modalType: payload?.type || '',
                leaseModalConfig: payload?.config,
            };
            return newState;
        },

        [actions.addStaticLeaseRequest.toString()]: (state: any) => ({
            ...state,
            processingAdding: true,
        }),
        [actions.addStaticLeaseFailure.toString()]: (state: any) => ({
            ...state,
            processingAdding: false,
        }),
        [actions.addStaticLeaseSuccess.toString()]: (state: any, { payload }: any) => {
            const { ip, mac, hostname } = payload;
            const newLease = {
                ip,
                mac,
                hostname: hostname || '',
            };
            const leases = [...state.staticLeases, newLease];
            const newState = {
                ...state,
                staticLeases: leases,
                processingAdding: false,
            };
            return newState;
        },

        [actions.removeStaticLeaseRequest.toString()]: (state: any) => ({
            ...state,
            processingDeleting: true,
        }),
        [actions.removeStaticLeaseFailure.toString()]: (state: any) => ({
            ...state,
            processingDeleting: false,
        }),
        [actions.removeStaticLeaseSuccess.toString()]: (state: any, { payload }: any) => {
            const leaseToRemove = payload.ip;
            const leases = state.staticLeases.filter((item: any) => item.ip !== leaseToRemove);
            const newState = {
                ...state,
                staticLeases: leases,
                processingDeleting: false,
            };
            return newState;
        },

        [actions.updateStaticLeaseRequest.toString()]: (state: any) => ({
            ...state,
            processingUpdating: true,
        }),
        [actions.updateStaticLeaseFailure.toString()]: (state: any) => ({
            ...state,
            processingUpdating: false,
        }),
        [actions.updateStaticLeaseSuccess.toString()]: (state: any) => {
            const newState = {
                ...state,
                processingUpdating: false,
            };
            return newState;
        },
    },
    {
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
    },
);

export default dhcp;
