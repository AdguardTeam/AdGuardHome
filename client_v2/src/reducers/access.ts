import { handleActions } from 'redux-actions';

import * as actions from '../actions/access';

const access = handleActions(
    {
        [actions.getAccessListRequest.toString()]: (state: any) => ({
            ...state,
            processing: true,
        }),
        [actions.getAccessListFailure.toString()]: (state: any) => ({
            ...state,
            processing: false,
        }),
        [actions.getAccessListSuccess.toString()]: (state: any, { payload }: any) => {
            const { allowed_clients, disallowed_clients, blocked_hosts } = payload;
            const newState = {
                ...state,
                allowed_clients: allowed_clients?.join('\n') || '',
                disallowed_clients: disallowed_clients?.join('\n') || '',
                blocked_hosts: blocked_hosts?.join('\n') || '',
                processing: false,
            };
            return newState;
        },

        [actions.setAccessListRequest.toString()]: (state: any) => ({
            ...state,
            processingSet: true,
        }),
        [actions.setAccessListFailure.toString()]: (state: any) => ({
            ...state,
            processingSet: false,
        }),
        [actions.setAccessListSuccess.toString()]: (state: any) => ({
            ...state,
            processingSet: false,
        }),

        [actions.toggleClientBlockRequest.toString()]: (state: any) => ({
            ...state,
            processingSet: true,
        }),
        [actions.toggleClientBlockFailure.toString()]: (state: any) => ({
            ...state,
            processingSet: false,
        }),
        [actions.toggleClientBlockSuccess.toString()]: (state: any, { payload }: any) => {
            const { allowed_clients, disallowed_clients, blocked_hosts } = payload;
            const newState = {
                ...state,
                allowed_clients: allowed_clients?.join('\n') || '',
                disallowed_clients: disallowed_clients?.join('\n') || '',
                blocked_hosts: blocked_hosts?.join('\n') || '',
                processingSet: false,
            };
            return newState;
        },
    },
    {
        processing: true,
        processingSet: false,
        allowed_clients: '',
        disallowed_clients: '',
        blocked_hosts: '',
    },
);

export default access;
