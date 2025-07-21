import React from 'react';
import RSelect, { GroupBase, MenuListProps, SelectComponentsConfig } from 'react-select';
import cn from 'clsx';
import { IOption } from 'panel/lib/helpers/utils';

import { CustomClearIndicator } from './CustomClearIndicator';
import { CustomDropdownIndicator } from './CustomDropdownIndicator';
import { CustomLoadingIndicator } from './CustomLoadingIndicator';
import { CustomLoadingMessage } from './CustomLoadingMessage';
import { CustomOption } from './CustomOption';

import './Select.pcss';
import s from './MenuList.module.pcss';

const SEARCH_ENABLE_LIMIT = 10;

export type ISelectSize = 'auto' | 'small' | 'medium' | 'big' | 'big-limit' | 'responsive';
export type ISelectHeight = 'small' | 'medium' | 'big' | 'big-mobile';
export type ISelectMenuSize = 'small' | 'medium' | 'big' | 'large';
export type ISelectValue<T, Multi extends boolean> = Multi extends true ? IOption<T>[] : IOption<T>;

const CustomMenuList = <
    T,
    Multi extends boolean,
    Group extends GroupBase<IOption<T> & ExtendOption>,
    ExtendOption extends Record<string, any>,
>(
    props: MenuListProps<IOption<T> & ExtendOption, Multi, Group>,
): React.ReactElement => {
    const { children } = props;
    const scrollContainerRef = React.useRef<HTMLDivElement>(null);

    const childArray = React.Children.toArray(children);
    if (childArray.length > 0 && React.isValidElement(childArray[0]) && childArray.length === 1) {
        return <>{children}</>;
    }

    return (
        <div className={s.menuList} ref={scrollContainerRef}>
            {children}
        </div>
    );
};

interface SelectProps<
    T,
    Multi extends boolean,
    Group extends GroupBase<IOption<T> & ExtendOption>,
    ExtendOption extends Record<string, any>,
> {
    size?: ISelectSize;
    height?: ISelectHeight;
    menuSize?: ISelectMenuSize;
    menuPosition?: 'right';
    checkmark?: boolean;
    mobile?: boolean; // true: mobile-always, false: desktop-always, undefined: responsive
    isDisabled?: boolean;
    isMulti?: Multi;
    menuIsOpen?: boolean;
    placeholder?: React.ReactNode;
    className?: string;
    autoFocus?: boolean;
    low?: boolean;
    options: ((IOption<T> & ExtendOption) | Group)[];
    onChange: (value: Multi extends true ? (IOption<T> & ExtendOption)[] : IOption<T> & ExtendOption) => void;
    value?: (IOption<T> & ExtendOption) | (IOption<T> & ExtendOption)[];
    formatGroupLabel?: (group: Group) => string;
    components?: SelectComponentsConfig<IOption<T> & ExtendOption, Multi, Group>;
    isSearchable?: boolean;
    id?: string;
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
}

export const Select = <
    T,
    Multi extends boolean = false,
    ExtendOption extends Record<any, any> = {},
    Group extends GroupBase<IOption<T> & ExtendOption> = GroupBase<IOption<T> & ExtendOption>,
>({
    autoFocus,
    isDisabled,
    isMulti,
    options,
    components,
    value,
    onChange,
    formatGroupLabel,
    placeholder,
    className,
    mobile,
    menuIsOpen,
    menuPosition,
    checkmark = true,
    size,
    height,
    menuSize,
    low,
    isSearchable,
    id,
    borderless,
    adaptiveHeight,
    lazyList,
    closeMenuOnSelect,
    menuPlacement,
    isClearable,
    onMenuScrollToBottom,
    isLoading,
    isDropdownSelect,
    onMenuOpen,
    onMenuClose,
}: SelectProps<T, Multi, Group, ExtendOption>) => {
    const selectClass = cn(
        { 'desktop-select-always': mobile === false },
        { 'mobile-select-always': mobile },
        'react-select',
        { 'react-select-big': size === 'big' },
        { 'react-select-medium': size === 'medium' },
        { 'react-select-small': size === 'small' },
        { 'react-select-responsive': size === 'responsive' },
        { 'react-select--big-limit': size === 'big-limit' },
        { 'react-select--menu-small': menuSize === 'small' },
        { 'react-select--menu-medium': menuSize === 'medium' },
        { 'react-select--menu-big': menuSize === 'big' },
        { 'react-select--menu-right': menuPosition === 'right' },
        { 'react-select--borderless': borderless },
        { 'react-select--adaptive-height': adaptiveHeight },
        { 'react-select--height-small': height === 'small' },
        { 'react-select--height-medium': height === 'medium' },
        { 'react-select--height-big': height === 'big' },
        { 'react-select--height-big-mobile': height === 'big-mobile' },
        { 'react-select--dropdown': isDropdownSelect },
        { low },
        className,
    );

    const customComponents = {
        ...(checkmark ? { Option: CustomOption } : {}),
        ...(lazyList ? { MenuList: CustomMenuList } : {}),
        ClearIndicator: CustomClearIndicator,
        DropdownIndicator: CustomDropdownIndicator,
        LoadingIndicator: CustomLoadingIndicator,
        LoadingMessage: CustomLoadingMessage,
        ...components,
    };

    return (
        <RSelect<IOption<T> & ExtendOption, Multi, Group>
            hideSelectedOptions={false}
            menuIsOpen={menuIsOpen}
            placeholder={placeholder}
            className={selectClass}
            classNamePrefix="select"
            options={options}
            autoFocus={autoFocus}
            isDisabled={isDisabled}
            isMulti={isMulti}
            onChange={(value: Multi extends true ? (IOption<T> & ExtendOption)[] : IOption<T> & ExtendOption) =>
                onChange(value)
            }
            value={value}
            formatGroupLabel={formatGroupLabel}
            components={customComponents}
            isSearchable={isSearchable || options.length > SEARCH_ENABLE_LIMIT}
            id={id}
            closeMenuOnSelect={closeMenuOnSelect}
            menuPlacement={menuPlacement}
            isClearable={isClearable}
            onMenuScrollToBottom={onMenuScrollToBottom}
            controlShouldRenderValue={!isDropdownSelect}
            isLoading={isLoading}
            onMenuOpen={onMenuOpen}
            onMenuClose={onMenuClose}
        />
    );
};
