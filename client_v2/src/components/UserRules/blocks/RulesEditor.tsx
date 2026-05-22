import React, { useRef } from 'react';
import { Controller, UseFormHandleSubmit, Control } from 'react-hook-form';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import intl from 'panel/common/intl';
import { Textarea } from 'panel/common/controls/Textarea';
import { Button } from 'panel/common/ui/Button';
import { getTextareaCommentsHighlight, syncScroll } from 'panel/helpers/highlightTextareaComments';

import s from '../UserRules.module.pcss';

type UserRulesFormValues = {
    userRules: string;
};

type Props = {
    control: Control<UserRulesFormValues>;
    handleSubmit: UseFormHandleSubmit<UserRulesFormValues>;
    onSubmit: (data: UserRulesFormValues) => void;
    processingRules: boolean;
};

export const RulesEditor = ({ control, handleSubmit, onSubmit, processingRules }: Props) => {
    const highlightRef = useRef<HTMLDivElement | null>(null);

    return (
        <div className={s.section}>
            <form onSubmit={handleSubmit(onSubmit)}>
                <p className={cn(s.description, theme.text.t2)}>{intl.getMessage('user_rules_desc')}</p>

                <div className={s.textEditWrapper}>
                    <div className={s.textEditContainer}>
                        <Controller
                            name="userRules"
                            control={control}
                            render={({ field }) => (
                                <>
                                    {getTextareaCommentsHighlight(highlightRef, field.value || '', ['!', '#'])}

                                    <Textarea
                                        {...field}
                                        data-testid="user-rules-editor-textarea"
                                        placeholder={intl.getMessage('user_rules_placeholder')}
                                        rows={12}
                                        size="large"
                                        onScroll={(e) => syncScroll(e, highlightRef)}
                                    />
                                </>
                            )}
                        />
                    </div>
                </div>

                <div className={s.editorActions}>
                    <Button
                        type="submit"
                        variant="primary"
                        size="small"
                        disabled={processingRules}
                        className={s.editorSubmitButton}
                        data-testid="user-rules-editor-save"
                    >
                        {intl.getMessage('save')}
                    </Button>
                </div>
            </form>
        </div>
    );
};
