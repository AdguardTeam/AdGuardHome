import { type JSX, Show } from 'solid-js';
import { Combobox as ArkCombobox, useComboboxContext } from '@ark-ui/solid';
import { SelectMultiValue } from './SelectMultiValue';
import { optionToValue } from './helpers';

interface ComboboxMultiValueDisplayProps {
    placeholder?: string;
    inputId?: string;
    onInputRef?: (el: HTMLInputElement) => void;
}

// Multi-value pills + inline search input for searchable Combobox.
export const ComboboxMultiValueDisplay = (props: ComboboxMultiValueDisplayProps): JSX.Element => {
    const comboCtx = useComboboxContext();

    return (
        <>
            <Show when={comboCtx().hasSelectedItems}>
                <SelectMultiValue
                    items={comboCtx().selectedItems as any[]}
                    onRemove={(item) => comboCtx().clearValue(optionToValue(item.value))}
                />
            </Show>
            <ArkCombobox.Input
                id={props.inputId}
                placeholder={comboCtx().hasSelectedItems ? '' : (props.placeholder ?? '')}
                ref={props.onInputRef}
            />
        </>
    );
};
