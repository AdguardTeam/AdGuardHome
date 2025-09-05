import { combineReducers } from 'redux';

import toasts from './toasts';
import modals from './modals';
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
    modals,
});
