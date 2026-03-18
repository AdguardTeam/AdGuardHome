import React, { useState } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Icon } from 'panel/common/ui/Icon';

import theme from 'panel/lib/theme';
import s from './ActionsMenu.module.pcss';

type Props = {
    domain: string;
    client: string;
    onBlock: (type: string, domain: string) => void;
    onUnblock: (type: string, domain: string) => void;
    onBlockClient: (type: string, domain: string, client: string) => void;
    onDisallowClient: () => void;
    isBlocked: boolean;
};

export const ActionsMenu = ({
    domain,
    client,
    onBlock,
    onUnblock,
    onBlockClient,
    onDisallowClient,
    isBlocked,
}: Props) => {
    const [open, setOpen] = useState(false);

    const handleBlock = () => {
        if (isBlocked) {
            onUnblock('unblock', domain);
        } else {
            onBlock('block', domain);
        }
        setOpen(false);
    };

    const handleBlockClient = () => {
        if (isBlocked) {
            onUnblock('unblock', domain);
        } else {
            onBlockClient('block', domain, client);
        }
        setOpen(false);
    };

    const handleDisallowClient = () => {
        onDisallowClient();
        setOpen(false);
    };

    const menu = (
        <ul className={s.menu}>
            <li className={cn(s.menuItem, theme.text.t3)} onClick={handleBlock}>
                {isBlocked ? intl.getMessage('unblock') : intl.getMessage('block')}
            </li>
            {!isBlocked && (
                <li className={cn(s.menuItem, theme.text.t3)} onClick={handleBlockClient}>
                    {intl.getMessage('block_for_this_client_only')}
                </li>
            )}
            <li className={cn(s.menuItem, theme.text.t3)} onClick={handleDisallowClient}>
                {intl.getMessage('disallow_this_client')}
            </li>
        </ul>
    );

    return (
        <Dropdown
            trigger="click"
            menu={menu}
            open={open}
            onOpenChange={setOpen}
            position="bottomRight"
            noIcon
            overlayClassName={s.overlay}
        >
            <button type="button" className={s.trigger}>
                <Icon icon="bullets" />
            </button>
        </Dropdown>
    );
};
