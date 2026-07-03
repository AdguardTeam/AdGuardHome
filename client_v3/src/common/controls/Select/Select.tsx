import { type JSX, For, Show, createMemo, createSignal } from 'solid-js';
import cn from 'clsx';
import { Select as ArkSelect, Combobox as ArkCombobox, createListCollection } from '@ark-ui/solid';
import { Icon } from 'panel/common/ui/Icon';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

import { SelectMultiValueDisplay } from './SelectMultiValueDisplay';
import { ComboboxMultiValueDisplay } from './ComboboxMultiValueDisplay';
import { ComboboxInputReset } from './ComboboxInputReset';
import { SelectItemContent } from './SelectItemContent';
import { ComboboxItemContent } from './ComboboxItemContent';

import { SelectProps, SEARCH_ENABLE_LIMIT } from './types';
import { optionToValue, filterOptions, getItemTestId } from './helpers';

import './Select.pcss';
import s from './MenuList.module.pcss';

export type {
    ISelectSize,
    ISelectHeight,
    ISelectMenuSize,
    ISelectValue,
    SelectProps,
} from './types';

export const Select = <
    T,
    Multi extends boolean = false,
    ExtendOption extends Record<any, any> = object,
>(
    props: SelectProps<T, Multi, ExtendOption>,
): JSX.Element => {
    /* ---- Derived state ---- */
    const selectClass = createMemo(() =>
        cn(
            { 'desktop-select-always': props.mobile === false },
            { 'mobile-select-always': props.mobile },
            'solid-select',
            { 'solid-select--multi': props.isMulti },
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

    const isSearchable = createMemo(() => {
        // Explicit isSearchable prop takes precedence over defaults.
        if (props.isSearchable !== undefined) {
            return props.isSearchable;
        }
        // Default: only non-multi selects with > SEARCH_ENABLE_LIMIT options are searchable.
        return !(props.isMulti ?? false) && props.options.length > SEARCH_ENABLE_LIMIT;
    });

    const isDisabled = createMemo(() => props.isDisabled ?? false);
    const isMulti = createMemo(() => props.isMulti ?? false);
    const isClearable = createMemo(() => props.isClearable ?? false);

    const [searchInput, setSearchInput] = createSignal('');
    // Label shown immediately after selection, before the async onChange round-trip
    // updates props.value. Always overridden by props.value when available.
    const [pendingLabel, setPendingLabel] = createSignal<string | undefined>();
    // DOM ref for defensive input clear in multi-select (avoids onInputValueChange loop).
    let inputRef: HTMLInputElement | undefined;

    const filteredOptions = createMemo(() => filterOptions(props.options, searchInput()));

    const collection = createMemo(() =>
        createListCollection({
            items: filteredOptions() as any[],
            itemToString: (item: any) => item.label,
            itemToValue: (item: any) => optionToValue(item.value),
            isItemDisabled: (item: any) => item.isDisabled ?? item.disabled ?? false,
        }),
    );

    const currentValue = createMemo<string[]>(() => {
        if (!props.value) return [];
        const arr = Array.isArray(props.value) ? props.value : [props.value];
        return arr.map((v: any) => optionToValue(v.value));
    });

    const handleValueChange = (details: { items: any[] }) => {
        if (isMulti()) {
            props.onChange(details.items as any);
        } else {
            const item = details.items[0];
            setPendingLabel(item?.label);
            props.onChange(item as any);
        }
    };

    // Attached to both Content and inner scroll div (scroll events don't bubble).
    const onContentScroll = (e: Event) => {
        if (!props.onMenuScrollToBottom) return;
        const el = e.currentTarget as HTMLElement;
        if (el.scrollHeight - el.scrollTop <= el.clientHeight + 50) {
            props.onMenuScrollToBottom();
        }
    };

    const positioning = createMemo(() => ({
        placement: (props.menuPlacement === 'top' ? 'top-start' : 'bottom-start') as
            | 'top-start'
            | 'bottom-start',
        sameWidth: true,
        fitViewport: true,
        flip: (props.menuPlacement === 'auto'
            ? true
            : props.menuPlacement === 'top'
              ? ['top-start', 'bottom-start']
              : ['bottom-start', 'top-start']) as boolean | ('top-start' | 'bottom-start')[],
    }));

    const testId = (option: any) => getItemTestId(props.optionTestIdPrefix, option.value);

    const clearAriaLabel = intl.getMessage('clear_btn');

    // Reads the label from the controlled value prop (option object).
    // Falls back to pendingLabel (set on selection) before props.value updates.
    const singleValueLabel = createMemo(() => {
        const value = props.value;
        if (value) {
            const arr = Array.isArray(value) ? value : [value];
            const label = (arr[0] as any)?.label;
            if (label) return label;
        }
        return pendingLabel() ?? '';
    });

    return (
        <Show
            when={isSearchable()}
            fallback={
                /* ===== Non-searchable mode: Ark UI Select ===== */
                <div class={selectClass()} id={props.id}>
                    <ArkSelect.Root
                        collection={collection()}
                        value={currentValue()}
                        onValueChange={handleValueChange}
                        multiple={isMulti()}
                        disabled={isDisabled()}
                        closeOnSelect={props.closeMenuOnSelect ?? true}
                        open={props.menuIsOpen}
                        onOpenChange={(details) => {
                            if (details.open) {
                                props.onMenuOpen?.();
                            } else {
                                props.onMenuClose?.();
                            }
                        }}
                        positioning={positioning()}
                    >
                        <Show when={!props.isDropdownSelect}>
                            <ArkSelect.Control>
                                <ArkSelect.Trigger
                                    ref={(el: HTMLButtonElement) => {
                                        if (props.autoFocus) el.focus();
                                    }}
                                >
                                    <Show
                                        when={!isMulti()}
                                        fallback={
                                            <div class="solid-select-multi-value-container">
                                                <SelectMultiValueDisplay
                                                    placeholder={props.placeholder as string}
                                                />
                                            </div>
                                        }
                                    >
                                        <ArkSelect.ValueText
                                            placeholder={props.placeholder as string}
                                        />
                                    </Show>
                                    <Show when={!isMulti() || !currentValue().length}>
                                        <ArkSelect.Indicator>
                                            <Icon icon="arrow_bottom" />
                                        </ArkSelect.Indicator>
                                    </Show>
                                </ArkSelect.Trigger>
                                <Show when={isClearable() && !isMulti()}>
                                    <ArkSelect.ClearTrigger aria-label={clearAriaLabel}>
                                        <Icon icon="cross" />
                                    </ArkSelect.ClearTrigger>
                                </Show>
                            </ArkSelect.Control>
                        </Show>
                        <ArkSelect.Positioner>
                            <ArkSelect.Content onScroll={onContentScroll}>
                                <Show
                                    when={!props.isLoading}
                                    fallback={
                                        <div class={theme.select.menuLoaderOverlay}>
                                            <Icon icon="loader" class={theme.select.menuLoader} />
                                        </div>
                                    }
                                >
                                    <ArkSelect.ItemGroup>
                                        <Show
                                            when={props.lazyList}
                                            fallback={
                                                <For each={props.options}>
                                                    {(option) => (
                                                        <ArkSelect.Item
                                                            item={option as any}
                                                            data-testid={testId(option)}
                                                        >
                                                            <SelectItemContent
                                                                option={option as any}
                                                                showOptionIcon={
                                                                    props.showOptionIcon
                                                                }
                                                                showIcons={props.showIcons}
                                                            />
                                                        </ArkSelect.Item>
                                                    )}
                                                </For>
                                            }
                                        >
                                            <div
                                                class={s.menuList}
                                                data-part="menu-list"
                                                onScroll={onContentScroll}
                                            >
                                                <For each={props.options}>
                                                    {(option) => (
                                                        <ArkSelect.Item
                                                            item={option as any}
                                                            data-testid={testId(option)}
                                                        >
                                                            <SelectItemContent
                                                                option={option as any}
                                                                showOptionIcon={
                                                                    props.showOptionIcon
                                                                }
                                                                showIcons={props.showIcons}
                                                            />
                                                        </ArkSelect.Item>
                                                    )}
                                                </For>
                                            </div>
                                        </Show>
                                    </ArkSelect.ItemGroup>
                                    <Show when={props.menuFooter}>{props.menuFooter}</Show>
                                </Show>
                            </ArkSelect.Content>
                        </ArkSelect.Positioner>
                    </ArkSelect.Root>
                </div>
            }
        >
            {/* ===== Searchable (Combobox) mode: Ark UI Combobox ===== */}
            <div class={selectClass()} id={props.id}>
                <ArkCombobox.Root
                    collection={collection()}
                    value={currentValue()}
                    onValueChange={handleValueChange}
                    multiple={isMulti()}
                    disabled={isDisabled()}
                    closeOnSelect={props.closeMenuOnSelect ?? true}
                    openOnClick
                    autoFocus={props.autoFocus}
                    open={props.menuIsOpen}
                    selectionBehavior="clear"
                    onInputValueChange={(details) => {
                        if (details.reason === 'input-change') {
                            setSearchInput(details.inputValue);
                        } else {
                            // Non-typing change (select/clear/close): reset filter.
                            setSearchInput('');
                            if (isMulti() && inputRef) {
                                inputRef.value = '';
                            }
                        }
                    }}
                    onOpenChange={(details) => {
                        if (details.open) {
                            setSearchInput('');
                            props.onMenuOpen?.();
                        } else {
                            // Stale search cleared by ComboboxInputReset.
                            props.onMenuClose?.();
                        }
                    }}
                    positioning={positioning()}
                >
                    <ComboboxInputReset />
                    <Show when={!props.isDropdownSelect}>
                        <ArkCombobox.Control
                            onClick={(e: MouseEvent) => {
                                // Click the input when clicking the control's dead zone.
                                if (e.target === e.currentTarget) {
                                    const input = (e.currentTarget as HTMLElement).querySelector(
                                        'input',
                                    );
                                    input?.click();
                                }
                            }}
                        >
                            <Show when={isMulti()}>
                                <div class="solid-select-multi-value-container">
                                    <ComboboxMultiValueDisplay
                                        placeholder={props.placeholder as string}
                                        inputId={props.inputId}
                                        onInputRef={(el) => {
                                            inputRef = el;
                                        }}
                                    />
                                </div>
                                {/* Clear button shown whenever values exist. */}
                                <Show when={currentValue().length > 0}>
                                    <ArkCombobox.ClearTrigger aria-label={clearAriaLabel}>
                                        <Icon icon="cross" />
                                    </ArkCombobox.ClearTrigger>
                                </Show>
                                <Show when={!currentValue().length}>
                                    <ArkCombobox.Trigger>
                                        <Icon icon="arrow_bottom" />
                                    </ArkCombobox.Trigger>
                                </Show>
                            </Show>
                            <Show when={!isMulti()}>
                                <div class="solid-combobox-single-value-wrapper">
                                    <Show
                                        when={!searchInput()}
                                        fallback={<span class="solid-combobox-single-value" />}
                                    >
                                        <Show
                                            when={singleValueLabel()}
                                            fallback={
                                                <span class="solid-select-placeholder">
                                                    {props.placeholder as string}
                                                </span>
                                            }
                                        >
                                            <span class="solid-combobox-single-value">
                                                {singleValueLabel()}
                                            </span>
                                        </Show>
                                    </Show>
                                    <ArkCombobox.Input
                                        id={props.inputId}
                                        placeholder=""
                                        ref={(el) => {
                                            inputRef = el;
                                        }}
                                    />
                                </div>
                                <Show when={isClearable() && currentValue().length > 0}>
                                    <ArkCombobox.ClearTrigger aria-label={clearAriaLabel}>
                                        <Icon icon="cross" />
                                    </ArkCombobox.ClearTrigger>
                                </Show>
                                <ArkCombobox.Trigger>
                                    <Icon icon="arrow_bottom" />
                                </ArkCombobox.Trigger>
                            </Show>
                        </ArkCombobox.Control>
                    </Show>
                    <ArkCombobox.Positioner>
                        <ArkCombobox.Content onScroll={onContentScroll}>
                            <Show
                                when={!props.isLoading}
                                fallback={
                                    <div class={theme.select.menuLoaderOverlay}>
                                        <Icon icon="loader" class={theme.select.menuLoader} />
                                    </div>
                                }
                            >
                                <ArkCombobox.ItemGroup>
                                    <Show
                                        when={props.lazyList}
                                        fallback={
                                            <For each={filteredOptions()}>
                                                {(option) => (
                                                    <ArkCombobox.Item
                                                        item={option as any}
                                                        data-testid={testId(option)}
                                                    >
                                                        <ComboboxItemContent
                                                            option={option as any}
                                                            showOptionIcon={props.showOptionIcon}
                                                            showIcons={props.showIcons}
                                                        />
                                                    </ArkCombobox.Item>
                                                )}
                                            </For>
                                        }
                                    >
                                        <div
                                            class={s.menuList}
                                            data-part="menu-list"
                                            onScroll={onContentScroll}
                                        >
                                            <For each={filteredOptions()}>
                                                {(option) => (
                                                    <ArkCombobox.Item
                                                        item={option as any}
                                                        data-testid={testId(option)}
                                                    >
                                                        <ComboboxItemContent
                                                            option={option as any}
                                                            showOptionIcon={props.showOptionIcon}
                                                            showIcons={props.showIcons}
                                                        />
                                                    </ArkCombobox.Item>
                                                )}
                                            </For>
                                        </div>
                                    </Show>
                                </ArkCombobox.ItemGroup>
                                <Show when={props.menuFooter}>{props.menuFooter}</Show>
                                <ArkCombobox.Empty>
                                    {intl.getMessage('nothing_found')}
                                </ArkCombobox.Empty>
                            </Show>
                        </ArkCombobox.Content>
                    </ArkCombobox.Positioner>
                </ArkCombobox.Root>
            </div>
        </Show>
    );
};
