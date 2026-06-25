import { createSignal, createMemo, createEffect, Index } from 'solid-js';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import { Icon } from 'panel/common/ui/Icon';
import {
    clientFormState,
    updateClientFormField,
    computeExistingClientIds,
} from 'panel/stores/clientForm';
import { validateIdentifier } from 'panel/helpers/validators';
import theme from 'panel/lib/theme';

import s from './Identifiers.module.pcss';

export const Identifiers = () => {
    const [errors, setErrors] = createSignal<(string | undefined)[]>([]);

    // The store (`clientFormState.ids`) is the single source of truth for the
    // identifiers list. We bind to it directly (the same pattern the Name field
    // uses) and update it through `updateClientFormField`.
    //
    // A previous implementation kept a local `ids` signal and tried to sync it
    // with the store via a `createEffect`. In Solid, `setIds` notifies effects
    // synchronously, so that effect ran *before* `syncToStore` had a chance to
    // update the store, saw a mismatch and reset the local value back to the
    // stale store value — which made typed values erase immediately and the
    // "add identifier" button appear to do nothing.

    // Sync external (server) errors into the local per-index error state.
    createEffect(() => {
        const formErrors = clientFormState.formErrors;
        if (Array.isArray(formErrors.ids)) {
            setErrors(formErrors.ids);
        } else {
            // Store errors were cleared — clear stale local errors too.
            setErrors((prev) => prev.map((): undefined => undefined));
        }
    });

    const existingClientIds = createMemo(() => computeExistingClientIds());

    const handleAdd = () => {
        updateClientFormField('ids', [...clientFormState.ids, '']);
    };

    const handleRemove = (index: number) => {
        const newIds = clientFormState.ids.filter((_, i) => i !== index);
        updateClientFormField('ids', newIds);
        const newErrors = errors().filter((_, i) => i !== index);
        setErrors(newErrors);
    };

    const handleChange = (index: number, value: string) => {
        const newIds = [...clientFormState.ids];
        newIds[index] = value;
        updateClientFormField('ids', newIds);
        // Clear the local error for this index — re-validated on blur.
        setErrors((prev) => {
            const next = [...prev];
            next[index] = undefined;
            return next;
        });
    };

    const handleBlur = (index: number) => {
        const ids = clientFormState.ids;
        const value = ids[index];
        const err = validateIdentifier(value, ids, index, existingClientIds());
        const newErrors = [...errors()];
        newErrors[index] = err || undefined;
        setErrors(newErrors);
    };

    return (
        <div class={s.wrapper}>
            <div class={cn(theme.text.t2, theme.text.semibold, s.label)}>
                {intl.getMessage('clients_identifiers')}
            </div>
            <div class={cn(theme.text.t3, s.desc)}>
                {intl.getMessage('clients_identifiers_desc', {
                    a: (text: string) => (
                        <a
                            href="https://github.com/AdguardTeam/AdGuardHome/wiki/Clients#idclient"
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            {text}
                        </a>
                    ),
                })}
            </div>

            <Index each={clientFormState.ids}>
                {(value, index) => {
                    const activeError = createMemo(() => errors()[index]);

                    return (
                        <div class={s.row}>
                            <div class={s.inputCell}>
                                <Input
                                    id={`client-identifier-${index}`}
                                    type="text"
                                    value={value()}
                                    onChange={(e: Event) =>
                                        handleChange(index, (e.target as HTMLInputElement).value)
                                    }
                                    onInput={(e: Event) =>
                                        handleChange(index, (e.target as HTMLInputElement).value)
                                    }
                                    onBlur={() => handleBlur(index)}
                                    placeholder={intl.getMessage('clients_identifier_format_error')}
                                    error={!!activeError()}
                                    errorMessage={activeError()}
                                    size="large"
                                    suffixIcon={
                                        index > 0 ? (
                                            <button
                                                type="button"
                                                class={s.removeSuffixBtn}
                                                onClick={() => handleRemove(index)}
                                                aria-label={intl.getMessage('delete_btn')}
                                            >
                                                <Icon icon="cross" color="gray" />
                                            </button>
                                        ) : undefined
                                    }
                                />
                            </div>
                        </div>
                    );
                }}
            </Index>

            <button
                type="button"
                class={s.addButton}
                onClick={handleAdd}
                data-testid="client-form-add-identifier"
            >
                <Icon icon="plus" color="green" />
                {intl.getMessage('clients_add_identifier')}
            </button>
        </div>
    );
};
