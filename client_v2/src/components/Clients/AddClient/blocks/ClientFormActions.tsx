import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import { clientFormState, saveClient } from 'panel/stores/clientForm';

import s from './ClientFormActions.module.pcss';

type ClientFormActionsProps = {
    onCancel: () => void;
    onSaved?: (clientName: string) => void;
};

/**
 * The Save/Cancel footer shared by the client form and its sub-pages.
 *
 * The sub-pages edit the same draft as the parent form, so saving there submits
 * the whole client configuration.  Saves stay explicit to avoid pushing fields
 * the operator has not finished editing on the parent page.
 */
export const ClientFormActions = (props: ClientFormActionsProps) => {
    const handleSave = async () => {
        const clientName = clientFormState.originalName;
        const saved = await saveClient();

        if (saved) {
            props.onSaved?.(clientName);
        }
    };

    return (
        <div class={s.actions}>
            <Button
                variant="primary"
                size="small"
                onClick={handleSave}
                disabled={clientFormState.processingSave}
                data-testid="client-form-save"
                class={s.actionButton}
            >
                {intl.getMessage('save_btn')}
            </Button>
            <Button
                variant="secondary"
                size="small"
                onClick={props.onCancel}
                data-testid="client-form-cancel"
                class={s.actionButton}
            >
                {intl.getMessage('cancel_btn')}
            </Button>
        </div>
    );
};
