import { createEffect } from 'solid-js';
import { createForm, required, setError, setValue } from '@modular-forms/solid';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { HTML_PAGES } from 'panel/helpers/constants';
import intl from 'panel/common/intl';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { PasswordInput } from 'panel/common/controls/Input/PasswordInput';
import { loginState } from 'panel/stores/login';
import styles from './styles.module.pcss';

export type LoginFormValues = {
    username: string;
    password: string;
};

type LoginFormProps = {
    onSubmit: (data: LoginFormValues) => void;
};

const Form = (props: LoginFormProps) => {
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

    // TODO: replace with link
    const handleForgotPassword = () => {
        window.location.assign(HTML_PAGES.FORGOT_PASSWORD);
    };

    return (
        <Form onSubmit={handleSubmit} class="card">
            <div class={styles.formContainer}>
                <div class={styles.group}>
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
                            />
                        )}
                    </Field>
                </div>

                <div class={styles.group}>
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
                            />
                        )}
                    </Field>
                </div>

                <div class={styles.footer}>
                    <Button
                        class={styles.button}
                        id="sign_in"
                        type="submit"
                        variant="primary"
                        size="small"
                        disabled={loginState.processingLogin}
                    >
                        {intl.getMessage('login')}
                    </Button>

                    <div class={styles.info}>
                        <button
                            type="button"
                            class={cn(theme.link.link, theme.text.t2)}
                            onClick={handleForgotPassword}
                        >
                            {intl.getMessage('forgot_password')}
                        </button>
                    </div>
                </div>
            </div>
        </Form>
    );
};

export default Form;
