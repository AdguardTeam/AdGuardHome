import { createAction } from 'redux-actions';
import i18next from 'i18next';
import apiClient from '../api/Api';
import { addErrorToast, addSuccessToast } from './toasts';

export const toggleRewritesModal = createAction('TOGGLE_REWRITES_MODAL');

export const getRewritesListRequest = createAction('GET_REWRITES_LIST_REQUEST');
export const getRewritesListFailure = createAction('GET_REWRITES_LIST_FAILURE');
export const getRewritesListSuccess = createAction('GET_REWRITES_LIST_SUCCESS');

export const getRewritesList = () => async (dispatch) => {
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

export const addRewrite = (config) => async (dispatch) => {
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

export const deleteRewriteRequest = createAction('DELETE_REWRITE_REQUEST');
export const deleteRewriteFailure = createAction('DELETE_REWRITE_FAILURE');
export const deleteRewriteSuccess = createAction('DELETE_REWRITE_SUCCESS');

export const deleteRewrite = (config) => async (dispatch) => {
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
