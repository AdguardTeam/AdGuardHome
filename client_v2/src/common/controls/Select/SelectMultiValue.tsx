import { For } from 'solid-js';
import { Icon } from 'panel/common/ui/Icon';
import intl from 'panel/common/intl';

import s from './SelectMultiValue.module.pcss';

interface SelectMultiValueProps {
    items: {
        value: string | null;
        label: string;
        [key: string]: any;
    }[];
    onRemove: (item: any) => void;
}

export const SelectMultiValue = (props: SelectMultiValueProps) => (
    <For each={props.items}>
        {(item) => (
            <div class={s.pill}>
                <span class={s.label}>{item.label}</span>
                <button
                    type="button"
                    class={s.removeBtn}
                    onClick={(e) => {
                        e.stopPropagation();
                        props.onRemove(item);
                    }}
                    aria-label={intl.getMessage('remove_tag', {
                        value: item.label,
                    })}
                >
                    <Icon icon="cross" color="gray" />
                </button>
            </div>
        )}
    </For>
);
