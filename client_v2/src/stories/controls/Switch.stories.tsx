import React, { useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react-webpack5';
import { Switch } from 'panel/common/controls/Switch';

const SwitchWrapper = (props: any) => {
    const [checked, setChecked] = useState(props.checked || false);

    const handleChange = (value: boolean) => {
        setChecked(value);
        props.onChange?.(value);
    };

    return <Switch {...props} checked={checked} onChange={handleChange} />;
};

const meta: Meta<typeof Switch> = {
    title: 'Controls/Switch',
    component: SwitchWrapper,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        id: {
            control: 'text',
            description: 'Unique identifier for the switch input element',
        },
        checked: {
            control: 'boolean',
            description: 'Whether the switch is in the checked/on state',
        },
        disabled: {
            control: 'boolean',
            description: 'Whether the switch is disabled and cannot be toggled',
        },
        children: {
            control: 'text',
            description: 'Label content displayed next to the switch',
        },
        className: {
            control: 'text',
            description: 'Additional CSS class for the switch container',
        },
        labelClassName: {
            control: 'text',
            description: 'Additional CSS class for the label text',
        },
        wrapperClassName: {
            control: 'text',
            description: 'Additional CSS class for the wrapper element',
        },
        handleChange: { action: 'changed' },
    },
};

export default meta;

type Story = StoryObj<typeof Switch>;

export const Default: Story = {
    args: {
        children: 'Default Switch',
    },
};

export const Checked: Story = {
    args: {
        children: 'Checked Switch',
        checked: true,
    },
};

export const Disabled: Story = {
    args: {
        children: 'Disabled Switch',
        disabled: true,
    },
};

export const DisabledChecked: Story = {
    args: {
        children: 'Disabled Checked Switch',
        disabled: true,
        checked: true,
    },
};
