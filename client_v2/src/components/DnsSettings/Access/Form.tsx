import React, { ReactNode } from 'react';
import { Controller, useForm } from 'react-hook-form';

import intl from 'panel/common/intl';
import { Textarea } from 'panel/common/controls/Textarea';
import { Button } from 'panel/common/ui/Button';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { CLIENT_ID_LINK } from 'panel/helpers/constants';
import { removeEmptyLines, trimMultilineString } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';

type FormData = {
    allowed_clients: string;
    disallowed_clients: string;
    blocked_hosts: string;
};

const fields: {
    id: keyof FormData;
    title: string;
    faq: ReactNode;
    normalizeOnBlur: (value: string) => string;
}[] = [
    {
        id: 'allowed_clients',
        title: intl.getMessage('access_settings_allowed_title'),
        faq: intl.getMessage('access_settings_allowed_faq', {
            a: (text: string) => (
                <a href={CLIENT_ID_LINK} target="_blank" rel="noopener noreferrer">
                    {text}
                </a>
            ),
        }),
        normalizeOnBlur: removeEmptyLines,
    },
    {
        id: 'disallowed_clients',
        title: intl.getMessage('access_settings_disallowed_title'),
        faq: intl.getMessage('access_settings_disallowed_faq', {
            a: (text: string) => (
                <a href={CLIENT_ID_LINK} target="_blank" rel="noopener noreferrer">
                    {text}
                </a>
            ),
        }),
        normalizeOnBlur: trimMultilineString,
    },
    {
        id: 'blocked_hosts',
        title: intl.getMessage('access_settings_blocked_title'),
        faq: (
            <>
                <div>{intl.getMessage('access_settings_blocked_faq_1')}</div>
                <div>{intl.getMessage('access_settings_blocked_faq_2')}</div>
            </>
        ),
        normalizeOnBlur: removeEmptyLines,
    },
];

type FormProps = {
    initialValues?: {
        allowed_clients?: string;
        disallowed_clients?: string;
        blocked_hosts?: string;
    };
    onSubmit: (data: FormData) => void;
    processingSet: boolean;
};

export const Form = ({ initialValues, onSubmit, processingSet }: FormProps) => {
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
        faq,
        normalizeOnBlur,
    }: {
        id: keyof FormData;
        title: string;
        faq: ReactNode;
        normalizeOnBlur: (value: string) => string;
    }) => {
        const disabled = allowedClients && id === 'disallowed_clients';

        return (
            <div key={id} className={theme.form.input}>
                <Controller
                    name={id}
                    control={control}
                    render={({ field }) => (
                        <Textarea
                            {...field}
                            id={id}
                            data-testid={id}
                            label={
                                <>
                                    {title}
                                    {disabled && <>&nbsp;({intl.getMessage('disabled')})</>}
                                    <FaqTooltip text={faq} menuSize="large" spacing={id === 'blocked_hosts'} />
                                </>
                            }
                            disabled={disabled || processingSet}
                            onBlur={(e) => {
                                field.onChange(normalizeOnBlur(e.target.value));
                            }}
                            size="medium"
                        />
                    )}
                />
            </div>
        );
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)} className={theme.form.form}>
            <div className={theme.form.group}>{fields.map((f) => renderField(f))}</div>

            <div className={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    id="access_save"
                    variant="primary"
                    size="small"
                    disabled={isSubmitting || !isDirty || processingSet}
                    className={theme.form.button}>
                    {intl.getMessage('save')}
                </Button>
            </div>
        </form>
    );
};
