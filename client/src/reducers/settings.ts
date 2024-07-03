import { handleActions } from 'redux-actions';

import * as actions from '../actions';

const settings = handleActions(
    {
        [actions.initSettingsRequest.toString()]: (state: any) => ({
            ...state,
            processing: true,
        }),
        [actions.initSettingsFailure.toString()]: (state: any) => ({
            ...state,
            processing: false,
        }),
        [actions.initSettingsSuccess.toString()]: (state: any, { payload }: any) => {
            const { settingsList } = payload;
            const newState = {
                ...state,
                settingsList,
                processing: false,
            };
            return newState;
        },
        [actions.toggleSettingStatus.toString()]: (state: any, { payload }: any) => {
            const { settingsList } = state;
            const { settingKey, value } = payload;

            const setting = settingsList[settingKey];

            const newSetting = value || {
                ...setting,
                enabled: !setting.enabled,
            };
            const newSettingsList = {
                ...settingsList,
                [settingKey]: newSetting,
            };
            return {
                ...state,
                settingsList: newSettingsList,
            };
        },
        [actions.testUpstreamRequest.toString()]: (state: any) => ({
            ...state,
            processingTestUpstream: true,
        }),
        [actions.testUpstreamFailure.toString()]: (state: any) => ({
            ...state,
            processingTestUpstream: false,
        }),
        [actions.testUpstreamSuccess.toString()]: (state: any) => ({
            ...state,
            processingTestUpstream: false,
        }),
    },
    {
        processing: true,
        processingTestUpstream: false,
        processingDhcpStatus: false,
        settingsList: {},
    },
);

export default settings;
