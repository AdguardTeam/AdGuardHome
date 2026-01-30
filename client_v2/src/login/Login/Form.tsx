import React from 'react';
import { Controller, useForm } from 'react-hook-form';
import { useSelector } from 'react-redux';
import { validateRequiredValue } from 'panel/helpers/validators';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import { HTML_PAGES } from 'panel/helpers/constants';
import intl from 'panel/common/intl';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { PasswordInput } from 'panel/common/controls/Input/PasswordInput';
import styles from './styles.module.pcss';

export type LoginFormValues = {
    username: string;
    password: string;
};

type LoginFormProps = {
    onSubmit: (data: LoginFormValues) => void;
    processing: boolean;
};

const Form = ({ onSubmit, processing }: LoginFormProps) => {
    const loginError = useSelector((state: any) => state.login?.error);
    const {
        handleSubmit,
        control,
        setError,
        formState: { isValid },
    } = useForm<LoginFormValues>({
        mode: 'onChange',
        defaultValues: {
            username: '',
            password: '',
        },
    });

    React.useEffect(() => {
        if (!loginError) {
            return;
        }

        setError('password', { type: 'server', message: intl.getMessage('password_login_error') });
    }, [loginError, setError]);

    const handleForgotPassword = () => {
        window.location.assign(HTML_PAGES.FORGOT_PASSWORD);
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="card">
            <div className={styles.formContainer}>
                <div className={styles.group}>
                    <Controller
                        name="username"
                        control={control}
                        rules={{ validate: validateRequiredValue }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                id="username"
                                type="text"
                                label={intl.getMessage('username_label')}
                                placeholder={intl.getMessage('username_placeholder')}
                                errorMessage={fieldState.error?.message}
                                autoComplete="username"
                                autoCapitalize="none"
                            />
                        )}
                    />
                </div>

                <div className={styles.group}>
                    <Controller
                        name="password"
                        control={control}
                        rules={{ validate: validateRequiredValue,  }}
                        render={({ field, fieldState }) => (
                            <PasswordInput
                                {...field}
                                id="password"
                                label={intl.getMessage('password_label')}
                                placeholder={intl.getMessage('password_placeholder')}
                                inputError={fieldState.error?.message}
                                autoComplete="current-password"
                            />
                        )}
                    />
                </div>

                <div className={styles.footer}>
                    <Button className={styles.button} id="sign_in" type="submit" variant="primary" size="small" disabled={processing || !isValid}>
                        {intl.getMessage('login')}
                    </Button>

                    <div className={styles.info}>
                        <button type="button" className={cn(theme.link.link, 'link')} onClick={handleForgotPassword}>
                            {intl.getMessage('forgot_password')}
                        </button>
                    </div>
                </div>
            </div>
        </form>
    );
};

export default Form;
