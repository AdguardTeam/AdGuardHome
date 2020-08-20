import { handleActions } from 'redux-actions';
import * as actions from '../actions';

const settings = handleActions(
    {
        [actions.initSettingsRequest]: (state) => ({
            ...state,
            processing: true,
        }),
        [actions.initSettingsFailure]: (state) => ({
            ...state,
            processing: false,
        }),
        [actions.initSettingsSuccess]: (state, { payload }) => {
            const { settingsList } = payload;
            const newState = {
                ...state,
                settingsList,
                processing: false,
            };
            return newState;
        },
        [actions.toggleSettingStatus]: (state, { payload }) => {
            const { settingsList } = state;
            const { settingKey } = payload;

            const setting = settingsList[settingKey];

            const newSetting = {
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
        [actions.testUpstreamRequest]: (state) => ({
            ...state,
            processingTestUpstream: true,
        }),
        [actions.testUpstreamFailure]: (state) => ({
            ...state,
            processingTestUpstream: false,
        }),
        [actions.testUpstreamSuccess]: (state) => ({
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
