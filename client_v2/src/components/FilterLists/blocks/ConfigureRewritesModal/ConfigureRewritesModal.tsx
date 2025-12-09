import React, { useEffect } from 'react';

import intl from 'panel/common/intl';
import { Dialog } from 'panel/common/ui/Dialog/Dialog';
import { MODAL_TYPE } from 'panel/helpers/constants';
import cn from 'clsx';

import { ModalWrapper } from 'panel/common/ui/ModalWrapper';
import { useDispatch, useSelector } from 'react-redux';
import { closeModal } from 'panel/reducers/modals';
import theme from 'panel/lib/theme';
import { Button } from 'panel/common/ui/Button';
import { Controller, FormProvider, useForm } from 'react-hook-form';
import { RootState } from 'panel/initialState';
import { addRewrite, updateRewrite } from 'panel/actions/rewrites';
import { Input } from 'panel/common/controls/Input';
import { validateAnswer, validateDomain, validateRequiredValue } from 'panel/helpers/validators';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import s from './ConfigureRewritesModal.module.pcss';

type FormValues = {
    answer: string;
    domain: string;
    enabled: boolean;
};

const defaultValues: FormValues = {
    answer: '',
    domain: '',
    enabled: false,
};

type ConfigureRewritesModalIdType = 'ADD_REWRITE' | 'EDIT_REWRITE';

type Props = {
    modalId: ConfigureRewritesModalIdType;
    rewriteToEdit?: FormValues;
};

const getTitle = (modalId: ConfigureRewritesModalIdType) => {
    if (modalId === MODAL_TYPE.EDIT_REWRITE) {
        return intl.getMessage('rewrite_edit');
    }

    return intl.getMessage('rewrite_add');
};

const getButtonText = (modalId: ConfigureRewritesModalIdType) => {
    if (modalId === MODAL_TYPE.EDIT_REWRITE) {
        return intl.getMessage('save');
    }

    return intl.getMessage('add');
};

export const ConfigureRewritesModal = ({ modalId, rewriteToEdit }: Props) => {
    const dispatch = useDispatch();
    const { rewrites } = useSelector((state: RootState) => state);
    const { processingAdd, processingUpdate, processing } = rewrites;

    const methods = useForm({
        defaultValues: {
            ...defaultValues,
            ...rewriteToEdit,
        },
        mode: 'onBlur',
    });
    const { handleSubmit, reset, control } = methods;

    useEffect(() => {
        reset({
            ...defaultValues,
            ...rewriteToEdit,
        });
    }, [rewriteToEdit, reset]);

    const handleFormSubmit = async (values: FormValues) => {
        switch (modalId) {
            case MODAL_TYPE.ADD_REWRITE: {
                dispatch(addRewrite({ answer: values.answer, domain: values.domain, enabled: true }));
                dispatch(closeModal());
                break;
            }
            case MODAL_TYPE.EDIT_REWRITE: {
                dispatch(updateRewrite({
                    target: rewriteToEdit,
                    update: { answer: values.answer, domain: values.domain, enabled: values.enabled },
                }));
                dispatch(closeModal());
                break;
            }
            default: {
                break;
            }
        }
    };

    const handleCancel = () => {
        reset(defaultValues);
        dispatch(closeModal());
    };

    return (
        <ModalWrapper id={modalId}>
            <Dialog visible onClose={handleCancel} title={getTitle(modalId)}>
                <FormProvider {...methods}>
                    <form onSubmit={handleSubmit(handleFormSubmit)}>
                        <div>
                            <div className={theme.form.group}>
                                <div className={theme.form.input}>
                                    <Controller
                                        name="domain"
                                        control={control}
                                        rules={{
                                            validate: { validateRequiredValue, validateDomain },
                                        }}
                                        render={({ field, fieldState }) => (
                                            <Input
                                                {...field}
                                                type="text"
                                                id="domain"
                                                data-testid="rewrite-domain-input"
                                                label={
                                                    <>
                                                        {intl.getMessage('upstream_examples_title')}

                                                        <FaqTooltip
                                                            overlayClassName={s.dropdown}
                                                            menuClassName={s.tooltip}
                                                            text={
                                                                <>
                                                                    <div className={cn(theme.text.t3, s.tooltipTitle)}>
                                                                        {intl.getMessage('upstream_examples_title')}
                                                                    </div>

                                                                    {[
                                                                        { message: 'rewrites_tooltip_examples_item1', code: 'example.org' },
                                                                        { message: 'rewrites_tooltip_examples_item2', code: '*.example.org' },
                                                                    ].map((item, index) => (
                                                                        <div key={index} className={s.tooltipItem}>
                                                                            <div className={s.tooltipItemDot}></div>
                                                                            {intl.getMessage(item.message)}
                                                                            <code>{item.code}</code>
                                                                        </div>
                                                                    ))}
                                                                </>
                                                            }
                                                        />
                                                    </>
                                                }
                                                placeholder={intl.getMessage('rewrite_domain_input_placeholder')}
                                                errorMessage={fieldState.error?.message}
                                            />
                                        )}
                                    />
                                </div>

                                <div className={theme.form.input}>
                                    <Controller
                                        name="answer"
                                        control={control}
                                        rules={{
                                            validate: { validateRequiredValue, validateAnswer },
                                        }}
                                        render={({ field, fieldState }) => (
                                            <Input
                                                {...field}
                                                type="text"
                                                id="answer"
                                                data-testid="rewrite-answer-input"
                                                label={
                                                    <>
                                                        {intl.getMessage('instructions')}

                                                        <FaqTooltip
                                                            overlayClassName={s.dropdown}
                                                            menuClassName={s.tooltip}
                                                            text={
                                                                <>
                                                                    <div className={cn(theme.text.t3, s.tooltipTitle)}>
                                                                        {intl.getMessage('instructions')}
                                                                    </div>

                                                                    {[
                                                                        { message: 'rewrites_tooltip_instructions_item1' },
                                                                        { message: 'rewrites_tooltip_instructions_item2' },
                                                                        { message: 'rewrites_tooltip_instructions_item3', code: 'A' },
                                                                        { message: 'rewrites_tooltip_instructions_item4', code: 'AAAA' },
                                                                    ].map((item, index) => (
                                                                        <div key={index} className={s.tooltipItem}>
                                                                            <div className={s.tooltipItemDot}></div>
                                                                            {intl.getMessage(item.message)}

                                                                            {item.code && (
                                                                                <code>{item.code}</code>
                                                                            )}
                                                                        </div>
                                                                    ))}
                                                                </>
                                                            }
                                                        />
                                                    </>
                                                }
                                                placeholder={intl.getMessage('rewrites_answer_input_placeholder')}
                                                errorMessage={fieldState.error?.message}
                                            />
                                        )}
                                    />
                                </div>
                            </div>
                        </div>

                        <div className={theme.dialog.footer}>
                            <Button
                                type="submit"
                                id="save"
                                data-testid="rewrite-save-button"
                                variant="primary"
                                size="small"
                                disabled={processingAdd || processingUpdate || processing}
                                className={theme.dialog.button}
                            >
                                {getButtonText(modalId)}
                            </Button>

                            <Button
                                type="button"
                                id="cancel"
                                data-testid="rewrite-cancel-button"
                                variant="secondary"
                                size="small"
                                onClick={handleCancel}
                                className={theme.dialog.button}
                            >
                                {intl.getMessage('cancel')}
                            </Button>
                        </div>
                    </form>
                </FormProvider>
            </Dialog>
        </ModalWrapper>
    );
};
