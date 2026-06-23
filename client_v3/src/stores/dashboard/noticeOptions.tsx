import intl from 'panel/common/intl';
import { MANUAL_UPDATE_LINK } from 'panel/helpers/constants';

/**
 * Resolves the "update_failed" notice message with its clickable
 * hyperlink.  Callers pass the result directly as the toast message
 * so the Toast component never needs to do i18n resolution.
 */
export const getUpdateFailedMessage = () => {
    const components = {
        a: (children: string) => (
            <a
                href={MANUAL_UPDATE_LINK}
                target="_blank"
                rel="noopener noreferrer"
            >
                {children}
            </a>
        ),
    };
    return intl.getMessage('update_failed', components);
};
