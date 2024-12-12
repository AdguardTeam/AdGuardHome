import React from 'react';
import { useForm } from 'react-hook-form';
import { Trans, useTranslation } from 'react-i18next';

import { validateAnswer, validateDomain } from '../../../helpers/validators';

interface FormValues {
    domain: string;
    answer: string;
}

type Props = {
    processingAdd: boolean;
    currentRewrite?: { answer: string, domain: string; };
    toggleRewritesModal: () => void;
    onSubmit?: (data: FormValues) => Promise<void> | void;
}

const Form = ({ processingAdd, currentRewrite, toggleRewritesModal, onSubmit }: Props) => {
    const { t } = useTranslation();

    const {
        register,
        handleSubmit,
        reset,
        formState: { isDirty, isSubmitting, errors },
    } = useForm<FormValues>({
        mode: 'onChange', 
        defaultValues: {
            domain: currentRewrite?.domain || '',
            answer: currentRewrite?.answer || '',
        },
    });

    const handleFormSubmit = async (data: FormValues) => {
        if (onSubmit) {
            await onSubmit(data);
        }
    };

    return (
        <form onSubmit={handleSubmit(handleFormSubmit)}>
            <div className="modal-body">
                <div className="form__desc form__desc--top">
                    <Trans>domain_desc</Trans>
                </div>
                <div className="form__group">
                    <input
                        id="domain"
                        type="text"
                        className="form-control"
                        placeholder={t('form_domain')}
                        {...register('domain', {
                            validate: validateDomain,
                            required: t('form_error_required'),
                        })}
                    />
                    {errors.domain && (
                        <div className="form__message form__message--error">
                            {errors.domain.message}
                        </div>
                    )}
                </div>
                <Trans>examples_title</Trans>:
                <ol className="leading-loose">
                    <li>
                        <code>example.org</code> – <Trans>example_rewrite_domain</Trans>
                    </li>
                    <li>
                        <code>*.example.org</code> –&nbsp;
                        <span>
                            <Trans components={[<code key="0">text</code>]}>example_rewrite_wildcard</Trans>
                        </span>
                    </li>
                </ol>
                <div className="form__group">
                    <input
                        id="answer"
                        type="text"
                        className="form-control"
                        placeholder={t('form_answer') ?? ''}
                        {...register('answer', {
                            validate: validateAnswer,
                            required: t('form_error_required'),
                        })}
                    />
                    {errors.answer && (
                        <div className="form__message form__message--error">
                            {errors.answer.message}
                        </div>
                    )}
                </div>
            </div>

            <ul>
                {['rewrite_ip_address', 'rewrite_domain_name', 'rewrite_A', 'rewrite_AAAA'].map((str) => (
                    <li key={str}>
                        <Trans components={[<code key="0">text</code>]}>{str}</Trans>
                    </li>
                ))}
            </ul>

            <div className="modal-footer">
                <div className="btn-list">
                    <button
                        type="button"
                        className="btn btn-secondary btn-standard"
                        disabled={isSubmitting || processingAdd}
                        onClick={() => {
                            reset();
                            toggleRewritesModal();
                        }}
                    >
                        <Trans>cancel_btn</Trans>
                    </button>

                    <button
                        type="submit"
                        className="btn btn-success btn-standard"
                        disabled={isSubmitting || !isDirty || processingAdd}
                    >
                        <Trans>save_btn</Trans>
                    </button>
                </div>
            </div>
        </form>
    );
};

export default Form;
