import { handleActions } from 'redux-actions';
import * as actions from '../actions';
import { enrichWithConcatenatedIpAddresses } from '../helpers/helpers';

const dhcp = handleActions(
    {
        [actions.getDhcpStatusRequest]: (state) => ({
            ...state,
            processing: true,
        }),
        [actions.getDhcpStatusFailure]: (state) => ({
            ...state,
            processing: false,
        }),
        [actions.getDhcpStatusSuccess]: (state, { payload }) => {
            const { static_leases: staticLeases, ...values } = payload;

            const newState = {
                ...state,
                staticLeases,
                processing: false,
                ...values,
            };

            return newState;
        },

        [actions.getDhcpInterfacesRequest]: (state) => ({
            ...state,
            processingInterfaces: true,
        }),
        [actions.getDhcpInterfacesFailure]: (state) => ({
            ...state,
            processingInterfaces: false,
        }),
        [actions.getDhcpInterfacesSuccess]: (state, { payload }) => {
            const newState = {
                ...state,
                interfaces: enrichWithConcatenatedIpAddresses(payload),
                processingInterfaces: false,
            };
            return newState;
        },

        [actions.findActiveDhcpRequest]: (state) => ({
            ...state,
            processingStatus: true,
        }),
        [actions.findActiveDhcpFailure]: (state) => ({
            ...state,
            processingStatus: false,
        }),
        [actions.findActiveDhcpSuccess]: (state, { payload }) => {
            const newState = {
                ...state,
                check: payload,
                processingStatus: false,
            };
            return newState;
        },

        [actions.toggleDhcpRequest]: (state) => ({
            ...state,
            processingDhcp: true,
        }),
        [actions.toggleDhcpFailure]: (state) => ({
            ...state,
            processingDhcp: false,
        }),
        [actions.toggleDhcpSuccess]: (state) => {
            const { enabled } = state;
            const newState = {
                ...state,
                enabled: !enabled,
                check: null,
                processingDhcp: false,
            };
            return newState;
        },

        [actions.setDhcpConfigRequest]: (state) => ({
            ...state,
            processingConfig: true,
        }),
        [actions.setDhcpConfigFailure]: (state) => ({
            ...state,
            processingConfig: false,
        }),
        [actions.setDhcpConfigSuccess]: (state, { payload }) => {
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

        [actions.resetDhcpRequest]: (state) => ({
            ...state,
            processingReset: true,
        }),
        [actions.resetDhcpFailure]: (state) => ({
            ...state,
            processingReset: false,
        }),
        [actions.resetDhcpSuccess]: (state) => ({
            ...state,
            processingReset: false,
            enabled: false,
            v4: {},
            v6: {},
            interface_name: '',
        }),
        [actions.resetDhcpLeasesSuccess]: (state) => ({
            ...state,
            leases: [],
            staticLeases: [],
        }),

        [actions.toggleLeaseModal]: (state, { payload }) => {
            const newState = {
                ...state,
                isModalOpen: !state.isModalOpen,
                modalType: payload?.type || '',
                leaseModalConfig: payload?.config,
            };
            return newState;
        },

        [actions.addStaticLeaseRequest]: (state) => ({
            ...state,
            processingAdding: true,
        }),
        [actions.addStaticLeaseFailure]: (state) => ({
            ...state,
            processingAdding: false,
        }),
        [actions.addStaticLeaseSuccess]: (state, { payload }) => {
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

        [actions.removeStaticLeaseRequest]: (state) => ({
            ...state,
            processingDeleting: true,
        }),
        [actions.removeStaticLeaseFailure]: (state) => ({
            ...state,
            processingDeleting: false,
        }),
        [actions.removeStaticLeaseSuccess]: (state, { payload }) => {
            const leaseToRemove = payload.ip;
            const leases = state.staticLeases.filter((item) => item.ip !== leaseToRemove);
            const newState = {
                ...state,
                staticLeases: leases,
                processingDeleting: false,
            };
            return newState;
        },

        [actions.updateStaticLeaseRequest]: (state) => ({ ...state, processingUpdating: true }),
        [actions.updateStaticLeaseFailure]: (state) => ({ ...state, processingUpdating: false }),
        [actions.updateStaticLeaseSuccess]: (state) => {
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
