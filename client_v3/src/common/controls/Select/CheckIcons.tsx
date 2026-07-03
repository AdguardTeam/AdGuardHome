import { type JSX } from 'solid-js';
import { useSelectItemContext, useComboboxItemContext } from '@ark-ui/solid';
import { Icon } from 'panel/common/ui/Icon';

const renderCheckIcon = (selected: boolean): JSX.Element => (
    <Icon icon={selected ? 'check' : 'dot'} />
);

// Check/dot icon for ArkSelect.Item (reads context reactively in JSX).
export const SelectCheckIcon = () => {
    const itemCtx = useSelectItemContext();
    return renderCheckIcon(itemCtx().selected);
};

// Check/dot icon for ArkCombobox.Item.
export const ComboboxCheckIcon = () => {
    const itemCtx = useComboboxItemContext();
    return renderCheckIcon(itemCtx().selected);
};
