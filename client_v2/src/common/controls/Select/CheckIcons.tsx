import { useSelectItemContext, useComboboxItemContext } from '@ark-ui/solid';
import { Icon } from 'panel/common/ui/Icon';

// Check/dot icon for ArkSelect.Item (reads context reactively in JSX).
export const SelectCheckIcon = () => {
    const itemCtx = useSelectItemContext();
    return <Icon icon={itemCtx().selected ? 'check' : 'dot'} />;
};

// Check/dot icon for ArkCombobox.Item.
export const ComboboxCheckIcon = () => {
    const itemCtx = useComboboxItemContext();
    return <Icon icon={itemCtx().selected ? 'check' : 'dot'} />;
};
