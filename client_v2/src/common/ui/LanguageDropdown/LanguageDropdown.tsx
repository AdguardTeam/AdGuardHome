import { createSignal, createMemo, For, untrack } from 'solid-js';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import intl from 'panel/common/intl';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Icon } from 'panel/common/ui/Icon';
import s from './LanguageDropdown.module.pcss';

type LanguageDropdownProps = {
    value: string;
    languages: Record<string, string>;
    languageNames?: Record<string, string>;
    onChange: (lang: string) => void | Promise<void>;
    position?: 'bottomRight' | 'bottomLeft' | 'topRight' | 'topLeft';
    class?: string;
    sort?: boolean;
};

const getLanguageShortLabel = (lang: string) => {
    const code = (lang || '').toLowerCase();
    const parts = code.split('-');
    if (parts.length === 1) {
        return parts[0].slice(0, 2).toUpperCase();
    }

    const prefix = parts[0].slice(0, 2).toUpperCase();
    const suffix = parts
        .slice(1)
        .map((p) => p.toUpperCase())
        .join('-');

    // Skip redundant suffix, e.g. pt-pt → PT (not PT (PT))
    if (prefix === suffix) {
        return prefix;
    }

    return `${prefix} (${suffix})`;
};

export const LanguageDropdown = (props: LanguageDropdownProps) => {
    const [open, setOpen] = createSignal(false);

    const sortedKeys = createMemo(() => {
        const keys = Object.keys(props.languages);
        if (!props.sort) {
            return keys;
        }
        const languages = props.languages;
        return keys.sort((a, b) => (languages[a] || '').localeCompare(languages[b] || ''));
    });

    const currentLabel = () => getLanguageShortLabel(props.value);

    return (
        <Dropdown
            open={open()}
            onOpenChange={setOpen}
            menu={
                <div class={cn(theme.dropdown.menu, theme.dropdown.menu_lang)}>
                    <For each={sortedKeys()}>
                        {(lang) => (
                            <button
                                type="button"
                                class={cn(theme.dropdown.item, {
                                    [theme.dropdown.item_active]: props.value === lang,
                                })}
                                onClick={() => {
                                    // Fire-and-forget with finally-guaranteed close.
                                    // If the onChange throws or hangs, the dropdown
                                    // still closes — no stuck-open state.
                                    Promise.resolve(untrack(() => props.onChange)(lang)).finally(
                                        () => setOpen(false),
                                    );
                                }}
                            >
                                {props.languageNames?.[lang] || getLanguageShortLabel(lang)}
                            </button>
                        )}
                    </For>
                </div>
            }
            class={props.class}
            overlayClass={s.langOverlay}
            position={props.position ?? 'bottomRight'}
        >
            <button
                type="button"
                class={cn(s.langButton, props.class)}
                aria-label={intl.getMessage('language')}
            >
                <Icon icon="lang" />
                <span class={s.langLabel}>{currentLabel()}</span>
            </button>
        </Dropdown>
    );
};
