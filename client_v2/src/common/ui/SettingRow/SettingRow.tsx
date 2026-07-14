import { type JSX, Show, createMemo } from 'solid-js';
import cn from 'clsx';

import { Switch } from 'panel/common/controls/Switch';
import { Icon } from 'panel/common/ui/Icon';

import s from './SettingRow.module.pcss';
import theme from 'panel/lib/theme';

type SettingRowVariant = 'switch' | 'link' | 'switch-link';

type Props = {
    id: string;
    title: string;
    titleClass?: string;
    description?: string | JSX.Element;
    descriptionClass?: string;
    value?: string;
    variant: SettingRowVariant;
    checked?: boolean;
    disabled?: boolean;
    onChange?: (checked: boolean) => void;
    onClick?: () => void;
    class?: string;
    children?: JSX.Element;
    divider?: boolean;
    align?: 'top' | 'center';
    largeTitle?: boolean;
    inputClass?: string;
};

export const SettingRow = (props: Props) => {
    let inputRef: HTMLInputElement | undefined;

    const isSwitch = createMemo(() => props.variant === 'switch');
    const isLink = createMemo(() => props.variant === 'link');
    const isSwitchLink = createMemo(() => props.variant === 'switch-link');

    const handleRowClick = (e?: MouseEvent) => {
        if (props.disabled) {
            return;
        }
        // Skip programmatic click if the user already clicked the switch/label
        // — the native label behaviour already toggles it.
        if (e) {
            const target = e.target as HTMLElement;
            if (target.tagName === 'INPUT' || target.closest('label')) {
                return;
            }
        }
        if (isSwitch()) {
            inputRef?.click();
        } else if (isLink() || isSwitchLink()) {
            props.onClick?.();
        }
    };

    const handleSwitchChange = (e: Event) => {
        e.stopPropagation();
        if (props.disabled) {
            return;
        }
        const target = e.target as HTMLInputElement;
        props.onChange?.(target.checked);
    };

    const handleLinkClick = (e: Event) => {
        e.stopPropagation();
        if (props.disabled) {
            return;
        }
        props.onClick?.();
    };

    const handleInputClick = (e: MouseEvent) => {
        const target = e.target as HTMLElement;
        const isSwitchClick = target.tagName === 'INPUT' || !!target.closest('label');

        // Native label click already toggled the switch — don't double-fire.
        if (isSwitchClick) {
            return;
        }

        if ((isSwitch() || isSwitchLink()) && !props.disabled) {
            e.stopPropagation();
            inputRef?.click();
        }
    };

    const isSwitchVariant = () => isSwitch() || isSwitchLink();

    const isLinkVariant = () => isLink();

    return (
        <div
            class={cn(s.switch, props.class, {
                [s.switchDisabled]: props.disabled,
            })}
            role="button"
            tabIndex={props.disabled ? -1 : 0}
            onClick={handleRowClick}
            onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    handleRowClick();
                }
            }}
        >
            <div
                class={cn(s.row, {
                    [s.rowTop]: props.align === 'top',
                    [s.rowCenter]: props.align === 'center',
                })}
            >
                <div class={s.text}>
                    <div
                        class={cn(s.title, props.titleClass, {
                            [s.titleDisabled]: props.disabled,
                        })}
                    >
                        {props.title}
                    </div>
                    <Show when={props.description}>
                        <div
                            class={cn(s.desc, props.descriptionClass, theme.text.t3, {
                                [s.descDisabled]: props.disabled,
                            })}
                        >
                            {props.description}
                        </div>
                    </Show>
                    <Show when={props.value}>
                        <div
                            class={cn(s.value, theme.text.t3, {
                                [s.valueDisabled]: props.disabled,
                            })}
                        >
                            {props.value}
                        </div>
                    </Show>
                </div>
                <Show when={isSwitchLink() && props.divider}>
                    <div class={s.divider} />
                </Show>
                <div
                    class={cn(s.input, props.inputClass, props.largeTitle && s.largeTitle)}
                    onClick={handleInputClick}
                >
                    <Show when={isSwitchVariant()}>
                        <Switch
                            id={props.id}
                            checked={!!props.checked}
                            disabled={!!props.disabled}
                            onChange={handleSwitchChange}
                            ref={(el: HTMLInputElement) => {
                                inputRef = el;
                            }}
                        />
                    </Show>
                    <Show when={isLinkVariant()}>
                        <button
                            type="button"
                            class={s.link}
                            disabled={!!props.disabled}
                            onClick={handleLinkClick}
                        >
                            <Icon icon="arrow" class={s.arrow} />
                        </button>
                    </Show>
                </div>
            </div>
            <Show when={props.children}>
                <div class={s.content}>{props.children}</div>
            </Show>
        </div>
    );
};
