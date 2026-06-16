import { createSignal, createMemo, For } from 'solid-js';
import type { JSX } from 'solid-js';

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
    faq: JSX.Element;
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

export const Form = (props: FormProps) => {
    const [allowedClients, setAllowedClients] = createSignal(
        props.initialValues?.allowed_clients || '',
    );
    const [disallowedClients, setDisallowedClients] = createSignal(
        props.initialValues?.disallowed_clients || '',
    );
    const [blockedHosts, setBlockedHosts] = createSignal(
        props.initialValues?.blocked_hosts || '',
    );

    const [allowedClientsError, setAllowedClientsError] = createSignal('');
    const [disallowedClientsError, setDisallowedClientsError] = createSignal('');

    const getFieldValue = (id: keyof FormData) => {
        switch (id) {
            case 'allowed_clients': return allowedClients();
            case 'disallowed_clients': return disallowedClients();
            case 'blocked_hosts': return blockedHosts();
        }
    };

    const setFieldValue = (id: keyof FormData, value: string) => {
        switch (id) {
            case 'allowed_clients': setAllowedClients(value); break;
            case 'disallowed_clients': setDisallowedClients(value); break;
            case 'blocked_hosts': setBlockedHosts(value); break;
        }
    };

    const getFieldError = (id: keyof FormData) => {
        switch (id) {
            case 'allowed_clients': return allowedClientsError();
            case 'disallowed_clients': return disallowedClientsError();
            default: return '';
        }
    };

    const validateField = (id: keyof FormData) => {
        const isIpField = id === 'allowed_clients' || id === 'disallowed_clients';
        if (!isIpField) return;
        const value = getFieldValue(id);
        const err = value ? validateIpPerLine(value) : undefined;
        if (id === 'allowed_clients') setAllowedClientsError(err || '');
        if (id === 'disallowed_clients') setDisallowedClientsError(err || '');
    };

    const handleDisabledFieldState = (id: string) => {
        return id === 'disallowed_clients' && !!allowedClients();
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

    const handleSubmit = (e: Event) => {
        e.preventDefault();
        validateField('allowed_clients');
        validateField('disallowed_clients');

        if (allowedClientsError() || disallowedClientsError()) {
            return;
        }

        props.onSubmit({
            allowed_clients: allowedClients(),
            disallowed_clients: disallowedClients(),
            blocked_hosts: blockedHosts(),
        });
    };

    return (
        <form onSubmit={handleSubmit} class={theme.form.form}>
            <div class={theme.form.group}>
                <For each={fields}>
                    {(f) => (
                        <div class={theme.form.input}>
                            <Textarea
                                value={getFieldValue(f.id)}
                                onChange={(e: Event) => setFieldValue(f.id, (e.target as HTMLTextAreaElement).value)}
                                onBlur={() => {
                                    const field = fields.find((ff) => ff.id === f.id);
                                    if (field) {
                                        setFieldValue(f.id, field.normalizeOnBlur(getFieldValue(f.id)));
                                    }
                                    validateField(f.id);
                                }}
                                id={f.id}
                                data-testid={f.id}
                                label={
                                    <>
                                        {f.title}
                                        <FaqTooltip
                                            text={f.faq}
                                            menuSize="large"
                                            spacing={f.id === 'blocked_hosts'}
                                        />
                                    </>
                                }
                                errorMessage={getFieldError(f.id)}
                                size="medium"
                                disabled={handleDisabledFieldState(f.id)}
                                placeholder={getPlaceholder(f.id)}
                            />
                        </div>
                    )}
                </For>
            </div>

            <div class={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    id="access_save"
                    variant="primary"
                    size="small"
                    disabled={props.processingSet}
                    class={theme.form.button}
                >
                    {intl.getMessage('save')}
                </Button>
            </div>
        </form>
    );
};
