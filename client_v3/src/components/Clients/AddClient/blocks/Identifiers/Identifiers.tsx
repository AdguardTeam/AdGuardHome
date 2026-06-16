import { createSignal, createMemo, createEffect, For, Show } from 'solid-js';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import { Icon } from 'panel/common/ui/Icon';
import type { Client } from 'panel/initialState';
import { clientFormState, updateClientFormField } from 'panel/stores/clientForm';
import { dashboardState } from 'panel/stores/dashboard';
import { validateIdentifier } from 'panel/helpers/validators';
import theme from 'panel/lib/theme';

import s from './Identifiers.module.pcss';

export const Identifiers = () => {
    const [ids, setIds] = createSignal<string[]>([...clientFormState.ids]);
    const [errors, setErrors] = createSignal<(string | undefined)[]>([]);

    // Sync from store when external changes happen
    createEffect(() => {
        const storeIds = clientFormState.ids;
        const currentIds = ids();
        if (JSON.stringify(currentIds) !== JSON.stringify(storeIds)) {
            setIds([...storeIds]);
        }
    });

    // Sync external errors
    createEffect(() => {
        const formErrors = clientFormState.formErrors;
        if (Array.isArray(formErrors.ids)) {
            setErrors(formErrors.ids);
        }
    });

    const existingClientIds = createMemo(() => {
        const clients: Client[] = dashboardState.clients || [];
        const isEdit = clientFormState.mode === 'edit';
        return clients
            .filter((c) => !isEdit || c.name !== clientFormState.originalName)
            .flatMap((c) => c.ids);
    });

    const syncToStore = () => {
        updateClientFormField('ids', ids());
    };

    const handleAdd = () => {
        const newIds = [...ids(), ''];
        setIds(newIds);
        syncToStore();
    };

    const handleRemove = (index: number) => {
        const newIds = ids().filter((_, i) => i !== index);
        setIds(newIds);
        const newErrors = errors().filter((_, i) => i !== index);
        setErrors(newErrors);
        syncToStore();
    };

    const handleChange = (index: number, value: string) => {
        const newIds = [...ids()];
        newIds[index] = value;
        setIds(newIds);
        syncToStore();
    };

    const handleBlur = (index: number) => {
        const value = ids()[index];
        const err = validateIdentifier(value, ids(), index, existingClientIds());
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

            <For each={ids()}>
                {(value, index) => {
                    const idx = index();
                    const activeError = createMemo(() => errors()[idx]);

                    return (
                        <div class={s.row}>
                            <div class={s.inputCell}>
                                <Input
                                    id={`client-identifier-${idx}`}
                                    type="text"
                                    value={value}
                                    onChange={(e: Event) => handleChange(idx, (e.target as HTMLInputElement).value)}
                                    onBlur={() => handleBlur(idx)}
                                    placeholder={intl.getMessage('clients_identifier_format_error')}
                                    error={!!activeError()}
                                    errorMessage={activeError()}
                                    size="large"
                                    suffixIcon={
                                        idx > 0 ? (
                                            <button
                                                type="button"
                                                class={s.removeSuffixBtn}
                                                onClick={() => handleRemove(idx)}
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
            </For>

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
