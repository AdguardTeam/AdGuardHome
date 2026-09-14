import { Show, createMemo, createSignal } from 'solid-js';

import intl from 'panel/common/intl';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { accessState, toggleClientBlock } from 'panel/stores/access';
import { addErrorToast } from 'panel/stores/toasts';

export type ClientBlockAction = 'block' | 'unblock';

export type ClientBlockConfirmState = {
    open: boolean;
    client: string;
    action: ClientBlockAction;
};

export const initialClientBlockConfirmState: ClientBlockConfirmState = {
    open: false,
    client: '',
    action: 'block',
};

export const CLIENT_BLOCK_CONFIRM_SUBMIT_TEST_ID = 'confirm-dialog-submit';

type ClientBlockConfirmDialogProps = {
    state: ClientBlockConfirmState;
    onClose: () => void;
    onConfirm: () => void;
};

export const ClientBlockConfirmDialog = (props: ClientBlockConfirmDialogProps) => {
    const isBlock = () => props.state.action === 'block';

    const title = createMemo(() =>
        isBlock()
            ? intl.getMessage('confirm_client_block_title', { ip: props.state.client })
            : intl.getMessage('confirm_client_unblock_title', { ip: props.state.client }),
    );

    const text = createMemo(() =>
        isBlock()
            ? intl.getMessage('confirm_client_block_desc', { ip: props.state.client })
            : intl.getMessage('confirm_client_unblock_desc', { ip: props.state.client }),
    );

    return (
        <Show when={props.state.open}>
            <ConfirmDialog
                onClose={props.onClose}
                onConfirm={props.onConfirm}
                title={title()}
                text={text()}
                buttonText={isBlock() ? intl.getMessage('block') : intl.getMessage('unblock')}
                cancelText={intl.getMessage('cancel')}
                buttonVariant={isBlock() ? 'danger' : 'primary'}
                submitTestId={CLIENT_BLOCK_CONFIRM_SUBMIT_TEST_ID}
            />
        </Show>
    );
};

/**
 * State and actions for the client block confirm dialog: blocked status
 * checks, opening/closing the dialog and applying the block/unblock action
 * to the access list.
 */
export const useClientBlockConfirm = () => {
    const [confirmState, setConfirmState] = createSignal<ClientBlockConfirmState>(
        initialClientBlockConfirmState,
    );

    const isClientBlocked = (clientName: string) => {
        const str = accessState.disallowed_clients || '';
        return str ? str.split('\n').filter(Boolean).includes(clientName) : false;
    };

    const openConfirmDialog = (client: string, action: ClientBlockAction) =>
        setConfirmState({ open: true, client, action });

    const closeConfirmDialog = () => setConfirmState(initialClientBlockConfirmState);

    const handleConfirm = async () => {
        const { client, action } = confirmState();
        if (action === 'block') {
            if (isClientBlocked(client)) {
                addErrorToast({
                    error: intl.getMessage('client_already_blocked', { ip: client }),
                });
                closeConfirmDialog();
                return;
            }
            await toggleClientBlock(client, false, '');
        } else {
            await toggleClientBlock(client, true, client);
        }
        closeConfirmDialog();
    };

    return {
        confirmState,
        isClientBlocked,
        openConfirmDialog,
        closeConfirmDialog,
        handleConfirm,
    };
};
