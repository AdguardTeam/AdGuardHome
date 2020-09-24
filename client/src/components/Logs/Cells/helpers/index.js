import i18next from 'i18next';

export const BUTTON_PREFIX = 'btn_';

export const getBlockClientInfo = (ip, disallowed, disallowed_rule) => {
    const confirmMessage = disallowed
        ? i18next.t('client_confirm_unblock', { ip: disallowed_rule })
        : `${i18next.t('adg_will_drop_dns_queries')} ${i18next.t('client_confirm_block', { ip })}`;

    const buttonKey = i18next.t(disallowed ? 'allow_this_client' : 'disallow_this_client');
    const isNotInAllowedList = disallowed && disallowed_rule === '';

    return {
        confirmMessage,
        buttonKey,
        isNotInAllowedList,
    };
};
