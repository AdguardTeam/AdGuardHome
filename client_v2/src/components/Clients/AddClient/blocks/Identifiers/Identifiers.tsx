import React, { useCallback, useEffect, useMemo } from 'react';
import { useFieldArray, useForm } from 'react-hook-form';
import { useDispatch, useSelector } from 'react-redux';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import { Icon } from 'panel/common/ui/Icon';
import { RootState, Client } from 'panel/initialState';
import { updateClientFormField } from 'panel/actions/clientForm';
import { validateIdentifier } from 'panel/helpers/validators';
import theme from 'panel/lib/theme';

import s from './Identifiers.module.pcss';

type FormValues = {
    ids: { value: string }[];
};

export const Identifiers = () => {
    const dispatch = useDispatch();
    const formState = useSelector((state: RootState) => state.clientForm);
    const dashboard = useSelector((state: RootState) => state.dashboard);
    const { formErrors } = formState;

    // Collect all identifiers from existing persistent clients, excluding the
    // client currently being edited (if any).
    const existingClientIds = useMemo(() => {
        const clients: Client[] = dashboard?.clients || [];
        const isEdit = formState.mode === 'edit';
        return clients
            .filter((c) => !isEdit || c.name !== formState.originalName)
            .flatMap((c) => c.ids);
    }, [dashboard?.clients, formState.mode, formState.originalName]);

    const {
        control,
        register,
        formState: rhfState,
        setValue,
        getValues,
        trigger,
    } = useForm<FormValues>({
        defaultValues: {
            ids: formState.ids.map((id: string) => ({ value: id })),
        },
        mode: 'onBlur',
    });

    const { fields, append, remove } = useFieldArray({
        control,
        name: 'ids',
    });

    // Sync Redux → RHF when external errors arrive (e.g. from saveClient)
    useEffect(() => {
        if (Array.isArray(formErrors.ids)) {
            formErrors.ids.forEach((err: string | undefined, idx: number) => {
                if (err) {
                    trigger(`ids.${idx}.value`);
                }
            });
        }
    }, [formErrors.ids, trigger]);

    const syncToRedux = () => {
        const values = getValues('ids').map((item) => item.value);
        dispatch(updateClientFormField({ field: 'ids', value: values }));
    };

    const handleAdd = () => {
        append({ value: '' });
        const values = getValues('ids').map((item) => item.value);
        dispatch(updateClientFormField({ field: 'ids', value: [...values, ''] }));
    };

    const handleRemove = (index: number) => {
        remove(index);
        const values = getValues('ids')
            .filter((_: { value: string }, i: number) => i !== index)
            .map((item) => item.value);
        dispatch(updateClientFormField({ field: 'ids', value: values }));
    };

    const handleValidate = useCallback(
        (value: string, index: number) => {
            const allValues = getValues('ids').map((item) => item.value);
            return validateIdentifier(value, allValues, index, existingClientIds) || true;
        },
        [getValues, existingClientIds],
    );

    return (
        <div className={s.wrapper}>
            <div className={cn(theme.text.t2, theme.text.semibold, s.label)}>
                {intl.getMessage('clients_identifiers')}
            </div>
            <div className={cn(theme.text.t3, s.desc)}>
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

            {fields.map((field, index) => {
                const saveError = Array.isArray(formErrors.ids) ? formErrors.ids[index] : undefined;
                const rhfError = rhfState.errors.ids?.[index]?.value?.message;
                const activeError = saveError || rhfError;

                const suffixBtn =
                    index > 0 ? (
                        <button
                            type="button"
                            className={s.removeSuffixBtn}
                            onClick={() => handleRemove(index)}
                            aria-label={intl.getMessage('delete_btn')}
                        >
                            <Icon icon="cross" color="gray" />
                        </button>
                    ) : undefined;

                return (
                    <div key={field.id} className={s.row}>
                        <div className={s.inputCell}>
                            <Input
                                id={`client-identifier-${index}`}
                                type="text"
                                {...register(`ids.${index}.value`, {
                                    validate: (value: string) => handleValidate(value, index),
                                    onBlur: () => syncToRedux(),
                                })}
                                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                    setValue(`ids.${index}.value`, e.target.value);
                                    syncToRedux();
                                }}
                                placeholder={intl.getMessage('clients_identifier_format_error')}
                                error={!!activeError}
                                errorMessage={activeError}
                                size="large"
                                suffixIcon={suffixBtn}
                            />
                        </div>
                    </div>
                );
            })}
            <button
                type="button"
                className={s.addButton}
                onClick={handleAdd}
                data-testid="client-form-add-identifier"
            >
                <Icon icon="plus" color="green" />
                {intl.getMessage('clients_add_identifier')}
            </button>
        </div>
    );
};
