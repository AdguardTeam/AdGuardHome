import React, { ReactNode } from 'react';
import { Controller, useForm } from 'react-hook-form';

import intl from 'panel/common/intl';
import { Textarea } from 'panel/common/controls/Textarea';
import { Button } from 'panel/common/ui/Button';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { CLIENT_ID_LINK } from 'panel/helpers/constants';
import { validateIpPerLine } from 'panel/helpers/validators';
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
        formState: { isSubmitting },
    } = useForm<FormData>({
        mode: 'onBlur',
        defaultValues: {
            allowed_clients: initialValues?.allowed_clients || '',
            disallowed_clients: initialValues?.disallowed_clients || '',
            blocked_hosts: initialValues?.blocked_hosts || '',
        },
    });

    const allowedClientsValue = watch('allowed_clients');
    const handleDisabledFieldState = (id: string) => {
        return id === 'disallowed_clients' && !!allowedClientsValue;
    };

    const getPlaceholder = (id: string) => {
        if (id === 'allowed_clients') {
            return intl.getMessage('access_settings_allowed_placeholder');
        }

        if (id === 'disallowed_clients') {
            return intl.getMessage('access_settings_disallowed_placeholder');
        }

        if (id === 'blocked_hosts') {
            return intl.getMessage('access_settings_blocked_placeholder');
        }

        return '';
    };

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
        const isIpField = id === 'allowed_clients' || id === 'disallowed_clients';
        const placeholder = getPlaceholder(id);

        return (
            <div key={id} className={theme.form.input}>
                <Controller
                    name={id}
                    control={control}
                    rules={isIpField ? { validate: validateIpPerLine } : undefined}
                    render={({ field, fieldState }) => (
                        <Textarea
                            {...field}
                            id={id}
                            data-testid={id}
                            label={
                                <>
                                    {title}
                                    <FaqTooltip
                                        text={faq}
                                        menuSize="large"
                                        spacing={id === 'blocked_hosts'}
                                    />
                                </>
                            }
                            errorMessage={fieldState.error?.message}
                            onBlur={(e) => {
                                field.onChange(normalizeOnBlur(e.target.value));
                            }}
                            size="medium"
                            disabled={handleDisabledFieldState(id)}
                            placeholder={placeholder}
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
                    disabled={isSubmitting || processingSet}
                    className={theme.form.button}
                >
                    {intl.getMessage('save')}
                </Button>
            </div>
        </form>
    );
};
