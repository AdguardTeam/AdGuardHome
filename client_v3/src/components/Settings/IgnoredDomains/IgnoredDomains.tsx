import cn from 'clsx';

import { Textarea } from 'panel/common/controls/Textarea';
import intl from 'panel/common/intl';
import { trimLinesAndRemoveEmpty } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import { SwitchGroup } from 'panel/common/ui/SettingsGroup';

import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import s from './styles.module.pcss';

type Props = {
    ignoredValue: string;
    onIgnoredChange: (value: string) => void;
    processing: boolean;
    ignoreEnabled: boolean;
    onIgnoreEnabledChange: (checked: boolean) => void;
    switchId: string;
    textareaId: string;
    description: string;
    error?: string;
};

export const IgnoredDomains = (props: Props) => {
    return (
        <SwitchGroup
            id={props.switchId}
            title={intl.getMessage('ignore_domains_title')}
            description={props.description}
            checked={props.ignoreEnabled}
            onChange={(e: Event) =>
                props.onIgnoreEnabledChange((e.target as HTMLInputElement).checked)
            }
            disabled={props.processing}
        >
            <Textarea
                id={props.textareaId}
                value={props.ignoredValue}
                onChange={(e: Event) => {
                    const { value } = e.target as HTMLTextAreaElement;
                    props.onIgnoredChange(value);
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
                disabled={props.processing || !props.ignoreEnabled}
                errorMessage={props.error}
                onBlur={(e: Event) => {
                    const trimmed = trimLinesAndRemoveEmpty(
                        (e.target as HTMLTextAreaElement).value,
                    );
                    props.onIgnoredChange(trimmed);
                }}
            />
        </SwitchGroup>
    );
};
