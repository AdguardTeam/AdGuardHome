import { createAction } from 'redux-actions';

import intl from 'panel/common/intl';
import type { AppDispatch, AppGetState } from 'panel/store/types';
import type { Filter } from 'panel/helpers/helpers';
import { normalizeFilteringStatus, normalizeRulesTextarea } from '../helpers/helpers';
import { apiClient } from '../api/Api';
import { addErrorToast, addSuccessToast } from './toasts';

type RulesMutationOptions = {
    showToast?: boolean;
};

type CheckHostRequest = {
    name: string;
    client?: string;
    qtype?: string;
};

type FilterUrlConfig = {
    name: string;
    url: string;
    enabled: boolean;
};

type FilterIdentity = Pick<Filter, 'name' | 'url'>;

type FilterEditConfig = FilterIdentity & {
    enabled?: boolean;
};

type FilterRemovalConfig = Pick<Filter, 'url'>;

type RefreshFiltersConfig = {
    whitelist: boolean;
};

export const toggleFilteringModal = createAction('FILTERING_MODAL_TOGGLE');
export const setFilterModalUrl = createAction('SET_FILTER_MODAL_URL');
export const handleRulesChange = createAction('HANDLE_RULES_CHANGE');

export const getFilteringStatusRequest = createAction('GET_FILTERING_STATUS_REQUEST');
export const getFilteringStatusFailure = createAction('GET_FILTERING_STATUS_FAILURE');
export const getFilteringStatusSuccess = createAction('GET_FILTERING_STATUS_SUCCESS');

export const getFilteringStatus = () => async (dispatch: AppDispatch) => {
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

export const setRules =
    (rules: string, options: RulesMutationOptions = {}) =>
    async (dispatch: AppDispatch): Promise<boolean> => {
        dispatch(setRulesRequest());
        try {
            const normalizedUserRules = normalizeRulesTextarea(rules) || '';
            const normalizedRules = {
                rules: normalizedUserRules ? normalizedUserRules.split('\n') : [],
            };
            await apiClient.setRules(normalizedRules);

            if (options.showToast !== false) {
                dispatch(addSuccessToast(intl.getMessage('updated_custom_filtering_toast')));
            }

            dispatch(
                setRulesSuccess({
                    userRules: normalizedUserRules,
                }),
            );

            return true;
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(setRulesFailure());

            return false;
        }
    };

export const addFilterRequest = createAction('ADD_FILTER_REQUEST');
export const addFilterFailure = createAction('ADD_FILTER_FAILURE');
export const addFilterSuccess = createAction('ADD_FILTER_SUCCESS');

export const addFilter =
    (url: FilterIdentity['url'], name: FilterIdentity['name'], whitelist = false) =>
    async (dispatch: AppDispatch, getState: AppGetState) => {
        dispatch(addFilterRequest());
        try {
            await apiClient.addFilter({ url, name, whitelist });
            dispatch(addFilterSuccess(url));
            if (getState().filtering.isModalOpen) {
                dispatch(toggleFilteringModal());
            }
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
    (url: FilterRemovalConfig['url'], whitelist = false) =>
    async (dispatch: AppDispatch, getState: AppGetState) => {
        dispatch(removeFilterRequest());
        try {
            await apiClient.removeFilter({ url, whitelist });
            dispatch(removeFilterSuccess(url));
            if (getState().filtering.isModalOpen) {
                dispatch(toggleFilteringModal());
            }
            dispatch(addSuccessToast(intl.getMessage('filter_removed_successfully')));
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
    (url: string, data: FilterUrlConfig, whitelist = false) =>
    async (dispatch: AppDispatch): Promise<boolean> => {
        dispatch(toggleFilterRequest());
        try {
            await apiClient.setFilterUrl({ url, data, whitelist });
            dispatch(toggleFilterSuccess(url));
            dispatch(getFilteringStatus());

            return true;
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(toggleFilterFailure());

            return false;
        }
    };

export const editFilterRequest = createAction('EDIT_FILTER_REQUEST');
export const editFilterFailure = createAction('EDIT_FILTER_FAILURE');
export const editFilterSuccess = createAction('EDIT_FILTER_SUCCESS');

export const editFilter =
    (url: FilterIdentity['url'], data: FilterEditConfig, whitelist = false) =>
    async (dispatch: AppDispatch, getState: AppGetState) => {
        dispatch(editFilterRequest());
        try {
            await apiClient.setFilterUrl({ url, data, whitelist });
            dispatch(editFilterSuccess(url));
            if (getState().filtering.isModalOpen) {
                dispatch(toggleFilteringModal());
            }
            dispatch(addSuccessToast(intl.getMessage('changes_saved_success')));
            dispatch(getFilteringStatus());
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(editFilterFailure());
        }
    };

export const refreshFiltersRequest = createAction('FILTERING_REFRESH_REQUEST');
export const refreshFiltersFailure = createAction('FILTERING_REFRESH_FAILURE');
export const refreshFiltersSuccess = createAction('FILTERING_REFRESH_SUCCESS');

export const refreshFilters = (config: RefreshFiltersConfig) => async (dispatch: AppDispatch) => {
    dispatch(refreshFiltersRequest());
    try {
        const data = await apiClient.refreshFilters(config);
        const { updated } = data;
        dispatch(refreshFiltersSuccess());

        if (updated > 0) {
            dispatch(addSuccessToast(intl.getPlural('list_updated', updated)));
        } else {
            dispatch(addSuccessToast(intl.getMessage('all_lists_up_to_date_toast')));
        }

        dispatch(getFilteringStatus());
    } catch (error) {
        dispatch(addErrorToast({ error }));
        dispatch(refreshFiltersFailure());
    }
};

export const setFiltersConfigRequest = createAction('SET_FILTERS_CONFIG_REQUEST');
export const setFiltersConfigFailure = createAction('SET_FILTERS_CONFIG_FAILURE');
export const setFiltersConfigSuccess = createAction('SET_FILTERS_CONFIG_SUCCESS');

export const setFiltersConfig = (config: any) => async (dispatch: AppDispatch) => {
    dispatch(setFiltersConfigRequest());
    try {
        await apiClient.setFiltersConfig(config);
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
export const checkHost =
    (host: CheckHostRequest) =>
    async (dispatch: AppDispatch): Promise<boolean> => {
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

            return true;
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(checkHostFailure());

            return false;
        }
    };
