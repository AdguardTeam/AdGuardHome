import { createSignal, untrack } from 'solid-js';
import { fn } from 'storybook/test';
import type { Meta, StoryObj } from 'storybook-solidjs-vite';

import type { IOption } from 'panel/lib/helpers/utils';
import s from 'panel/stories/Stories.module.pcss';

import { Select, type SelectProps } from './Select';
import { SELECT_MENU_SIZES, SELECT_SIZES } from './types';

type Option = IOption<string>;
type SelectArgs = SelectProps<string, false>;

const options: Option[] = [
    { value: 'all', label: 'All queries' },
    { value: 'blocked', label: 'Blocked' },
    { value: 'allowed', label: 'Allowed' },
    { value: 'rewritten', label: 'Rewritten' },
];

const manyOptions: Option[] = Array.from({ length: 14 }, (_, index) => ({
    value: `option-${index + 1}`,
    label: `Option ${index + 1}`,
}));

const SingleSelect = (props: SelectArgs) => {
    const [value, setValue] = createSignal<Option | undefined>(
        untrack(() => props.value) as Option | undefined,
    );

    return (
        <Select
            {...props}
            class={s.column}
            value={value()}
            onChange={(next) => {
                setValue(next as Option);
                props.onChange(next);
            }}
        />
    );
};

const MultiSelect = () => {
    const [value, setValue] = createSignal<Option[]>([options[0], options[1]]);

    return (
        <Select
            class={s.column}
            isMulti
            options={options}
            value={value()}
            onChange={(next) => setValue(next as Option[])}
        />
    );
};

const meta = {
    title: 'Controls/Select',
    component: Select,
    parameters: {
        layout: 'padded',
        docs: {
            story: {
                height: '320px',
            },
        },
    },
    tags: ['autodocs'],
    argTypes: {
        options: { control: false },
        value: { control: false },
        placeholder: { control: 'text' },
        isDisabled: { control: 'boolean' },
        isClearable: { control: 'boolean' },
        isSearchable: { control: 'boolean' },
        isLoading: { control: 'boolean' },
        isDropdownSelect: { control: 'boolean' },
        menuIsOpen: { control: 'boolean' },
        size: {
            control: 'select',
            options: [...SELECT_SIZES],
        },
        menuSize: {
            control: 'select',
            options: [...SELECT_MENU_SIZES],
        },
        class: { control: false },
        onChange: { action: 'changed' },
    },
    args: {
        options,
        onChange: fn(),
    },
    render: (args) => <SingleSelect {...args} />,
} satisfies Meta<SelectArgs>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
    args: {
        options,
        value: options[0],
    },
};

export const WithPlaceholder: Story = {
    args: {
        options,
        placeholder: 'Select a query status',
    },
};

export const Disabled: Story = {
    args: {
        options,
        value: options[0],
        isDisabled: true,
    },
};

export const Clearable: Story = {
    args: {
        options,
        value: options[0],
        isClearable: true,
    },
};

export const Searchable: Story = {
    args: {
        options: manyOptions,
        value: manyOptions[0],
        isSearchable: true,
    },
};

export const DropdownSelect: Story = {
    args: {
        options,
        value: options[0],
        isDropdownSelect: true,
    },
};

export const Multi: Story = {
    render: () => <MultiSelect />,
};
