import { createMemo } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

import s from './ClientBlockMenuItem.module.pcss';

import type { ClientBlockAction } from './ClientBlockConfirm';

type Props = {
    action: ClientBlockAction;
    onClick: (action: ClientBlockAction) => void;
    class?: string;
};

/**
 * The block/unblock action for one client, shared by the Dashboard card and
 * the Top clients page.  Rendered as a `<button>` so it is keyboard-operable
 * both inline on mobile and inside the desktop dropdown.
 */
export const ClientBlockMenuItem = (props: Props) => {
    const isBlock = createMemo(() => props.action === 'block');

    return (
        <button
            type="button"
            class={cn(theme.text.t2, theme.text.condenced, s.item, props.class, {
                [s.itemDanger]: isBlock(),
            })}
            data-testid={isBlock() ? 'client-block-menu-item' : 'client-unblock-menu-item'}
            onClick={() => props.onClick(props.action)}
        >
            {isBlock() ? intl.getMessage('block_client') : intl.getMessage('unblock_client')}
        </button>
    );
};
