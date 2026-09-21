import { createEffect } from 'solid-js';
import { createForm, required, setError, setValue } from '@modular-forms/solid';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { HTML_PAGES } from 'panel/helpers/constants';
import intl from 'panel/common/intl';

import theme from 'panel/lib/theme';
import { PasswordInput } from 'panel/common/controls/Input/PasswordInput';
import { loginState } from 'panel/stores/login';

export type LoginFormValues = {
    username: string;
    password: string;
};

type Props = {
    onSubmit: (data: LoginFormValues) => void;
};

export const Form = (props: Props) => {
    const [loginForm, { Form, Field }] = createForm<LoginFormValues>({
        validateOn: 'input',
    });

    createEffect(() => {
        if (loginState.error) {
            setError(loginForm, 'password', intl.getMessage('password_login_error'));
        }
    });

    const handleSubmit = (values: LoginFormValues) => {
        props.onSubmit(values);
    };

    return (
        <Form onSubmit={handleSubmit}>
            <div class={theme.auth.group}>
                <Field
                    name="username"
                    validate={[required(intl.getMessage('form_error_required'))]}
                >
                    {(field, fieldProps) => (
                        <Input
                            {...fieldProps}
                            id="username"
                            type="text"
                            value={(field.value as string) || ''}
                            label={intl.getMessage('username_label')}
                            placeholder={intl.getMessage('username_placeholder')}
                            errorMessage={field.error as string}
                            autocomplete="username"
                            autocapitalize="none"
                            size="large"
                            onCard
                        />
                    )}
                </Field>
            </div>

            <div class={theme.auth.group}>
                <Field
                    name="password"
                    validate={[required(intl.getMessage('form_error_required'))]}
                >
                    {(field, fieldProps) => (
                        <PasswordInput
                            {...fieldProps}
                            id="password"
                            value={(field.value as string) || ''}
                            label={intl.getMessage('password_label')}
                            placeholder={intl.getMessage('password_placeholder')}
                            inputError={field.error as string}
                            autocomplete="current-password"
                            onChange={(value: string) => setValue(loginForm, 'password', value)}
                            size="large"
                            onCard
                        />
                    )}
                </Field>
            </div>

            <div class={theme.auth.footer}>
                <div class={theme.auth.footerRow}>
                    <Button
                        class={theme.auth.footerButton}
                        id="sign_in"
                        type="submit"
                        variant="primary"
                        size="small"
                        disabled={loginState.processingLogin}
                    >
                        {intl.getMessage('login')}
                    </Button>

                    <a href={HTML_PAGES.FORGOT_PASSWORD} class={theme.auth.footerLink}>
                        {intl.getMessage('forgot_password')}
                    </a>
                </div>
            </div>
        </Form>
    );
};
