import React from 'react';
import { Controller, type Control, type UseFormSetValue } from 'react-hook-form';
import cn from 'clsx';

import { Textarea } from 'panel/common/controls/Textarea';
import { SwitchGroup } from '../SettingsGroup';
import intl from 'panel/common/intl';

import { trimLinesAndRemoveEmpty } from 'panel/helpers/helpers';

import s from './styles.module.pcss';
import { Dropdown } from 'panel/common/ui/Dropdown';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';

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
                name={'ignored'}
                control={control}
                render={({ field, fieldState }) => (
                    <Textarea
                        {...field}
                        id={textareaId}
                        label={
                            <div className={s.label}>
                                {intl.getMessage('settings_domain_names')}
                                <Dropdown
                                    trigger="hover"
                                    menu={
                                        <div className={cn(theme.dropdown.menu, s.dropdownMenu)}>
                                            <div className={s.dropdownTitle}>
                                                {intl.getMessage('settings_tooltip_domain_names')}
                                            </div>
                                            <div className={s.dropdownText}>
                                                <strong>{intl.getMessage('settings_tooltip_examples')}</strong>
                                                <div>example.com</div>
                                                <div>*.example.com</div>
                                                <div>||example.com^</div>
                                            </div>
                                        </div>
                                    }
                                    className={s.dropdown}
                                    position="bottomLeft"
                                    noIcon>
                                    <div className={s.dropdownTrigger}>
                                        <Icon icon="faq" className={s.icon} />
                                    </div>
                                </Dropdown>
                                <a
                                    href="https://link.adtidy.org/forward.html?action=dns_kb_filtering_syntax&from=ui&app=home"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className={cn(s.link, theme.link.link, theme.link.noDecoration)}>
                                    {intl.getMessage('settings_rule_syntax')}
                                </a>
                            </div>
                        }
                        placeholder={'example.com\n*.example.com\n||example.com^'}
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
