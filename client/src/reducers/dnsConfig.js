import { handleActions } from 'redux-actions';

import * as actions from '../actions/dnsConfig';
import { BLOCKING_MODES } from '../helpers/constants';

const dnsConfig = handleActions(
    {
        [actions.getDnsConfigRequest]: state => ({ ...state, processingGetConfig: true }),
        [actions.getDnsConfigFailure]: state =>
            ({ ...state, processingGetConfig: false }),
        [actions.getDnsConfigSuccess]: (state, { payload }) => ({
            ...state,
            ...payload,
            processingGetConfig: false,
        }),

        [actions.setDnsConfigRequest]: state => ({ ...state, processingSetConfig: true }),
        [actions.setDnsConfigFailure]: state =>
            ({ ...state, processingSetConfig: false }),
        [actions.setDnsConfigSuccess]: (state, { payload }) => ({
            ...state,
            ...payload,
            processingSetConfig: false,
        }),
    },
    {
        processingGetConfig: false,
        processingSetConfig: false,
        blocking_mode: BLOCKING_MODES.nxdomain,
        ratelimit: 20,
        blocking_ipv4: '',
        blocking_ipv6: '',
    },
);

export default dnsConfig;
