import { createAction } from 'redux-actions';
import i18next from 'i18next';
import apiClient from '../api/Api';
import { addErrorToast, addSuccessToast } from './toasts';

export const toggleRewritesModal = createAction('TOGGLE_REWRITES_MODAL');

export const getRewritesListRequest = createAction('GET_REWRITES_LIST_REQUEST');
export const getRewritesListFailure = createAction('GET_REWRITES_LIST_FAILURE');
export const getRewritesListSuccess = createAction('GET_REWRITES_LIST_SUCCESS');

export const getRewritesList = () => async (dispatch: any) => {
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

export const addRewrite = (config: any) => async (dispatch: any) => {
    dispatch(addRewriteRequest());
    try {
        await apiClient.addRewrite(config);
        dispatch(addRewriteSuccess(config));
        dispatch(toggleRewritesModal());
        dispatch(getRewritesList());
        dispatch(addSuccessToast(i18next.t('rewrite_added', { key: config.domain })));
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
export const updateRewrite = (config: any) => async (dispatch: any) => {
    dispatch(updateRewriteRequest());
    try {
        await apiClient.updateRewrite(config);
        dispatch(updateRewriteSuccess());
        dispatch(toggleRewritesModal());
        dispatch(getRewritesList());
        dispatch(addSuccessToast(i18next.t('rewrite_updated', { key: config.domain })));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(updateRewriteFailure());
    }
};

export const deleteRewriteRequest = createAction('DELETE_REWRITE_REQUEST');
export const deleteRewriteFailure = createAction('DELETE_REWRITE_FAILURE');
export const deleteRewriteSuccess = createAction('DELETE_REWRITE_SUCCESS');

export const deleteRewrite = (config: any) => async (dispatch: any) => {
    dispatch(deleteRewriteRequest());
    try {
        await apiClient.deleteRewrite(config);
        dispatch(deleteRewriteSuccess());
        dispatch(getRewritesList());
        dispatch(addSuccessToast(i18next.t('rewrite_deleted', { key: config.domain })));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(deleteRewriteFailure());
    }
};
