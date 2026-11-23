import { useNavigate } from '@solidjs/router';
import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import { Paths, RoutePath, linkPathBuilder } from 'panel/components/Routes/Paths';
import { clientFormState, clearClientForm, saveClient } from 'panel/stores/clientForm';
import { addErrorToast } from 'panel/stores/toasts';

import s from './ClientFormActions.module.pcss';

type ClientFormActionsProps = {
    /** Overrides the default Cancel navigation. */
    onCancel?: () => void;
};

/**
 * The Save/Cancel footer shared by the client form and its sub-pages.
 *
 * The sub-pages edit the same draft as the parent form, so saving there submits
 * the whole client configuration.  Cancel defaults to the client edit page;
 * callers override it where their target differs.
 */
export const ClientFormActions = (props: ClientFormActionsProps) => {
    const navigate = useNavigate();

    const handleCancel = () => {
        if (props.onCancel) {
            props.onCancel();
            return;
        }

        const clientName = clientFormState.originalName;
        clearClientForm();
        if (!clientName) {
            navigate(Paths.Clients);
            return;
        }

        navigate(
            linkPathBuilder(RoutePath.ClientsEdit, {
                clientName: encodeURIComponent(clientName),
            }),
        );
    };

    const handleSave = async () => {
        const saved = await saveClient();

        if (saved) {
            navigate(Paths.Clients);
            return;
        }

        // Validation is the one failure `saveClient` does not toast (it clears
        // `formErrors` before a request); the sub-pages render no fields for it.
        const firstError = Object.values(clientFormState.formErrors)
            .flat()
            .find((message): message is string => typeof message === 'string');

        if (firstError) {
            addErrorToast({ error: new Error(firstError) });
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
                onClick={handleCancel}
                data-testid="client-form-cancel"
                class={s.actionButton}
            >
                {intl.getMessage('cancel_btn')}
            </Button>
        </div>
    );
};
