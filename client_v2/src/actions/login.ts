import { createAction } from 'redux-actions';

import apiClient from '../api/Api';
import { addErrorToast } from './toasts';
import { HTML_PAGES } from '../helpers/constants';

export const processLoginRequest = createAction('PROCESS_LOGIN_REQUEST');
export const processLoginFailure = createAction('PROCESS_LOGIN_FAILURE');
export const processLoginSuccess = createAction('PROCESS_LOGIN_SUCCESS');

export const processLogin = (values: any) => async (dispatch: any) => {
    dispatch(processLoginRequest());
    try {
        await apiClient.login(values);
        const dashboardUrl =
            window.location.origin + window.location.pathname.replace(HTML_PAGES.LOGIN, HTML_PAGES.MAIN);
        window.location.replace(dashboardUrl);
        dispatch(processLoginSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(processLoginFailure());
    }
};
