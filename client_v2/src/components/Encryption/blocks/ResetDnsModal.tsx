import { Show } from 'solid-js';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import intl from 'panel/common/intl';
import { setTlsConfig, validateTlsConfig, resetValidationStatus } from 'panel/stores/encryption';
import { defaultTlsValues } from './helpers';

type Props = {
    open: boolean;
    onClose: () => void;
};

export const ResetDnsModal = (props: Props) => (
    <Show when={props.open}>
        <ConfirmDialog
            title={intl.getMessage('reset_dns_confirm_title')}
            text={intl.getMessage('reset_dns_confirm_text')}
            buttonText={intl.getMessage('yes_reset')}
            cancelText={intl.getMessage('cancel')}
            buttonVariant="danger"
            onClose={props.onClose}
            onConfirm={() => {
                resetValidationStatus();
                setTlsConfig(defaultTlsValues);
                validateTlsConfig(defaultTlsValues);
                props.onClose();
            }}
        />
    </Show>
);
