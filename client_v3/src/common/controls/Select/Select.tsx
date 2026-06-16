import { type JSX, For, Show, createMemo } from 'solid-js';
import cn from 'clsx';
import * as SelectPrimitive from '@thisbeyond/solid-select';
import { IOption } from 'panel/lib/helpers/utils';

import './Select.pcss';
import s from './MenuList.module.pcss';

const SEARCH_ENABLE_LIMIT = 10;

export type ISelectSize = 'auto' | 'small' | 'medium' | 'big' | 'big-limit' | 'responsive';
export type ISelectHeight = 'small' | 'medium' | 'big' | 'big-mobile';
export type ISelectMenuSize = 'small' | 'medium' | 'big' | 'large';
export type ISelectValue<T, Multi extends boolean> = Multi extends true ? IOption<T>[] : IOption<T>;

interface SelectProps<
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
    formatGroupLabel?: (group: any) => string;
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
    onBlur?: () => void;
    showIcons?: boolean;
    showOptionIcon?: boolean;
    optionTestIdPrefix?: string;
}

export const Select = <
    T,
    Multi extends boolean = false,
    ExtendOption extends Record<any, any> = object,
>(props: SelectProps<T, Multi, ExtendOption>) => {
    const selectClass = createMemo(() =>
        cn(
            { 'desktop-select-always': props.mobile === false },
            { 'mobile-select-always': props.mobile },
            'solid-select',
            { 'solid-select-big': props.size === 'big' },
            { 'solid-select-medium': props.size === 'medium' },
            { 'solid-select-small': props.size === 'small' },
            { 'solid-select-responsive': props.size === 'responsive' },
            { 'solid-select--big-limit': props.size === 'big-limit' },
            { 'solid-select--menu-small': props.menuSize === 'small' },
            { 'solid-select--menu-medium': props.menuSize === 'medium' },
            { 'solid-select--menu-big': props.menuSize === 'big' },
            { 'solid-select--menu-right': props.menuPosition === 'right' },
            { 'solid-select--borderless': props.borderless },
            { 'solid-select--adaptive-height': props.adaptiveHeight },
            { 'solid-select--height-small': props.height === 'small' },
            { 'solid-select--height-medium': props.height === 'medium' },
            { 'solid-select--height-big': props.height === 'big' },
            { 'solid-select--height-big-mobile': props.height === 'big-mobile' },
            { 'solid-select--dropdown': props.isDropdownSelect },
            { low: props.low },
            props.class,
        ),
    );

    const isSearchable = () =>
        props.isSearchable ?? props.options.length > SEARCH_ENABLE_LIMIT;

    const currentValue = createMemo(() => {
        if (!props.value) return null;
        if (props.isMulti) {
            return (props.value as (IOption<T> & ExtendOption)[]).map((v) => ({
                label: v.label,
                value: v,
            }));
        }
        const v = props.value as IOption<T> & ExtendOption;
        return { label: v.label, value: v };
    });

    const formattedOptions = createMemo(() =>
        props.options.map((opt) => ({
            label: opt.label,
            value: opt,
            icon: (opt as any).icon,
        })),
    );

    const handleChange = (newValue: any) => {
        if (props.isMulti) {
            const values = (Array.isArray(newValue) ? newValue : [newValue]).map(
                (v: any) => v.value,
            );
            props.onChange(values as any);
        } else {
            props.onChange(newValue?.value as any);
        }
    };

    return (
        <div class={selectClass()} id={props.id}>
            <SelectPrimitive.Select
                options={formattedOptions()}
                value={currentValue()}
                onChange={handleChange}
                multiple={props.isMulti}
                disabled={props.isDisabled}
                searchable={isSearchable()}
                placeholder={props.placeholder as string}
                menuPlacement={props.menuPlacement ?? 'bottom'}
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                {...({} as any)}
            />
        </div>
    );
};
