import { type JSX } from 'solid-js';
import { IOption } from 'panel/lib/helpers/utils';

export const SEARCH_ENABLE_LIMIT = 10;

export type ISelectSize = 'auto' | 'small' | 'medium' | 'big' | 'big-limit' | 'responsive';
export type ISelectHeight = 'small' | 'medium' | 'big' | 'big-mobile';
export type ISelectMenuSize = 'small' | 'medium' | 'big' | 'large';
export type ISelectValue<T, Multi extends boolean> = Multi extends true ? IOption<T>[] : IOption<T>;

export interface SelectProps<
    T,
    Multi extends boolean = false,
    ExtendOption extends Record<any, any> = object,
> {
    size?: ISelectSize;
    height?: ISelectHeight;
    menuSize?: ISelectMenuSize;
    menuPosition?: 'right';
    mobile?: boolean;
    isDisabled?: boolean;
    isMulti?: Multi;
    menuIsOpen?: boolean;
    placeholder?: JSX.Element;
    class?: string;
    autoFocus?: boolean;
    low?: boolean;
    options: (IOption<T> & ExtendOption)[];
    onChange: (
        value: Multi extends true ? (IOption<T> & ExtendOption)[] : IOption<T> & ExtendOption,
    ) => void;
    value?: (IOption<T> & ExtendOption) | (IOption<T> & ExtendOption)[];
    isSearchable?: boolean;
    id?: string;
    inputId?: string;
    borderless?: boolean;
    adaptiveHeight?: boolean;
    lazyList?: boolean;
    closeMenuOnSelect?: boolean;
    menuPlacement?: 'top' | 'bottom' | 'auto';
    isClearable?: boolean;
    onMenuScrollToBottom?: () => void;
    isLoading?: boolean;
    isDropdownSelect?: boolean;
    onMenuOpen?: () => void;
    onMenuClose?: () => void;
    showIcons?: boolean;
    showOptionIcon?: boolean;
    optionTestIdPrefix?: string;
    /** Optional footer rendered after the scrollable item list inside the dropdown. */
    menuFooter?: JSX.Element;
}
