import { handleActions } from 'redux-actions';

import * as actions from '../actions/encryption';

const encryption = handleActions(
    {
        [actions.getTlsStatusRequest.toString()]: (state: any) => ({
            ...state,
            processing: true,
        }),
        [actions.getTlsStatusFailure.toString()]: (state: any) => ({
            ...state,
            processing: false,
        }),
        [actions.getTlsStatusSuccess.toString()]: (state: any, { payload }: any) => {
            const newState = {
                ...state,
                ...payload,
                /* TODO: handle property delete on api refactor */
                server_name: payload.server_name || '',
                processing: false,
            };
            return newState;
        },

        [actions.setTlsConfigRequest.toString()]: (state: any) => ({
            ...state,
            processingConfig: true,
        }),
        [actions.setTlsConfigFailure.toString()]: (state: any) => ({
            ...state,
            processingConfig: false,
        }),
        [actions.setTlsConfigSuccess.toString()]: (state: any, { payload }: any) => {
            const newState = {
                ...state,
                ...payload,
                server_name: payload.server_name || '',
                processingConfig: false,
            };
            return newState;
        },

        [actions.validateTlsConfigRequest.toString()]: (state: any) => ({
            ...state,
            processingValidate: true,
        }),
        [actions.validateTlsConfigFailure.toString()]: (state: any) => ({
            ...state,
            processingValidate: false,
        }),
        [actions.validateTlsConfigSuccess.toString()]: (state: any, { payload }: any) => {
            const {
                issuer = '',
                key_type = '',
                not_after = '',
                not_before = '',
                subject = '',
                warning_validation = '',
                dns_names = '',
                ...values
            } = payload;

            const newState = {
                ...state,
                ...values,
                issuer,
                key_type,
                not_after,
                not_before,
                subject,
                warning_validation,
                dns_names,
                server_name: payload.server_name || '',
                processingValidate: false,
            };
            return newState;
        },
    },
    {
        processing: true,
        processingConfig: false,
        processingValidate: false,
        enabled: false,
        serve_plain_dns: false,
        dns_names: null,
        force_https: false,
        issuer: '',
        key_type: '',
        not_after: '',
        not_before: '',
        port_dns_over_tls: '',
        port_https: '',
        subject: '',
        valid_chain: false,
        valid_key: false,
        valid_cert: false,
        valid_pair: false,
        status_cert: '',
        status_key: '',
        certificate_chain: '',
        private_key: '',
        server_name: '',
        warning_validation: '',
        certificate_path: '',
        private_key_path: '',
    },
);

export default encryption;
