import { combineReducers } from 'redux';
import { loadingBarReducer } from 'react-redux-loading-bar';

import { reducer as formReducer } from 'redux-form';
import toasts from './toasts';
import encryption from './encryption';
import clients from './clients';
import access from './access';
import rewrites from './rewrites';
import services from './services';
import stats from './stats';
import queryLogs from './queryLogs';
import dnsConfig from './dnsConfig';
import filtering from './filtering';
import settings from './settings';
import dashboard from './dashboard';
import dhcp from './dhcp';

export default combineReducers({
    settings,
    dashboard,
    queryLogs,
    filtering,
    toasts,
    dhcp,
    encryption,
    clients,
    access,
    rewrites,
    services,
    stats,
    dnsConfig,
    loadingBar: loadingBarReducer,
    form: formReducer,
});
