import { combineReducers } from 'redux';
import { handleActions } from 'redux-actions';
import { reducer as formReducer } from 'redux-form';

import * as actions from '../actions/login';
import toasts from './toasts';

const login = handleActions({
    [actions.processLoginRequest]: (state) => ({ ...state, processingLogin: true }),
    [actions.processLoginFailure]: (state) => ({ ...state, processingLogin: false }),
    [actions.processLoginSuccess]: (state, { payload }) => ({
        ...state, ...payload, processingLogin: false,
    }),
}, {
    processingLogin: false,
    email: '',
    password: '',
});

export default combineReducers({
    login,
    toasts,
    form: formReducer,
});
