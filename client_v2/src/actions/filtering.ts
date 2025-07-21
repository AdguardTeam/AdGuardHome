import { createAction } from 'redux-actions';
import { showLoading, hideLoading } from 'react-redux-loading-bar';
import i18next from 'i18next';

import { normalizeFilteringStatus, normalizeRulesTextarea } from '../helpers/helpers';
import apiClient from '../api/Api';
import { addErrorToast, addSuccessToast } from './toasts';

export const toggleFilteringModal = createAction('FILTERING_MODAL_TOGGLE');
export const handleRulesChange = createAction('HANDLE_RULES_CHANGE');

export const getFilteringStatusRequest = createAction('GET_FILTERING_STATUS_REQUEST');
export const getFilteringStatusFailure = createAction('GET_FILTERING_STATUS_FAILURE');
export const getFilteringStatusSuccess = createAction('GET_FILTERING_STATUS_SUCCESS');

export const getFilteringStatus = () => async (dispatch: any) => {
    dispatch(getFilteringStatusRequest());
    try {
        const status = await apiClient.getFilteringStatus();
        dispatch(getFilteringStatusSuccess({ ...normalizeFilteringStatus(status) }));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(getFilteringStatusFailure());
    }
};

export const setRulesRequest = createAction('SET_RULES_REQUEST');
export const setRulesFailure = createAction('SET_RULES_FAILURE');
export const setRulesSuccess = createAction('SET_RULES_SUCCESS');

export const setRules = (rules: any) => async (dispatch: any) => {
    dispatch(setRulesRequest());
    try {
        const normalizedRules = {
            rules: normalizeRulesTextarea(rules)?.split('\n'),
        };
        await apiClient.setRules(normalizedRules);
        dispatch(addSuccessToast('updated_custom_filtering_toast'));
        dispatch(setRulesSuccess());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setRulesFailure());
    }
};

export const addFilterRequest = createAction('ADD_FILTER_REQUEST');
export const addFilterFailure = createAction('ADD_FILTER_FAILURE');
export const addFilterSuccess = createAction('ADD_FILTER_SUCCESS');

export const addFilter =
    (url: any, name: any, whitelist = false) =>
    async (dispatch: any, getState: any) => {
        dispatch(addFilterRequest());
        try {
            await apiClient.addFilter({ url, name, whitelist });
            dispatch(addFilterSuccess(url));
            if (getState().filtering.isModalOpen) {
                dispatch(toggleFilteringModal());
            }
            dispatch(addSuccessToast('filter_added_successfully'));
            dispatch(getFilteringStatus());
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(addFilterFailure());
        }
    };

export const removeFilterRequest = createAction('REMOVE_FILTER_REQUEST');
export const removeFilterFailure = createAction('REMOVE_FILTER_FAILURE');
export const removeFilterSuccess = createAction('REMOVE_FILTER_SUCCESS');

export const removeFilter =
    (url: any, whitelist = false) =>
    async (dispatch: any, getState: any) => {
        dispatch(removeFilterRequest());
        try {
            await apiClient.removeFilter({ url, whitelist });
            dispatch(removeFilterSuccess(url));
            if (getState().filtering.isModalOpen) {
                dispatch(toggleFilteringModal());
            }
            dispatch(addSuccessToast('filter_removed_successfully'));
            dispatch(getFilteringStatus());
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(removeFilterFailure());
        }
    };

export const toggleFilterRequest = createAction('FILTER_TOGGLE_REQUEST');
export const toggleFilterFailure = createAction('FILTER_TOGGLE_FAILURE');
export const toggleFilterSuccess = createAction('FILTER_TOGGLE_SUCCESS');

export const toggleFilterStatus =
    (url: any, data: any, whitelist = false) =>
    async (dispatch: any) => {
        dispatch(toggleFilterRequest());
        try {
            await apiClient.setFilterUrl({ url, data, whitelist });
            dispatch(toggleFilterSuccess(url));
            dispatch(getFilteringStatus());
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(toggleFilterFailure());
        }
    };

export const editFilterRequest = createAction('EDIT_FILTER_REQUEST');
export const editFilterFailure = createAction('EDIT_FILTER_FAILURE');
export const editFilterSuccess = createAction('EDIT_FILTER_SUCCESS');

export const editFilter =
    (url: any, data: any, whitelist = false) =>
    async (dispatch: any, getState: any) => {
        dispatch(editFilterRequest());
        try {
            await apiClient.setFilterUrl({ url, data, whitelist });
            dispatch(editFilterSuccess(url));
            if (getState().filtering.isModalOpen) {
                dispatch(toggleFilteringModal());
            }
            dispatch(addSuccessToast('filter_updated'));
            dispatch(getFilteringStatus());
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(editFilterFailure());
        }
    };

export const refreshFiltersRequest = createAction('FILTERING_REFRESH_REQUEST');
export const refreshFiltersFailure = createAction('FILTERING_REFRESH_FAILURE');
export const refreshFiltersSuccess = createAction('FILTERING_REFRESH_SUCCESS');

export const refreshFilters = (config: any) => async (dispatch: any) => {
    dispatch(refreshFiltersRequest());
    dispatch(showLoading());
    try {
        const data = await apiClient.refreshFilters(config);
        const { updated } = data;
        dispatch(refreshFiltersSuccess());

        if (updated > 0) {
            dispatch(addSuccessToast(i18next.t('list_updated', { count: updated })));
        } else {
            dispatch(addSuccessToast('all_lists_up_to_date_toast'));
        }

        dispatch(getFilteringStatus());
        dispatch(hideLoading());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(refreshFiltersFailure());
        dispatch(hideLoading());
    }
};

export const setFiltersConfigRequest = createAction('SET_FILTERS_CONFIG_REQUEST');
export const setFiltersConfigFailure = createAction('SET_FILTERS_CONFIG_FAILURE');
export const setFiltersConfigSuccess = createAction('SET_FILTERS_CONFIG_SUCCESS');

export const setFiltersConfig = (config: any) => async (dispatch: any, getState: any) => {
    dispatch(setFiltersConfigRequest());
    try {
        const { enabled } = config;
        const prevEnabled = getState().filtering.enabled;
        let successToastMessage = 'config_successfully_saved';

        if (prevEnabled !== enabled) {
            successToastMessage = enabled ? 'enabled_filtering_toast' : 'disabled_filtering_toast';
        }

        await apiClient.setFiltersConfig(config);
        dispatch(addSuccessToast(successToastMessage));
        dispatch(setFiltersConfigSuccess(config));
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(setFiltersConfigFailure());
    }
};

export const checkHostRequest = createAction('CHECK_HOST_REQUEST');
export const checkHostFailure = createAction('CHECK_HOST_FAILURE');
export const checkHostSuccess = createAction('CHECK_HOST_SUCCESS');

/**
 *
 * @param {object} host
 * @param {string} host.name
 * @returns {undefined}
 */
export const checkHost = (host: any) => async (dispatch: any) => {
    dispatch(checkHostRequest());
    try {
        const data = await apiClient.checkHost(host);
        const { name: hostname } = host;

        dispatch(
            checkHostSuccess({
                hostname,
                ...data,
            }),
        );
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(checkHostFailure());
    }
};
