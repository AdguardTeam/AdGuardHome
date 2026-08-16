import { type JSX, Show } from 'solid-js';
import cn from 'clsx';

import { Switch } from 'panel/common/controls/Switch';
import theme from 'panel/lib/theme';

import s from './styles.module.pcss';

type Props = {
    title: string;
    description?: string;
    id: string;
    class?: string;
    checked: boolean;
    onChange: (e: Event) => void;
    disabled?: boolean;
    children?: JSX.Element;
};

export const SwitchGroup = (props: Props) => {
    let inputRef: HTMLInputElement | undefined;

    const handleRowClick = () => {
        if (props.disabled || !inputRef) {
            return;
        }

        inputRef?.click();
    };

    return (
        <div class={cn(s.switch, { [s.switchDisabled]: props.disabled }, props.class)}>
            <div class={s.row} onClick={handleRowClick}>
                <div class={s.text}>
                    <div
                        class={cn(
                            theme.text.t2,
                            theme.text.semibold,
                            props.disabled ? s.titleDisabled : s.title,
                        )}
                    >
                        {props.title}
                    </div>
                    <Show when={props.description}>
                        <div class={cn(theme.text.t3, s.desc)}>{props.description}</div>
                    </Show>
                </div>
                <div class={s.input} onClick={(e: MouseEvent) => e.stopPropagation()}>
                    <Switch
                        id={props.id}
                        checked={props.checked}
                        onChange={props.onChange}
                        disabled={props.disabled}
                        ref={(el: HTMLInputElement) => {
                            inputRef = el;
                        }}
                    />
                </div>
            </div>

            <Show when={props.children}>
                <div class={s.content}>{props.children}</div>
            </Show>
        </div>
    );
};
