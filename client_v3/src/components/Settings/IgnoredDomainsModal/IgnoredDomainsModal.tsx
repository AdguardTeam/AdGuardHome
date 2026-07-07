import { createSignal, createEffect, on } from 'solid-js';
import cn from 'clsx';

import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Textarea } from 'panel/common/controls/Textarea';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { trimLinesAndRemoveEmpty } from 'panel/helpers/helpers';

import s from './IgnoredDomainsModal.module.pcss';

type Props = {
    open: boolean;
    title: string;
    ignored: string[];
    processing: boolean;
    onClose: () => void;
    onSave: (ignored: string[]) => void;
};

export const IgnoredDomainsModal = (props: Props) => {
    const [value, setValue] = createSignal('');

    createEffect(
        on(
            () => props.open,
            (open) => {
                if (!open) return;
                setValue(props.ignored.join('\n'));
            },
        ),
    );

    const handleSave = () => {
        const trimmed = trimLinesAndRemoveEmpty(value());
        const ignoredArray = trimmed.split('\n').filter(Boolean);
        props.onSave(ignoredArray);
    };

    return (
        <ConfigDialog
            open={props.open}
            title={props.title}
            onClose={props.onClose}
            onSubmit={handleSave}
            processing={props.processing}
        >
            <Textarea
                value={value()}
                onChange={(e: Event) => setValue((e.target as HTMLTextAreaElement).value)}
                onBlur={(e: Event) => {
                    const trimmed = trimLinesAndRemoveEmpty(
                        (e.target as HTMLTextAreaElement).value,
                    );
                    setValue(trimmed);
                }}
                label={
                    <>
                        {intl.getMessage('settings_domain_names')}
                        <FaqTooltip
                            text={
                                <>
                                    <div class={s.dropdownTitle}>
                                        {intl.getMessage('settings_tooltip_domain_names')}
                                    </div>
                                    <div>
                                        <strong>
                                            {intl.getMessage('settings_tooltip_examples')}
                                        </strong>
                                        <div>example.com</div>
                                        <div>*.example.com</div>
                                        <div>||example.com^</div>
                                    </div>
                                </>
                            }
                        />
                        <a
                            href="https://link.adtidy.org/forward.html?action=dns_kb_filtering_syntax&from=ui&app=home"
                            target="_blank"
                            rel="noopener noreferrer"
                            class={cn(s.link, theme.link.link, theme.link.noDecoration)}
                        >
                            {intl.getMessage('settings_rule_syntax')}
                        </a>
                    </>
                }
                placeholder={`example.com\n*.example.com\n||example.com^`}
                size="large"
                disabled={props.processing}
            />
        </ConfigDialog>
    );
};
