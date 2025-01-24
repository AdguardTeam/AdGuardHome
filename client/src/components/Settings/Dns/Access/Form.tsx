import React from 'react';
import { Controller, useForm } from 'react-hook-form';
import { Trans, useTranslation } from 'react-i18next';

import { CLIENT_ID_LINK } from '../../../../helpers/constants';
import { removeEmptyLines, trimMultilineString } from '../../../../helpers/helpers';
import { Textarea } from '../../../ui/Controls/Textarea';

const fields = [
    {
        id: 'allowed_clients',
        title: 'access_allowed_title',
        subtitle: 'access_allowed_desc',
        normalizeOnBlur: removeEmptyLines,
    },
    {
        id: 'disallowed_clients',
        title: 'access_disallowed_title',
        subtitle: 'access_disallowed_desc',
        normalizeOnBlur: trimMultilineString,
    },
    {
        id: 'blocked_hosts',
        title: 'access_blocked_title',
        subtitle: 'access_blocked_desc',
        normalizeOnBlur: removeEmptyLines,
    },
];

interface FormProps {
    initialValues?: {
        allowed_clients?: string;
        disallowed_clients?: string;
        blocked_hosts?: string;
    };
    onSubmit: (data: any) => void;
    processingSet: boolean;
}

interface FormData {
    allowed_clients: string;
    disallowed_clients: string;
    blocked_hosts: string;
}

const Form = ({ initialValues, onSubmit, processingSet }: FormProps) => {
    const { t } = useTranslation();

    const {
        control,
        handleSubmit,
        watch,
        formState: { isSubmitting, isDirty },
    } = useForm<FormData>({
        mode: 'onBlur',
        defaultValues: {
            allowed_clients: initialValues?.allowed_clients || '',
            disallowed_clients: initialValues?.disallowed_clients || '',
            blocked_hosts: initialValues?.blocked_hosts || '',
        },
    });

    const allowedClients = watch('allowed_clients');

    const renderField = ({
        id,
        title,
        subtitle,
        normalizeOnBlur,
    }: {
        id: keyof FormData;
        title: string;
        subtitle: string;
        normalizeOnBlur: (value: string) => string;
    }) => {
        const disabled = allowedClients && id === 'disallowed_clients';

        return (
            <div key={id} className="form__group mb-5">
                <label className="form__label form__label--with-desc" htmlFor={id}>
                    {t(title)}
                    {disabled && (
                        <>
                            <span> </span>({t('disabled')})
                        </>
                    )}
                </label>

                <div className="form__desc form__desc--top">
                    <Trans
                        components={{
                            a: <a href={CLIENT_ID_LINK} target="_blank" rel="noopener noreferrer" />,
                        }}>
                        {subtitle}
                    </Trans>
                </div>

                <Controller
                    name={id}
                    control={control}
                    render={({ field }) => (
                        <Textarea
                            {...field}
                            id={id}
                            disabled={disabled || processingSet}
                            onBlur={(e) => {
                                field.onChange(normalizeOnBlur(e.target.value));
                            }}
                        />
                    )}
                />
            </div>
        );
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            {fields.map((f) =>
                renderField(
                    f as {
                        id: keyof FormData;
                        title: string;
                        subtitle: string;
                        normalizeOnBlur: (value: string) => string;
                    },
                ),
            )}

            <div className="card-actions">
                <div className="btn-list">
                    <button
                        type="submit"
                        className="btn btn-success btn-standard"
                        disabled={isSubmitting || !isDirty || processingSet}>
                        {t('save_config')}
                    </button>
                </div>
            </div>
        </form>
    );
};

export default Form;
