import { handleActions } from 'redux-actions';

import * as actions from '../actions/services';

const services = handleActions(
    {
        [actions.getBlockedServicesRequest]: (state) => ({ ...state, processing: true }),
        [actions.getBlockedServicesFailure]: (state) => ({ ...state, processing: false }),
        [actions.getBlockedServicesSuccess]: (state, { payload }) => ({
            ...state,
            list: payload,
            processing: false,
        }),

        [actions.getAllBlockedServicesRequest]: (state) => ({ ...state, processingAll: true }),
        [actions.getAllBlockedServicesFailure]: (state) => ({ ...state, processingAll: false }),
        [actions.getAllBlockedServicesSuccess]: (state, { payload }) => ({
            ...state,
            allServices: payload.blocked_services,
            processingAll: false,
        }),

        [actions.setBlockedServicesRequest]: (state) => ({ ...state, processingSet: true }),
        [actions.setBlockedServicesFailure]: (state) => ({ ...state, processingSet: false }),
        [actions.setBlockedServicesSuccess]: (state) => ({
            ...state,
            processingSet: false,
        }),
    },
    {
        processing: true,
        processingAll: true,
        processingSet: false,
        list: [],
        allServices: [],
    },
);

export default services;
