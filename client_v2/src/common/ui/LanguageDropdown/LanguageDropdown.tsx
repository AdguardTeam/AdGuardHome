import React, { useMemo, useState } from 'react';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Icon } from 'panel/common/ui/Icon';
import s from './LanguageDropdown.module.pcss'

type LanguageDropdownProps = {
    value: string;
    languages: Record<string, string>;
    onChange: (lang: string) => void | Promise<void>;
    position?: 'bottomRight' | 'bottomLeft' | 'topRight' | 'topLeft';
    className?: string;
    sort?: boolean;
};

export const LanguageDropdown = ({
      value,
      languages,
      onChange,
      position = 'bottomRight',
      className,
      sort = true,
  }: LanguageDropdownProps) => {
    const [open, setOpen] = useState(false);

    const sortedKeys = useMemo(() => {
        const keys = Object.keys(languages);
        if (!sort) {
            return keys;
        }
        return keys.sort((a, b) => (languages[a] || '').localeCompare(languages[b] || ''));
    }, [languages, sort]);

    const currentLabel = languages[value] || '';

    return (
        <Dropdown
            trigger="click"
            open={open}
            onOpenChange={setOpen}
            menu={
                <div className={cn(theme.dropdown.menu, theme.dropdown.menu_lang)}>
                    {sortedKeys.map((lang) => (
                        <button
                            type="button"
                            key={lang}
                            className={cn(theme.dropdown.item, {
                                [theme.dropdown.item_active]: value === lang,
                            })}
                            onClick={async () => {
                                await onChange(lang);
                                setOpen(false);
                            }}>
                            {languages[lang]}
                        </button>
                    ))}
                </div>
            }
            className={className}
            overlayClassName={s.langOverlay}
            position={position}>
            <div className={cn(className)}>
                <div>
                    <Icon icon="lang" />
                    <span>{currentLabel}</span>
                </div>
            </div>
        </Dropdown>
    );
};
