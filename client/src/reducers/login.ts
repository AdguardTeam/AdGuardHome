import { combineReducers } from 'redux';

import { handleActions } from 'redux-actions';

import { reducer as formReducer } from 'redux-form';

import * as actions from '../actions/login';
import toasts from './toasts';

const login = handleActions(
    {
        [actions.processLoginRequest.toString()]: (state: any) => ({
            ...state,
            processingLogin: true,
        }),
        [actions.processLoginFailure.toString()]: (state: any) => ({
            ...state,
            processingLogin: false,
        }),
        [actions.processLoginSuccess.toString()]: (state, { payload }: any) => ({
            ...state,
            ...payload,
            processingLogin: false,
        }),
    },
    {
        processingLogin: false,
        email: '',
        password: '',
    },
);

export default combineReducers({
    login,
    toasts,
    form: formReducer,
});
