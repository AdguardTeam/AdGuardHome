import i18next from 'i18next';
import { splitByNewLine } from '../../../../helpers/helpers';

export const BUTTON_PREFIX = 'btn_';

export const getBlockClientInfo = (ip: any, disallowed: any, disallowed_rule: any, allowedClients: any) => {
    let confirmMessage;

    if (disallowed) {
        confirmMessage = i18next.t('client_confirm_unblock', { ip: disallowed_rule || ip });
    } else {
        confirmMessage = `${i18next.t('adg_will_drop_dns_queries')} ${i18next.t('client_confirm_block', { ip })}`;
        if (allowedClients.length > 0) {
            confirmMessage = confirmMessage.concat(`\n\n${i18next.t('filter_allowlist', { disallowed_rule: ip })}`);
        }
    }

    const buttonKey = i18next.t(disallowed ? 'allow_this_client' : 'disallow_this_client');
    const allowedClientsList = splitByNewLine(allowedClients || '');
    const lastRuleInAllowlist = !disallowed && allowedClientsList.length === 1 && allowedClientsList[0] === ip;

    return {
        confirmMessage,
        buttonKey,
        lastRuleInAllowlist,
    };
};
