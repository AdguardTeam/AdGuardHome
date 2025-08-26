import React from 'react';
import { Controller, type Control, type UseFormSetValue } from 'react-hook-form';
import cn from 'clsx';

import { Textarea } from 'panel/common/controls/Textarea';
import intl from 'panel/common/intl';
import { trimLinesAndRemoveEmpty } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import { SwitchGroup } from 'panel/common/ui/SettingsGroup';

import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import s from './styles.module.pcss';

type Props = {
    control: Control<any>;
    processing: boolean;
    ignoreEnabled: boolean;
    setValue: UseFormSetValue<any>;
    switchId: string;
    textareaId: string;
};

export const IgnoredDomains = ({ control, processing, ignoreEnabled, setValue, switchId, textareaId }: Props) => {
    return (
        <SwitchGroup
            id={switchId}
            title={intl.getMessage('ignore_domains_title')}
            description={intl.getMessage('ignore_domains_desc_query')}
            checked={ignoreEnabled}
            onChange={(e) => setValue('ignore_enabled', e.target.checked)}
            disabled={processing}>
            <Controller
                name="ignored"
                control={control}
                render={({ field, fieldState }) => (
                    <Textarea
                        {...field}
                        id={textareaId}
                        label={
                            <>
                                {intl.getMessage('settings_domain_names')}

                                <FaqTooltip
                                    text={
                                        <>
                                            <div className={s.dropdownTitle}>
                                                {intl.getMessage('settings_tooltip_domain_names')}
                                            </div>
                                            <div>
                                                <strong>{intl.getMessage('settings_tooltip_examples')}</strong>
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
                                    className={cn(s.link, theme.link.link, theme.link.noDecoration)}>
                                    {intl.getMessage('settings_rule_syntax')}
                                </a>
                            </>
                        }
                        placeholder={`example.com\n*.example.com\n||example.com^`}
                        size="large"
                        disabled={processing || !ignoreEnabled}
                        errorMessage={fieldState.error?.message}
                        onBlur={(e) => {
                            const trimmed = trimLinesAndRemoveEmpty(e.target.value);
                            setValue('ignored', trimmed);
                        }}
                    />
                )}
            />
        </SwitchGroup>
    );
};
