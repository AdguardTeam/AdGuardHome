import React from 'react';
import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';

type FormValues = {
    username: string;
    password: string;
}

type LoginFormProps = {
    onSubmit: (data: FormValues) => void;
    processing: boolean;
}

const Form = ({ onSubmit, processing }: LoginFormProps) => {
    const { t } = useTranslation();
    const {
        register,
        handleSubmit,
        formState: { errors, isValid },
    } = useForm<FormValues>({
        mode: 'onChange',
        defaultValues: {
            username: '',
            password: '',
        },
    });

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="card">
            <div className="card-body p-6">
                <div className="form__group form__group--settings">
                    <label className="form__label" htmlFor="username">
                        {t('username_label')}
                    </label>

                    <input
                        id="username"
                        type="text"
                        className="form-control"
                        placeholder={t('username_placeholder')}
                        autoComplete="username"
                        autoCapitalize="none"
                        disabled={processing}
                        {...register('username', {
                            required: t('form_error_required'),
                        })}
                    />
                    {errors.username && (
                        <span className="form__message form__message--error">
                            {errors.username.message}
                        </span>
                    )}
                </div>

                <div className="form__group form__group--settings">
                    <label className="form__label" htmlFor="password">
                        {t('password_label')}
                    </label>

                    <input
                        id="password"
                        type="password"
                        className="form-control"
                        placeholder={t('password_placeholder')}
                        autoComplete="current-password"
                        disabled={processing}
                        {...register('password', {
                            required: t('form_error_required'),
                        })}
                    />
                    {errors.password && (
                        <span className="form__message form__message--error">
                            {errors.password.message}
                        </span>
                    )}
                </div>

                <div className="form-footer">
                    <button
                        type="submit"
                        className="btn btn-success btn-block"
                        disabled={processing || !isValid}
                    >
                        {t('sign_in')}
                    </button>
                </div>
            </div>
        </form>
    );
};

export default Form;