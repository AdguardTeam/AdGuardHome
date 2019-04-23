import { combineReducers } from 'redux';
import { handleActions } from 'redux-actions';
import { reducer as formReducer } from 'redux-form';

import * as actions from '../actions/install';
import toasts from './toasts';
import { INSTALL_FIRST_STEP } from '../helpers/constants';

const install = handleActions({
    [actions.getDefaultAddressesRequest]: state => ({ ...state, processingDefault: true }),
    [actions.getDefaultAddressesFailure]: state => ({ ...state, processingDefault: false }),
    [actions.getDefaultAddressesSuccess]: (state, { payload }) => {
        const { interfaces } = payload;
        const web = { ...state.web, port: payload.web_port };
        const dns = { ...state.dns, port: payload.dns_port };

        const newState = {
            ...state, web, dns, interfaces, processingDefault: false,
        };
        return newState;
    },

    [actions.nextStep]: state => ({ ...state, step: state.step + 1 }),
    [actions.prevStep]: state => ({ ...state, step: state.step - 1 }),

    [actions.setAllSettingsRequest]: state => ({ ...state, processingSubmit: true }),
    [actions.setAllSettingsFailure]: state => ({ ...state, processingSubmit: false }),
    [actions.setAllSettingsSuccess]: state => ({ ...state, processingSubmit: false }),

    [actions.checkConfigRequest]: state => ({ ...state, processingCheck: true }),
    [actions.checkConfigFailure]: state => ({ ...state, processingCheck: false }),
    [actions.checkConfigSuccess]: (state, { payload }) => {
        const web = { ...state.web, ...payload.web };
        const dns = { ...state.dns, ...payload.dns };

        const newState = {
            ...state, web, dns, processingCheck: false,
        };
        return newState;
    },
}, {
    step: INSTALL_FIRST_STEP,
    processingDefault: true,
    processingSubmit: false,
    processingCheck: false,
    web: {
        ip: '0.0.0.0',
        port: 80,
        status: '',
        can_autofix: false,
    },
    dns: {
        ip: '0.0.0.0',
        port: 53,
        status: '',
        can_autofix: false,
    },
    interfaces: {},
});

export default combineReducers({
    install,
    toasts,
    form: formReducer,
});
