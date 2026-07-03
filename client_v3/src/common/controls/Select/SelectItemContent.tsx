import { type JSX, Show } from 'solid-js';
import { Select as ArkSelect } from '@ark-ui/solid';
import { Icon } from 'panel/common/ui/Icon';
import { SelectCheckIcon } from './CheckIcons';
import s from './MenuList.module.pcss';

interface ItemContentProps {
    option: any;
    showOptionIcon?: boolean;
    showIcons?: boolean;
}

// Option row for the non-searchable Select branch.
export const SelectItemContent = (props: ItemContentProps): JSX.Element => (
    <>
        <Show when={props.showOptionIcon !== false}>
            <ArkSelect.ItemIndicator>
                <SelectCheckIcon />
            </ArkSelect.ItemIndicator>
        </Show>
        <Show when={props.showIcons && props.option.icon}>
            <div class={s.selectIconContainer}>
                <Icon icon={props.option.icon} />
            </div>
        </Show>
        <ArkSelect.ItemText>{props.option.label}</ArkSelect.ItemText>
    </>
);
