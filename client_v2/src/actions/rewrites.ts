import { createAction } from 'redux-actions';
import type { AppDispatch } from 'panel/store/types';
import intl from 'panel/common/intl';
import { apiClient } from '../api/Api';
import { addErrorToast, addSuccessToast } from './toasts';

type RewriteConfig = {
    answer: string;
    domain: string;
    enabled: boolean;
};

type RewriteUpdateConfig = {
    target: RewriteConfig;
    update: RewriteConfig;
};

type RewriteMutationOptions = {
    showToast?: boolean;
    closeModal?: boolean;
};

export const toggleRewritesModal = createAction('TOGGLE_REWRITES_MODAL');

export const getRewritesListRequest = createAction('GET_REWRITES_LIST_REQUEST');
export const getRewritesListFailure = createAction('GET_REWRITES_LIST_FAILURE');
export const getRewritesListSuccess = createAction('GET_REWRITES_LIST_SUCCESS');

export const getRewritesList = () => async (dispatch: AppDispatch) => {
    dispatch(getRewritesListRequest());
    try {
        const data = await apiClient.getRewritesList();
        dispatch(getRewritesListSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getRewritesListFailure());
    }
};

export const addRewriteRequest = createAction('ADD_REWRITE_REQUEST');
export const addRewriteFailure = createAction('ADD_REWRITE_FAILURE');
export const addRewriteSuccess = createAction('ADD_REWRITE_SUCCESS');

export const addRewrite = (config: RewriteConfig) => async (dispatch: AppDispatch) => {
    dispatch(addRewriteRequest());
    try {
        await apiClient.addRewrite(config);
        dispatch(addRewriteSuccess(config));
        dispatch(toggleRewritesModal());
        dispatch(getRewritesList());
        dispatch(addSuccessToast(intl.getMessage('rewrite_added', { key: config.domain })));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(addRewriteFailure());
    }
};

export const updateRewriteRequest = createAction('UPDATE_REWRITE_REQUEST');
export const updateRewriteFailure = createAction('UPDATE_REWRITE_FAILURE');
export const updateRewriteSuccess = createAction('UPDATE_REWRITE_SUCCESS');

/**
 * @param {Object} config
 * @param {string} config.target - current DNS rewrite value
 * @param {string} config.update - updated DNS rewrite value
 */
export const updateRewrite =
    (config: RewriteUpdateConfig, options: RewriteMutationOptions = {}) =>
    async (dispatch: AppDispatch): Promise<boolean> => {
        dispatch(updateRewriteRequest());
        try {
            await apiClient.updateRewrite(config);
            dispatch(updateRewriteSuccess());

            if (options.closeModal !== false) {
                dispatch(toggleRewritesModal());
            }

            dispatch(getRewritesList());

            if (options.showToast !== false) {
                dispatch(
                    addSuccessToast(
                        intl.getMessage('rewrite_updated', { key: config.update.domain }),
                    ),
                );
            }

            return true;
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(updateRewriteFailure());

            return false;
        }
    };

export const deleteRewriteRequest = createAction('DELETE_REWRITE_REQUEST');
export const deleteRewriteFailure = createAction('DELETE_REWRITE_FAILURE');
export const deleteRewriteSuccess = createAction('DELETE_REWRITE_SUCCESS');

export const deleteRewrite =
    (config: RewriteConfig, options: RewriteMutationOptions = {}) =>
    async (dispatch: AppDispatch): Promise<boolean> => {
        dispatch(deleteRewriteRequest());
        try {
            await apiClient.deleteRewrite(config);
            dispatch(deleteRewriteSuccess());
            dispatch(getRewritesList());

            if (options.showToast !== false) {
                dispatch(
                    addSuccessToast(intl.getMessage('rewrite_deleted', { key: config.domain })),
                );
            }

            return true;
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(deleteRewriteFailure());

            return false;
        }
    };

export const getRewriteSettingsRequest = createAction('GET_REWRITE_SETTINGS_REQUEST');
export const getRewriteSettingsFailure = createAction('GET_REWRITE_SETTINGS_FAILURE');
export const getRewriteSettingsSuccess = createAction('GET_REWRITE_SETTINGS_SUCCESS');

export const getRewriteSettings = () => async (dispatch: AppDispatch) => {
    dispatch(getRewriteSettingsRequest());
    try {
        const data = await apiClient.getRewriteSettings();
        dispatch(getRewriteSettingsSuccess(data));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getRewriteSettingsFailure());
    }
};

export const updateRewriteSettingsRequest = createAction('UPDATE_REWRITE_SETTINGS_REQUEST');
export const updateRewriteSettingsFailure = createAction('UPDATE_REWRITE_SETTINGS_FAILURE');
export const updateRewriteSettingsSuccess = createAction('UPDATE_REWRITE_SETTINGS_SUCCESS');

export const updateRewriteSettings =
    (config: { enabled: boolean }) => async (dispatch: AppDispatch) => {
        dispatch(updateRewriteSettingsRequest());
        try {
            await apiClient.updateRewriteSettings(config);
            dispatch(updateRewriteSettingsSuccess(config));
            dispatch(addSuccessToast(intl.getMessage('rewrite_settings_updated')));
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(updateRewriteSettingsFailure());
        }
    };
