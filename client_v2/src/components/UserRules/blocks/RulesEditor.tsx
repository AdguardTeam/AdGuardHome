import cn from 'clsx';

import theme from 'panel/lib/theme';
import intl from 'panel/common/intl';
import { Textarea } from 'panel/common/controls/Textarea';
import { Button } from 'panel/common/ui/Button';
import { COMMENT_LINE_TOKENS } from 'panel/helpers/constants';

import s from '../UserRules.module.pcss';

type Props = {
    value: string;
    onChange: (value: string) => void;
    handleSubmit: () => void;
    processingRules: boolean;
};

export const RulesEditor = (props: Props) => {
    const handleSubmit = (e: Event) => {
        e.preventDefault();
        props.handleSubmit();
    };

    return (
        <div class={s.section}>
            <form onSubmit={handleSubmit}>
                <p class={cn(s.description, theme.text.t2)}>{intl.getMessage('user_rules_desc')}</p>

                <div class={s.textEditWrapper}>
                    <div class={s.textEditContainer}>
                        <Textarea
                            data-testid="user-rules-editor-textarea"
                            placeholder={`# ${intl.getMessage('user_rules_placeholder')}\n\n@@||example.org`}
                            rows={12}
                            size="large"
                            class={s.editorTextarea}
                            value={props.value}
                            onChange={(e: Event) =>
                                props.onChange((e.target as HTMLTextAreaElement).value)
                            }
                            highlightComments
                            commentPrefixes={COMMENT_LINE_TOKENS}
                        />
                    </div>
                </div>

                <div class={s.editorActions}>
                    <Button
                        type="submit"
                        variant="primary"
                        size="small"
                        disabled={props.processingRules}
                        class={s.editorSubmitButton}
                        data-testid="user-rules-editor-save"
                    >
                        {intl.getMessage('save')}
                    </Button>
                </div>
            </form>
        </div>
    );
};
