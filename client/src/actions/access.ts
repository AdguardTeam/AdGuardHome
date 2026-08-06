import { createAction } from 'redux-actions';
import i18next from 'i18next';

import apiClient from '../api/Api';
import { addErrorToast, addSuccessToast } from './toasts';

import { splitByNewLine } from '../helpers/helpers';

export const getAccessListRequest = createAction('GET_ACCESS_LIST_REQUEST');
export const getAccessListFailure = createAction('GET_ACCESS_LIST_FAILURE');
export const getAccessListSuccess = createAction('GET_ACCESS_LIST_SUCCESS');

export const getAccessList = () => async (dispatch: any) => {
    dispatch(getAccessListRequest());
    try {
        const data = await apiClient.getAccessList();
        dispatch(getAccessListSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getAccessListFailure());
    }
};

export const setAccessListRequest = createAction('SET_ACCESS_LIST_REQUEST');
export const setAccessListFailure = createAction('SET_ACCESS_LIST_FAILURE');
export const setAccessListSuccess = createAction('SET_ACCESS_LIST_SUCCESS');

export const setAccessList = (config: any) => async (dispatch: any) => {
    dispatch(setAccessListRequest());
    try {
        const { allowed_clients, disallowed_clients, blocked_hosts } = config;

        const values = {
            allowed_clients: splitByNewLine(allowed_clients),
            disallowed_clients: splitByNewLine(disallowed_clients),
            blocked_hosts: splitByNewLine(blocked_hosts),
        };

        await apiClient.setAccessList(values);
        dispatch(setAccessListSuccess());
        dispatch(addSuccessToast('access_settings_saved'));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setAccessListFailure());
    }
};

export const toggleClientBlockRequest = createAction('TOGGLE_CLIENT_BLOCK_REQUEST');
export const toggleClientBlockFailure = createAction('TOGGLE_CLIENT_BLOCK_FAILURE');
export const toggleClientBlockSuccess = createAction('TOGGLE_CLIENT_BLOCK_SUCCESS');

type AccessList = {
    allowed_clients?: string[];
    disallowed_clients?: string[];
    blocked_hosts?: string[];
};

type AccessListValues = {
    allowed_clients: string[];
    disallowed_clients: string[];
    blocked_hosts: string[];
};

type GetNextClientAccessListArgs = {
    accessList: AccessList;
    ip: string;
    disallowed: boolean;
    disallowedRule: string;
};

const addUnique = (items: string[], value: string) => (items.includes(value) ? items : items.concat(value));

const removeValue = (items: string[], value: string) => items.filter((item) => item !== value);

const getNextClientAccessList = ({
    accessList,
    ip,
    disallowed,
    disallowedRule,
}: GetNextClientAccessListArgs): AccessListValues => {
    const values = {
        blocked_hosts: accessList.blocked_hosts ?? [],
        allowed_clients: accessList.allowed_clients ?? [],
        disallowed_clients: accessList.disallowed_clients ?? [],
    };
    const isAllowlistMode = values.allowed_clients.length > 0;

    if (disallowed && isAllowlistMode) {
        return {
            ...values,
            allowed_clients: addUnique(values.allowed_clients, ip),
        };
    }

    if (disallowed) {
        return {
            ...values,
            disallowed_clients: removeValue(values.disallowed_clients, disallowedRule || ip),
        };
    }

    if (isAllowlistMode) {
        return {
            ...values,
            allowed_clients: removeValue(values.allowed_clients, ip),
        };
    }

    return {
        ...values,
        disallowed_clients: addUnique(values.disallowed_clients, ip),
    };
};

export const toggleClientBlock =
    (ip: string, disallowed: boolean, disallowed_rule: string) => async (dispatch: any) => {
        dispatch(toggleClientBlockRequest());
        try {
            const accessList: AccessList = await apiClient.getAccessList();
            const values = getNextClientAccessList({
                accessList,
                ip,
                disallowed,
                disallowedRule: disallowed_rule,
            });

            await apiClient.setAccessList(values);
            dispatch(toggleClientBlockSuccess(values));

            if (disallowed) {
                dispatch(addSuccessToast(i18next.t('client_unblocked', { ip: disallowed_rule || ip })));
            } else {
                dispatch(addSuccessToast(i18next.t('client_blocked', { ip })));
            }
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(toggleClientBlockFailure());
        }
    };
