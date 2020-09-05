import { getIpMatchListStatus } from '../../../../helpers/helpers';
import { BLOCK_ACTIONS, IP_MATCH_LIST_STATUS } from '../../../../helpers/constants';

export const BUTTON_PREFIX = 'btn_';

export const getBlockClientInfo = (client, disallowed_clients) => {
    const ipMatchListStatus = getIpMatchListStatus(client, disallowed_clients);

    const isNotFound = ipMatchListStatus === IP_MATCH_LIST_STATUS.NOT_FOUND;
    const type = isNotFound ? BLOCK_ACTIONS.BLOCK : BLOCK_ACTIONS.UNBLOCK;

    const confirmMessage = isNotFound ? 'client_confirm_block' : 'client_confirm_unblock';
    const buttonKey = isNotFound ? 'disallow_this_client' : 'allow_this_client';
    return {
        confirmMessage,
        buttonKey,
        type,
    };
};
