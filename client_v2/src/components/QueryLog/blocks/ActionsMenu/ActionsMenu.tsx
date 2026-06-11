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
    clientId?: string;
    onBlock: (domain: string) => void;
    onUnblock: (domain: string) => void;
    onBlockClient: (domain: string, client: string) => void;
    onDisallowClient: () => void;
    onAddPersistentClient?: (clientId: string) => void;
    isBlocked: boolean;
    showAddPersistentClient?: boolean;
    testIdPrefix?: string;
};

export const ActionsMenu = ({
    domain,
    client,
    clientId,
    onBlock,
    onUnblock,
    onBlockClient,
    onDisallowClient,
    onAddPersistentClient,
    isBlocked,
    showAddPersistentClient = false,
    testIdPrefix = 'actions-menu',
}: Props) => {
    const [open, setOpen] = useState(false);

    const handleBlock = () => {
        if (isBlocked) {
            onUnblock(domain);
        } else {
            onBlock(domain);
        }

        setOpen(false);
    };

    const handleBlockClient = () => {
        onBlockClient(domain, client);

        setOpen(false);
    };

    const handleDisallowClient = () => {
        onDisallowClient();
        setOpen(false);
    };

    const handleAddPersistentClient = () => {
        const nextClientId = clientId || client;

        if (onAddPersistentClient && nextClientId) {
            onAddPersistentClient(nextClientId);
        }

        setOpen(false);
    };

    const menu = (
        <ul
            className={s.menu}
            role="menu"
            data-testid={`${testIdPrefix}-actions-menu`}
            data-client={client}
        >
            <li role="none">
                <button
                    type="button"
                    data-testid={`${testIdPrefix}-action-toggle-block`}
                    className={cn(
                        s.menuItem,
                        s.menuButton,
                        theme.text.t3,
                        isBlocked ? s.statusGreen : s.statusRed,
                    )}
                    onClick={handleBlock}
                >
                    {isBlocked ? intl.getMessage('unblock') : intl.getMessage('block')}
                </button>
            </li>
            {!isBlocked && (
                <li role="none">
                    <button
                        type="button"
                        data-testid={`${testIdPrefix}-action-block-client`}
                        className={cn(s.menuItem, s.menuButton, theme.text.t3)}
                        onClick={handleBlockClient}
                    >
                        {intl.getMessage('block_for_this_client_only')}
                    </button>
                </li>
            )}
            <li role="none">
                <button
                    type="button"
                    data-testid={`${testIdPrefix}-action-disallow-client`}
                    className={cn(s.menuItem, s.menuButton, theme.text.t3)}
                    onClick={handleDisallowClient}
                >
                    {intl.getMessage('disallow_this_client')}
                </button>
            </li>
            {showAddPersistentClient && onAddPersistentClient && (
                <li role="none">
                    <button
                        type="button"
                        data-testid={`${testIdPrefix}-action-add-persistent-client`}
                        className={cn(s.menuItem, s.menuButton, theme.text.t3)}
                        onClick={handleAddPersistentClient}
                    >
                        {intl.getMessage('add_persistent_client')}
                    </button>
                </li>
            )}
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
            <button
                type="button"
                className={s.trigger}
                data-testid={`${testIdPrefix}-actions-trigger`}
            >
                <Icon icon="bullets" />
            </button>
        </Dropdown>
    );
};
