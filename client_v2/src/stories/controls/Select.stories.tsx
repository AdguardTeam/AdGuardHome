import type { Meta, StoryObj } from '@storybook/react-webpack5';
import { Select } from 'panel/common/controls/Select';

const meta: Meta<typeof Select> = {
    title: 'Controls/Select',
    component: Select,
    parameters: {
        layout: 'padded',
        docs: {
            story: {
                height: '250px',
            },
        },
    },
    tags: ['autodocs'],
    argTypes: {
        options: {
            control: false,
            description: 'Array of options or option groups to display in the select',
        },
        value: {
            control: false,
            description: 'Currently selected value(s)',
        },

        components: {
            control: false,
            description: 'Custom components to override default select components',
        },
        formatGroupLabel: {
            control: false,
            description: 'Function to format group labels',
        },
        isDisabled: {
            control: 'boolean',
            description: 'Whether the select is disabled and cannot be interacted with',
        },
        isMulti: {
            control: 'boolean',
            description: 'Allow multiple selections to be made',
        },
        isClearable: {
            control: 'boolean',
            description: 'Allow clearing the current selection with a clear button',
        },
        isSearchable: {
            control: 'boolean',
            description: 'Whether the select options can be searched/filtered',
        },
        isLoading: {
            control: 'boolean',
            description: 'Show loading indicator in the select',
        },
        size: {
            control: 'select',
            options: ['auto', 'small', 'medium', 'big', 'big-limit', 'responsive'],
            description: 'Size variant of the select component',
        },
        height: {
            control: 'select',
            options: ['small', 'medium', 'big', 'big-mobile'],
            description: 'Height variant of the select component',
        },
        menuSize: {
            control: 'select',
            options: ['small', 'medium', 'big', 'large'],
            description: 'Size variant of the dropdown menu',
        },
        menuPlacement: {
            control: 'select',
            options: ['top', 'bottom', 'auto'],
            description: 'Placement of the dropdown menu relative to the select',
        },
        borderless: {
            control: 'boolean',
            description: 'Whether to remove border styling from the select',
        },
        closeMenuOnSelect: {
            control: 'boolean',
            description: 'Whether to close the menu after selecting an option',
        },
        onMenuOpen: { action: 'menu opened' },
        onMenuClose: { action: 'menu closed' },
    },
};

export default meta;

type Story = StoryObj<typeof Select>;

export const Default: Story = {
    args: {
        options: [
            { value: 'option1', label: 'Option 1' },
            { value: 'option2', label: 'Option 2' },
            { value: 'option3', label: 'Option 3' },
        ],
        placeholder: 'Select an option',
    },
};

export const WithDefaultValue: Story = {
    args: {
        options: [
            { value: 'option1', label: 'Option 1' },
            { value: 'option2', label: 'Option 2' },
            { value: 'option3', label: 'Option 3' },
        ],
    },
};

export const Disabled: Story = {
    args: {
        options: [
            { value: 'option1', label: 'Option 1' },
            { value: 'option2', label: 'Option 2' },
            { value: 'option3', label: 'Option 3' },
        ],
        isDisabled: true,
        placeholder: 'Disabled select',
    },
};

export const Clearable: Story = {
    args: {
        options: [
            { value: 'option1', label: 'Option 1' },
            { value: 'option2', label: 'Option 2' },
            { value: 'option3', label: 'Option 3' },
        ],
        isClearable: true,
        placeholder: 'Clearable select',
    },
};

export const MultiSelect: Story = {
    args: {
        options: [
            { value: 'option1', label: 'Option 1' },
            { value: 'option2', label: 'Option 2' },
            { value: 'option3', label: 'Option 3' },
            { value: 'option4', label: 'Option 4' },
        ],
        isMulti: true,
        placeholder: 'Select multiple options',
    },
};

export const GroupedOptions: Story = {
    args: {
        options: [
            {
                label: 'Group 1',
                options: [
                    { value: 'option1', label: 'Option 1' },
                    { value: 'option2', label: 'Option 2' },
                ],
            },
            {
                label: 'Group 2',
                options: [
                    { value: 'option3', label: 'Option 3' },
                    { value: 'option4', label: 'Option 4' },
                ],
            },
        ],
        placeholder: 'Grouped options',
    },
};
