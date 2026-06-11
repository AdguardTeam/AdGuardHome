import { createAction } from 'redux-actions';

import intl from 'panel/common/intl';
import type { AppDispatch, AppGetState } from 'panel/store/types';
import type { Filter } from 'panel/helpers/helpers';
import { normalizeFilteringStatus, normalizeRulesTextarea } from '../helpers/helpers';
import { apiClient } from '../api/Api';
import { addErrorToast, addSuccessToast } from './toasts';
import { closeModal } from '../reducers/modals';

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

type FilterRemovalConfig = Pick<Filter, 'url' | 'name'>;

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
    (rules: string) =>
    async (dispatch: AppDispatch): Promise<boolean> => {
        dispatch(setRulesRequest());
        try {
            const normalizedUserRules = normalizeRulesTextarea(rules) || '';
            const normalizedRules = {
                rules: normalizedUserRules ? normalizedUserRules.split('\n') : [],
            };
            await apiClient.setRules(normalizedRules);

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
    async (dispatch: AppDispatch) => {
        dispatch(addFilterRequest());
        try {
            await apiClient.addFilter({ url, name, whitelist });
            dispatch(addFilterSuccess(url));
            dispatch(
                addSuccessToast({
                    message: whitelist
                        ? intl.getMessage('filter_added_successfully_allowlist', {
                              value: name || url,
                          })
                        : intl.getMessage('filter_added_successfully', { value: name || url }),
                }),
            );
            dispatch(closeModal());
            dispatch(getFilteringStatus());
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(addFilterFailure());
        }
    };

export const addFiltersBatch =
    (filters: FilterIdentity[]) =>
    async (dispatch: AppDispatch) => {
        dispatch(addFilterRequest());
        try {
            const results = await Promise.allSettled(
                filters.map(({ url, name }) =>
                    apiClient.addFilter({ url, name, whitelist: false }),
                ),
            );

            const successes: FilterIdentity[] = [];
            const failures: Array<{ filter: FilterIdentity; error: unknown }> = [];

            results.forEach((result, index) => {
                if (result.status === 'fulfilled') {
                    successes.push(filters[index]);
                } else {
                    failures.push({
                        filter: filters[index],
                        error: result.reason,
                    });
                }
            });

            if (successes.length === 1) {
                dispatch(
                    addSuccessToast({
                        message: intl.getMessage('filter_added_successfully', {
                            value: successes[0].name || successes[0].url,
                        }),
                    }),
                );
            } else if (successes.length > 1) {
                dispatch(
                    addSuccessToast({
                        message: intl.getMessage('filter_added_successfully_more', {
                            value: successes[0].name || successes[0].url,
                            more: String(successes.length - 1),
                        }),
                    }),
                );
            }

            failures.forEach(({ error }) => {
                dispatch(addErrorToast({ error }));
            });

            if (successes.length > 0) {
                dispatch(addFilterSuccess(successes[0].url));
                dispatch(closeModal());
                dispatch(getFilteringStatus());
            } else {
                dispatch(addFilterFailure());
                // Modal stays open so user can retry
            }
        } catch (error) {
            dispatch(addErrorToast({ error }));
            dispatch(addFilterFailure());
            // Modal stays open so user can retry
        }
    };

export const removeFilterRequest = createAction('REMOVE_FILTER_REQUEST');
export const removeFilterFailure = createAction('REMOVE_FILTER_FAILURE');
export const removeFilterSuccess = createAction('REMOVE_FILTER_SUCCESS');

export const removeFilter =
    (filter: FilterRemovalConfig, whitelist = false) =>
    async (dispatch: AppDispatch) => {
        dispatch(removeFilterRequest());
        try {
            await apiClient.removeFilter({ url: filter.url, whitelist });
            dispatch(removeFilterSuccess(filter.url));
            dispatch(closeModal());
            dispatch(
                addSuccessToast({
                    message: whitelist
                        ? intl.getMessage('filter_removed_successfully_allowlist', {
                              value: filter.name || filter.url,
                          })
                        : intl.getMessage('filter_removed_successfully', {
                              value: filter.name || filter.url,
                          }),
                }),
            );
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
