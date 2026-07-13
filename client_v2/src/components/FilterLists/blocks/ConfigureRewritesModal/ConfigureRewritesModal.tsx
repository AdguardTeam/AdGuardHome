import { createSignal, createEffect, untrack } from 'solid-js';

import intl from 'panel/common/intl';
import { Dialog } from 'panel/common/ui/Dialog/Dialog';
import { MODAL_TYPE } from 'panel/helpers/constants';

import { ModalWrapper } from 'panel/common/ui/ModalWrapper';
import { closeModal } from 'panel/stores/modals';
import theme from 'panel/lib/theme';
import { Button } from 'panel/common/ui/Button';
import { addRewrite, updateRewrite, rewritesState } from 'panel/stores/rewrites';
import { Input } from 'panel/common/controls/Input';
import {
    validateAnswer,
    validateDomain,
    validateRequiredValue,
    validateRewriteNotExists,
    validateRewriteNotSame,
} from 'panel/helpers/validators';
import { DomainFaqTooltip } from './DomainFaqTooltip';
import { AnswerFaqTooltip } from './AnswerFaqTooltip';

type FormValues = {
    answer: string;
    domain: string;
    enabled: boolean;
};

type ConfigureRewritesModalIdType = 'ADD_REWRITE' | 'EDIT_REWRITE';

type Props = {
    modalId: ConfigureRewritesModalIdType;
    rewriteToEdit?: FormValues;
    onSubmit?: (values: FormValues) => boolean | void | Promise<boolean | void>;
    onClose?: () => void;
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

export const ConfigureRewritesModal = (props: Props) => {
    const [domain, setDomain] = createSignal(untrack(() => props.rewriteToEdit?.domain) ?? '');
    const [answer, setAnswer] = createSignal(untrack(() => props.rewriteToEdit?.answer) ?? '');
    const [domainError, setDomainError] = createSignal<string | undefined>();
    const [answerError, setAnswerError] = createSignal<string | undefined>();

    createEffect(() => {
        setDomain(props.rewriteToEdit?.domain ?? '');
        setAnswer(props.rewriteToEdit?.answer ?? '');
    });

    const closeDialog = () => {
        setDomain('');
        setAnswer('');
        setDomainError(undefined);
        setAnswerError(undefined);
        props.onClose?.();
        closeModal();
    };

    const handleCancel = () => {
        setDomain(props.rewriteToEdit?.domain ?? '');
        setAnswer(props.rewriteToEdit?.answer ?? '');
        setDomainError(undefined);
        setAnswerError(undefined);
        props.onClose?.();
        closeModal();
    };

    const validateDomainField = () => {
        const err = validateRequiredValue(domain()) || validateDomain(domain());
        setDomainError(err || undefined);
        return !err;
    };

    const validateAnswerField = () => {
        const err =
            validateRequiredValue(answer()) ||
            validateAnswer(answer()) ||
            validateRewriteNotSame(domain(), answer()) ||
            validateRewriteNotExists(domain(), rewritesState.list, props.rewriteToEdit?.domain);
        setAnswerError(err || undefined);
        return !err;
    };

    const hasErrors = () => !!domainError() || !!answerError();

    const handleFormSubmit = async (e: Event) => {
        e.preventDefault();

        const domainValid = validateDomainField();
        const answerValid = validateAnswerField();
        if (!domainValid || !answerValid) {
            return;
        }

        const values: FormValues = {
            domain: domain(),
            answer: answer(),
            enabled: props.rewriteToEdit?.enabled ?? false,
        };

        if (props.onSubmit) {
            const shouldClose = await props.onSubmit(values);

            if (shouldClose !== false) {
                closeDialog();
            }

            return;
        }

        switch (props.modalId) {
            case MODAL_TYPE.ADD_REWRITE: {
                addRewrite({ answer: values.answer, domain: values.domain, enabled: true });
                closeDialog();
                break;
            }
            case MODAL_TYPE.EDIT_REWRITE: {
                updateRewrite({
                    target: props.rewriteToEdit,
                    update: {
                        answer: values.answer,
                        domain: values.domain,
                        enabled: values.enabled,
                    },
                });
                closeDialog();
                break;
            }
            default: {
                break;
            }
        }
    };

    return (
        <ModalWrapper id={props.modalId}>
            <Dialog
                visible
                onClose={handleCancel}
                title={getTitle(props.modalId)}
                noOverflowContent
            >
                <form onSubmit={handleFormSubmit}>
                    <div>
                        <div class={theme.form.group}>
                            <div class={theme.form.input}>
                                <Input
                                    type="text"
                                    id="domain"
                                    data-testid="rewrite-domain-input"
                                    label={
                                        <>
                                            {intl.getMessage('rewrite_domain')}
                                            <DomainFaqTooltip />
                                        </>
                                    }
                                    placeholder={intl.getMessage(
                                        'rewrite_domain_input_placeholder',
                                    )}
                                    value={domain()}
                                    onChange={(e) => {
                                        setDomain((e.target as HTMLInputElement).value);
                                        setDomainError(undefined);
                                    }}
                                    onBlur={validateDomainField}
                                    errorMessage={domainError()}
                                    size="large"
                                />
                            </div>

                            <div class={theme.form.input}>
                                <Input
                                    type="text"
                                    id="answer"
                                    data-testid="rewrite-answer-input"
                                    label={
                                        <>
                                            {intl.getMessage('result')}
                                            <AnswerFaqTooltip />
                                        </>
                                    }
                                    placeholder={intl.getMessage(
                                        'rewrites_answer_input_placeholder',
                                    )}
                                    value={answer()}
                                    onChange={(e) => {
                                        setAnswer((e.target as HTMLInputElement).value);
                                        setAnswerError(undefined);
                                    }}
                                    onBlur={validateAnswerField}
                                    errorMessage={answerError()}
                                    size="large"
                                />
                            </div>
                        </div>
                    </div>

                    <div class={theme.dialog.footer}>
                        <Button
                            type="submit"
                            id="save"
                            data-testid="rewrite-save-button"
                            variant="primary"
                            size="small"
                            disabled={
                                rewritesState.processingAdd ||
                                rewritesState.processingUpdate ||
                                rewritesState.processing ||
                                hasErrors()
                            }
                            class={theme.dialog.button}
                        >
                            {getButtonText(props.modalId)}
                        </Button>

                        <Button
                            type="button"
                            id="cancel"
                            data-testid="rewrite-cancel-button"
                            variant="secondary"
                            size="small"
                            onClick={handleCancel}
                            class={theme.dialog.button}
                        >
                            {intl.getMessage('cancel')}
                        </Button>
                    </div>
                </form>
            </Dialog>
        </ModalWrapper>
    );
};
