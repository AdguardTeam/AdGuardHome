import { createAction } from 'redux-actions';

import { addErrorToast } from './index';
import apiClient from '../api/Api';

export const processLoginRequest = createAction('PROCESS_LOGIN_REQUEST');
export const processLoginFailure = createAction('PROCESS_LOGIN_FAILURE');
export const processLoginSuccess = createAction('PROCESS_LOGIN_SUCCESS');

export const processLogin = values => async (dispatch) => {
    dispatch(processLoginRequest());
    try {
        await apiClient.login(values);
        window.location.replace(window.location.origin);
        dispatch(processLoginSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(processLoginFailure());
    }
};
