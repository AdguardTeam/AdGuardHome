import { type JSX, Show } from 'solid-js';
import { Combobox as ArkCombobox } from '@ark-ui/solid';
import { Icon } from 'panel/common/ui/Icon';
import { ComboboxCheckIcon } from './CheckIcons';
import s from './MenuList.module.pcss';

interface ItemContentProps {
    option: any;
    showOptionIcon?: boolean;
    showIcons?: boolean;
}

// Option row for the searchable Combobox branch (mirrors SelectItemContent).
export const ComboboxItemContent = (props: ItemContentProps): JSX.Element => (
    <>
        <Show when={props.showOptionIcon !== false}>
            <ArkCombobox.ItemIndicator>
                <ComboboxCheckIcon />
            </ArkCombobox.ItemIndicator>
        </Show>
        <Show when={props.showIcons && props.option.icon}>
            <div class={s.selectIconContainer}>
                <Icon icon={props.option.icon} />
            </div>
        </Show>
        <ArkCombobox.ItemText>{props.option.label}</ArkCombobox.ItemText>
    </>
);
