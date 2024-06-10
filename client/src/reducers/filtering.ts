import { handleActions } from 'redux-actions';

import * as actions from '../actions/filtering';

const filtering = handleActions(
    {
        [actions.setRulesRequest.toString()]: (state: any) => ({
            ...state,
            processingRules: true,
        }),
        [actions.setRulesFailure.toString()]: (state: any) => ({
            ...state,
            processingRules: false,
        }),
        [actions.setRulesSuccess.toString()]: (state: any) => ({
            ...state,
            processingRules: false,
        }),

        [actions.handleRulesChange.toString()]: (state: any, { payload }: any) => {
            const { userRules } = payload;
            return { ...state, userRules };
        },

        [actions.getFilteringStatusRequest.toString()]: (state: any) => ({
            ...state,
            processingFilters: true,
            check: {},
        }),
        [actions.getFilteringStatusFailure.toString()]: (state: any) => ({
            ...state,
            processingFilters: false,
        }),
        [actions.getFilteringStatusSuccess.toString()]: (state, { payload }: any) => ({
            ...state,
            ...payload,
            processingFilters: false,
        }),

        [actions.addFilterRequest.toString()]: (state: any) => ({
            ...state,
            processingAddFilter: true,
            isFilterAdded: false,
        }),
        [actions.addFilterFailure.toString()]: (state: any) => ({
            ...state,
            processingAddFilter: false,
            isFilterAdded: false,
        }),
        [actions.addFilterSuccess.toString()]: (state: any) => ({
            ...state,
            processingAddFilter: false,
            isFilterAdded: true,
        }),

        [actions.toggleFilteringModal.toString()]: (state: any, { payload }: any) => {
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
                modalType: '',
            };
            return newState;
        },

        [actions.toggleFilterRequest.toString()]: (state: any) => ({
            ...state,
            processingConfigFilter: true,
        }),
        [actions.toggleFilterFailure.toString()]: (state: any) => ({
            ...state,
            processingConfigFilter: false,
        }),
        [actions.toggleFilterSuccess.toString()]: (state: any) => ({
            ...state,
            processingConfigFilter: false,
        }),

        [actions.editFilterRequest.toString()]: (state: any) => ({
            ...state,
            processingConfigFilter: true,
        }),
        [actions.editFilterFailure.toString()]: (state: any) => ({
            ...state,
            processingConfigFilter: false,
        }),
        [actions.editFilterSuccess.toString()]: (state: any) => ({
            ...state,
            processingConfigFilter: false,
        }),

        [actions.refreshFiltersRequest.toString()]: (state: any) => ({
            ...state,
            processingRefreshFilters: true,
        }),
        [actions.refreshFiltersFailure.toString()]: (state: any) => ({
            ...state,
            processingRefreshFilters: false,
        }),
        [actions.refreshFiltersSuccess.toString()]: (state: any) => ({
            ...state,
            processingRefreshFilters: false,
        }),

        [actions.removeFilterRequest.toString()]: (state: any) => ({
            ...state,
            processingRemoveFilter: true,
        }),
        [actions.removeFilterFailure.toString()]: (state: any) => ({
            ...state,
            processingRemoveFilter: false,
        }),
        [actions.removeFilterSuccess.toString()]: (state: any) => ({
            ...state,
            processingRemoveFilter: false,
        }),

        [actions.setFiltersConfigRequest.toString()]: (state: any) => ({
            ...state,
            processingSetConfig: true,
        }),
        [actions.setFiltersConfigFailure.toString()]: (state: any) => ({
            ...state,
            processingSetConfig: false,
        }),
        [actions.setFiltersConfigSuccess.toString()]: (state, { payload }: any) => ({
            ...state,
            ...payload,
            processingSetConfig: false,
        }),

        [actions.checkHostRequest.toString()]: (state: any) => ({
            ...state,
            processingCheck: true,
        }),
        [actions.checkHostFailure.toString()]: (state: any) => ({
            ...state,
            processingCheck: false,
        }),
        [actions.checkHostSuccess.toString()]: (state, { payload }: any) => ({
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
