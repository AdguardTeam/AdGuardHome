import React from 'react';
import type { Meta, StoryObj } from '@storybook/react-webpack5';
import { Input } from '../../common/controls';
import { Icon } from 'panel/common/ui';

const meta: Meta<typeof Input> = {
    title: 'Controls/Input',
    component: Input,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        label: {
            control: 'text',
            description: 'Label text displayed above the input field',
        },
        placeholder: {
            control: 'text',
            description: 'Placeholder text shown when input is empty',
        },
        value: {
            control: 'text',
            description: 'Current value of the input field',
        },
        disabled: {
            control: 'boolean',
            description: 'Whether the input is disabled and cannot be interacted with',
        },
        error: {
            control: 'boolean',
            description: 'Whether the input is in an error state',
        },
        errorMessage: {
            control: 'text',
            description: 'Error message displayed below the input when in error state',
        },
        size: {
            control: 'select',
            options: ['small', 'medium', 'large'],
            description: 'Size variant of the input (small, medium, large)',
        },
        borderless: {
            control: 'boolean',
            description: 'Whether to remove the input border styling',
        },
        invalid: {
            control: 'boolean',
            description: 'Whether the input is in an invalid state (visual styling)',
        },
        maxLength: {
            control: 'number',
            description: 'Maximum number of characters allowed in the input',
        },
        type: {
            control: 'select',
            options: ['text', 'password', 'email', 'number', 'tel', 'url'],
            description: 'HTML input type attribute',
        },
        autoFocus: {
            control: 'boolean',
            description: 'Whether the input should automatically focus when mounted',
        },
        prefixIcon: { control: false },
        suffixIcon: { control: false },
        onChange: { action: 'changed' },
        onBlur: { action: 'blurred' },
        onFocus: { action: 'focused' },
    },
};

export default meta;

type Story = StoryObj<typeof Input>;

export const Default: Story = {
    args: {
        placeholder: 'Enter text...',
    },
};

export const WithLabel: Story = {
    args: {
        label: 'Input Label',
        placeholder: 'Enter text...',
    },
};

export const WithValue: Story = {
    args: {
        label: 'Input with Value',
        value: 'Example text',
    },
};

export const Disabled: Story = {
    args: {
        label: 'Disabled Input',
        placeholder: 'Cannot edit this field',
        disabled: true,
    },
};

export const WithError: Story = {
    args: {
        label: 'Input with error',
        value: 'Invalid value',
        error: true,
        errorMessage: 'This field has an error',
    },
};

export const WithIcons: Story = {
    args: {
        label: 'Input with icons',
        prefixIcon: <Icon icon="check" />,
        suffixIcon: <Icon icon="dot" />,
    },
};
