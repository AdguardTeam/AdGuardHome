import { createAction } from 'redux-actions';
import apiClient from '../api/Api';

import { redirectToCurrentProtocol } from '../helpers/helpers';
import { addErrorToast, addSuccessToast } from './toasts';

export const getTlsStatusRequest = createAction('GET_TLS_STATUS_REQUEST');
export const getTlsStatusFailure = createAction('GET_TLS_STATUS_FAILURE');
export const getTlsStatusSuccess = createAction('GET_TLS_STATUS_SUCCESS');

export const getTlsStatus = () => async (dispatch: any) => {
    dispatch(getTlsStatusRequest());
    try {
        const status = await apiClient.getTlsStatus();
        status.certificate_chain = atob(status.certificate_chain);
        status.private_key = atob(status.private_key);

        dispatch(getTlsStatusSuccess(status));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getTlsStatusFailure());
    }
};

export const setTlsConfigRequest = createAction('SET_TLS_CONFIG_REQUEST');
export const setTlsConfigFailure = createAction('SET_TLS_CONFIG_FAILURE');
export const setTlsConfigSuccess = createAction('SET_TLS_CONFIG_SUCCESS');
export const dnsStatusSuccess = createAction('DNS_STATUS_SUCCESS');

export const setTlsConfig = (config: any) => async (dispatch: any, getState: any) => {
    dispatch(setTlsConfigRequest());
    try {
        const { httpPort } = getState().dashboard;
        const values = { ...config };
        values.certificate_chain = btoa(values.certificate_chain);
        values.private_key = btoa(values.private_key);
        values.port_https = values.port_https || 0;
        values.port_dns_over_tls = values.port_dns_over_tls || 0;
        values.port_dns_over_quic = values.port_dns_over_quic || 0;

        const response = await apiClient.setTlsConfig(values);
        response.certificate_chain = atob(response.certificate_chain);
        response.private_key = atob(response.private_key);

        if (values.enabled && values.force_https && window.location.protocol === 'http:') {
            window.location.reload();
            return;
        }
        redirectToCurrentProtocol(response, httpPort);

        const dnsStatus = await apiClient.getGlobalStatus();
        if (dnsStatus) {
            if (dnsStatus.protection_disabled_duration === 0) {
                dnsStatus.protection_disabled_duration = null;
            }
            dispatch(dnsStatusSuccess(dnsStatus));
        }

        dispatch(setTlsConfigSuccess(response));
        dispatch(addSuccessToast('encryption_config_saved'));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setTlsConfigFailure());
    }
};

export const validateTlsConfigRequest = createAction('VALIDATE_TLS_CONFIG_REQUEST');
export const validateTlsConfigFailure = createAction('VALIDATE_TLS_CONFIG_FAILURE');
export const validateTlsConfigSuccess = createAction('VALIDATE_TLS_CONFIG_SUCCESS');

export const validateTlsConfig = (config: any) => async (dispatch: any) => {
    dispatch(validateTlsConfigRequest());
    try {
        const values = { ...config };
        values.certificate_chain = btoa(values.certificate_chain);
        values.private_key = btoa(values.private_key);
        values.port_https = values.port_https || 0;
        values.port_dns_over_tls = values.port_dns_over_tls || 0;
        values.port_dns_over_quic = values.port_dns_over_quic || 0;

        const response = await apiClient.validateTlsConfig(values);
        response.certificate_chain = atob(response.certificate_chain);
        response.private_key = atob(response.private_key);
        dispatch(validateTlsConfigSuccess(response));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(validateTlsConfigFailure());
    }
};
