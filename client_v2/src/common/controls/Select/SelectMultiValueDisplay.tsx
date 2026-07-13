import { type JSX, Show } from 'solid-js';
import { useSelectContext } from '@ark-ui/solid';
import { SelectMultiValue } from './SelectMultiValue';
import { optionToValue } from './helpers';

interface DisplayProps {
    placeholder?: string;
}

// Multi-value pills for non-searchable Select (reads from Select context).
export const SelectMultiValueDisplay = (props: DisplayProps): JSX.Element => {
    const selectCtx = useSelectContext();

    return (
        <Show
            when={selectCtx().hasSelectedItems}
            fallback={<span class="solid-select-placeholder">{props.placeholder ?? ''}</span>}
        >
            <SelectMultiValue
                items={selectCtx().selectedItems as any[]}
                onRemove={(item) => selectCtx().clearValue(optionToValue(item.value))}
            />
        </Show>
    );
};
