import React, { useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react-webpack5';
import { Radio } from '../../common/controls';

const meta: Meta<typeof Radio> = {
    title: 'Controls/Radio',
    component: Radio,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        options: {
            control: false,
            description: 'Array of radio options with text and value properties',
        },
        value: {
            control: false,
            description: 'Currently selected value',
        },
        disabled: {
            control: 'boolean',
            description: 'Whether all radio options are disabled',
        },
        className: {
            control: 'text',
            description: 'CSS class name for individual radio items',
        },
        wrapClass: {
            control: 'text',
            description: 'CSS class name for the radio group wrapper',
        },
        handleChange: { action: 'changed' },
    },
};

export default meta;

type Story = StoryObj<typeof Radio>;

const RadioWithState = (args: any) => {
    const [value, setValue] = useState(args.value);
    return (
        <Radio
            {...args}
            value={value}
            handleChange={(newValue) => {
                setValue(newValue);
                args.handleChange?.(newValue);
            }}
        />
    );
};

export const Default: Story = {
    render: RadioWithState,
    args: {
        options: [
            { text: 'Option 1', value: 'option1' },
            { text: 'Option 2', value: 'option2' },
            { text: 'Option 3', value: 'option3' },
        ],
        value: 'option1',
    },
};

export const Disabled: Story = {
    render: RadioWithState,
    args: {
        options: [
            { text: 'Option 1', value: 'option1' },
            { text: 'Option 2', value: 'option2' },
            { text: 'Option 3', value: 'option3' },
        ],
        value: 'option2',
        disabled: true,
    },
};

export const NumberValues: Story = {
    render: RadioWithState,
    args: {
        options: [
            { text: 'Small (1)', value: 1 },
            { text: 'Medium (2)', value: 2 },
            { text: 'Large (3)', value: 3 },
        ],
        value: 2,
    },
};

export const BooleanValues: Story = {
    render: RadioWithState,
    args: {
        options: [
            { text: 'Yes', value: true },
            { text: 'No', value: false },
        ],
        value: true,
    },
};
