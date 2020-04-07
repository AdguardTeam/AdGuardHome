import { handleActions } from 'redux-actions';

import * as actions from '../actions/filtering';

const filtering = handleActions(
    {
        [actions.setRulesRequest]: state => ({ ...state, processingRules: true }),
        [actions.setRulesFailure]: state => ({ ...state, processingRules: false }),
        [actions.setRulesSuccess]: state => ({ ...state, processingRules: false }),

        [actions.handleRulesChange]: (state, { payload }) => {
            const { userRules } = payload;
            return { ...state, userRules };
        },

        [actions.getFilteringStatusRequest]: state => ({
            ...state,
            processingFilters: true,
            check: {},
        }),
        [actions.getFilteringStatusFailure]: state => ({ ...state, processingFilters: false }),
        [actions.getFilteringStatusSuccess]: (state, { payload }) => ({
            ...state,
            ...payload,
            processingFilters: false,
        }),

        [actions.addFilterRequest]: state => ({
            ...state,
            processingAddFilter: true,
            isFilterAdded: false,
        }),
        [actions.addFilterFailure]: state => ({
            ...state,
            processingAddFilter: false,
            isFilterAdded: false,
        }),
        [actions.addFilterSuccess]: state => ({
            ...state,
            processingAddFilter: false,
            isFilterAdded: true,
        }),

        [actions.toggleFilteringModal]: (state, { payload }) => {
            if (payload) {
                const newState = {
                    ...state,
                    isModalOpen: !state.isModalOpen,
                    isFilterAdded: false,
                    modalType: payload.type || '',
                    modalFilterUrl: payload.url || '',
                };
                return newState;
            }
            const newState = {
                ...state,
                isModalOpen: !state.isModalOpen,
                isFilterAdded: false,
            };
            return newState;
        },

        [actions.toggleFilterRequest]: state => ({ ...state, processingConfigFilter: true }),
        [actions.toggleFilterFailure]: state => ({ ...state, processingConfigFilter: false }),
        [actions.toggleFilterSuccess]: state => ({ ...state, processingConfigFilter: false }),

        [actions.editFilterRequest]: state => ({ ...state, processingConfigFilter: true }),
        [actions.editFilterFailure]: state => ({ ...state, processingConfigFilter: false }),
        [actions.editFilterSuccess]: state => ({ ...state, processingConfigFilter: false }),

        [actions.refreshFiltersRequest]: state => ({ ...state, processingRefreshFilters: true }),
        [actions.refreshFiltersFailure]: state => ({ ...state, processingRefreshFilters: false }),
        [actions.refreshFiltersSuccess]: state => ({ ...state, processingRefreshFilters: false }),

        [actions.removeFilterRequest]: state => ({ ...state, processingRemoveFilter: true }),
        [actions.removeFilterFailure]: state => ({ ...state, processingRemoveFilter: false }),
        [actions.removeFilterSuccess]: state => ({ ...state, processingRemoveFilter: false }),

        [actions.setFiltersConfigRequest]: state => ({ ...state, processingSetConfig: true }),
        [actions.setFiltersConfigFailure]: state => ({ ...state, processingSetConfig: false }),
        [actions.setFiltersConfigSuccess]: (state, { payload }) => ({
            ...state,
            ...payload,
            processingSetConfig: false,
        }),

        [actions.checkHostRequest]: state => ({ ...state, processingCheck: true }),
        [actions.checkHostFailure]: state => ({ ...state, processingCheck: false }),
        [actions.checkHostSuccess]: (state, { payload }) => ({
            ...state,
            check: payload,
            processingCheck: false,
        }),
    },
    {
        isModalOpen: false,
        processingFilters: false,
        processingRules: false,
        processingAddFilter: false,
        processingRefreshFilters: false,
        processingConfigFilter: false,
        processingRemoveFilter: false,
        processingSetConfig: false,
        processingCheck: false,
        isFilterAdded: false,
        filters: [],
        whitelistFilters: [],
        userRules: '',
        interval: 24,
        enabled: true,
        modalType: '',
        modalFilterUrl: '',
        check: {},
    },
);

export default filtering;
