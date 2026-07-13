import { Show, createSignal } from 'solid-js';
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

export const ActionsMenu = (props: Props) => {
    const [open, setOpen] = createSignal(false);

    const handleBlock = () => {
        if (props.isBlocked) {
            props.onUnblock(props.domain);
        } else {
            props.onBlock(props.domain);
        }
        setOpen(false);
    };

    const handleBlockClient = () => {
        props.onBlockClient(props.domain, props.client);
        setOpen(false);
    };

    const handleDisallowClient = () => {
        props.onDisallowClient();
        setOpen(false);
    };

    const handleAddPersistentClient = () => {
        const nextClientId = props.clientId || props.client;
        if (props.onAddPersistentClient && nextClientId) {
            props.onAddPersistentClient(nextClientId);
        }
        setOpen(false);
    };

    const menu = (
        <ul
            class={s.menu}
            role="menu"
            data-testid={`${props.testIdPrefix}-actions-menu`}
            data-client={props.client}
        >
            <li role="none">
                <button
                    type="button"
                    data-testid={`${props.testIdPrefix}-action-toggle-block`}
                    class={cn(
                        s.menuItem,
                        s.menuButton,
                        theme.text.t3,
                        props.isBlocked ? s.statusGreen : s.statusRed,
                    )}
                    onClick={handleBlock}
                >
                    {props.isBlocked ? intl.getMessage('unblock') : intl.getMessage('block')}
                </button>
            </li>
            <Show when={!props.isBlocked}>
                <li role="none">
                    <button
                        type="button"
                        data-testid={`${props.testIdPrefix}-action-block-client`}
                        class={cn(s.menuItem, s.menuButton, theme.text.t3)}
                        onClick={handleBlockClient}
                    >
                        {intl.getMessage('block_for_this_client_only')}
                    </button>
                </li>
            </Show>
            <li role="none">
                <button
                    type="button"
                    data-testid={`${props.testIdPrefix}-action-disallow-client`}
                    class={cn(s.menuItem, s.menuButton, theme.text.t3)}
                    onClick={handleDisallowClient}
                >
                    {intl.getMessage('disallow_this_client')}
                </button>
            </li>
            <Show when={props.showAddPersistentClient && props.onAddPersistentClient}>
                <li role="none">
                    <button
                        type="button"
                        data-testid={`${props.testIdPrefix}-action-add-persistent-client`}
                        class={cn(s.menuItem, s.menuButton, theme.text.t3)}
                        onClick={handleAddPersistentClient}
                    >
                        {intl.getMessage('add_persistent_client')}
                    </button>
                </li>
            </Show>
        </ul>
    );

    return (
        <Dropdown
            trigger="click"
            menu={menu}
            open={open()}
            onOpenChange={setOpen}
            position="bottomRight"
            noIcon
            overlayClass={s.overlay}
        >
            <button
                type="button"
                class={s.trigger}
                data-testid={`${props.testIdPrefix}-actions-trigger`}
            >
                <Icon icon="bullets" />
            </button>
        </Dropdown>
    );
};
