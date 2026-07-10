import { Show } from 'solid-js';

import intl from 'panel/common/intl';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';

import s from './DisallowDialog.module.pcss';

type Props = {
    ip: string;
    isAllowlistMode: boolean;
    onConfirm: () => void;
    onClose: () => void;
};

export const DisallowDialog = (props: Props) => {
    return (
        <ConfirmDialog
            title={intl.getMessage('disallow_client_confirm_title')}
            text={
                <div>
                    <div>{intl.getMessage('disallow_client_confirm_text', { ip: props.ip })}</div>
                    <Show when={props.isAllowlistMode}>
                        <div class={s.note}>
                            {intl.getMessage('disallow_client_confirm_allowlist_note', {
                                ip: props.ip,
                            })}
                        </div>
                    </Show>
                </div>
            }
            buttonText={intl.getMessage('yes_disallow')}
            cancelText={intl.getMessage('cancel')}
            buttonVariant="danger"
            submitTestId="query-log-disallow-confirm"
            cancelTestId="query-log-disallow-cancel"
            onConfirm={props.onConfirm}
            onClose={props.onClose}
        />
    );
};
