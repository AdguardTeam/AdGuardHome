import { combineReducers } from 'redux';

import { handleActions } from 'redux-actions';

import * as actions from '../actions/login';
import toasts from './toasts';

const login = handleActions(
    {
        [actions.processLoginRequest.toString()]: (state: any) => ({
            ...state,
            processingLogin: true,
            error: null,
        }),
        [actions.processLoginFailure.toString()]: (state: any, { payload }: any) => ({
            ...state,
            processingLogin: false,
            error: payload || true,
        }),
        [actions.processLoginSuccess.toString()]: (state, { payload }: any) => ({
            ...state,
            ...payload,
            processingLogin: false,
            error: null,
        }),
    },
    {
        processingLogin: false,
        email: '',
        password: '',
        error: null,
    },
);

export default combineReducers({
    login,
    toasts,
});
