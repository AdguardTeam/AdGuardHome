import { MANUAL_UPDATE_LINK } from 'panel/helpers/constants';

/**
 * Options for the update_failed notice toast, providing a clickable
 * hyperlink to the manual update instructions.
 */
export const updateFailedNoticeOptions = {
    components: {
        a: (children: string) => (
            <a href={MANUAL_UPDATE_LINK} target="_blank" rel="noopener noreferrer">
                {children}
            </a>
        ),
    },
};
