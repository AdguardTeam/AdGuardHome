import { handleActions } from 'redux-actions';

import * as actions from '../actions/services';

const services = handleActions(
    {
        [actions.getBlockedServicesRequest.toString()]: (state: any) => ({
            ...state,
            processing: true,
        }),
        [actions.getBlockedServicesFailure.toString()]: (state: any) => ({
            ...state,
            processing: false,
        }),
        [actions.getBlockedServicesSuccess.toString()]: (state, { payload }: any) => ({
            ...state,
            list: payload,
            processing: false,
        }),

        [actions.getAllBlockedServicesRequest.toString()]: (state: any) => ({
            ...state,
            processingAll: true,
        }),
        [actions.getAllBlockedServicesFailure.toString()]: (state: any) => ({
            ...state,
            processingAll: false,
        }),
        [actions.getAllBlockedServicesSuccess.toString()]: (state, { payload }: any) => ({
            ...state,
            allServices: payload.blocked_services,
            allGroups: payload.groups,
            processingAll: false,
        }),

        [actions.updateBlockedServicesRequest.toString()]: (state: any) => ({
            ...state,
            processingSet: true,
        }),
        [actions.updateBlockedServicesFailure.toString()]: (state: any) => ({
            ...state,
            processingSet: false,
        }),
        [actions.updateBlockedServicesSuccess.toString()]: (state: any) => ({
            ...state,
            processingSet: false,
        }),
    },
    {
        processing: true,
        processingAll: true,
        processingSet: false,
        list: {},
        allServices: [],
        allGroups: [],
    },
);

export default services;
