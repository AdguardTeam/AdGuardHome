import { combineReducers } from 'redux';
import { handleActions } from 'redux-actions';
import { reducer as formReducer } from 'redux-form';

import * as actions from '../actions/install';

const install = handleActions({
    [actions.getDefaultAddressesRequest]: state => ({ ...state, processingDefault: true }),
    [actions.getDefaultAddressesFailure]: state => ({ ...state, processingDefault: false }),
    [actions.getDefaultAddressesSuccess]: (state, { payload }) => {
        const newState = { ...state, ...payload, processingDefault: false };
        return newState;
    },

    [actions.nextStep]: state => ({ ...state, step: state.step + 1 }),
    [actions.prevStep]: state => ({ ...state, step: state.step - 1 }),

    [actions.setAllSettingsRequest]: state => ({ ...state, processingSubmit: true }),
    [actions.setAllSettingsFailure]: state => ({ ...state, processingSubmit: false }),
    [actions.setAllSettingsSuccess]: state => ({ ...state, processingSubmit: false }),
}, {
    step: 1,
    processingDefault: true,
});

export default combineReducers({
    install,
    form: formReducer,
});
