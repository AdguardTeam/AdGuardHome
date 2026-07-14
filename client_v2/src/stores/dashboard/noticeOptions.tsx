import intl from 'panel/common/intl';
import { MANUAL_UPDATE_LINK } from 'panel/helpers/constants';
import theme from 'panel/lib/theme';

export const getUpdateFailedMessage = () =>
    intl.getMessage('update_failed', {
        a: (text: string) => (
            <a
                href={MANUAL_UPDATE_LINK}
                target="_blank"
                rel="noopener noreferrer"
                class={theme.link.link}
            >
                {text}
            </a>
        ),
    });
