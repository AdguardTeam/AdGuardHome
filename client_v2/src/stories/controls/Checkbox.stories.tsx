import React, { useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react-webpack5';
import { Checkbox } from '../../common/controls';

const meta: Meta<typeof Checkbox> = {
    title: 'Controls/Checkbox',
    component: Checkbox,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        checked: {
            control: 'boolean',
            description: 'Whether the checkbox is checked',
        },
        disabled: {
            control: 'boolean',
            description: 'Whether the checkbox is disabled',
        },
        children: {
            control: 'text',
            description: 'Label text for the checkbox',
        },
        className: {
            control: 'text',
            description: 'CSS class name for the checkbox wrapper',
        },
        labelClassName: {
            control: 'text',
            description: 'CSS class name for the label text',
        },
        overflow: {
            control: 'boolean',
            description: 'Whether to apply text overflow styling to the label',
        },
        plusStyle: {
            control: 'boolean',
            description: 'Use plus/minus icons instead of check/uncheck icons',
        },
        id: {
            control: 'text',
            description: 'HTML id attribute for the checkbox input',
        },
        name: {
            control: 'text',
            description: 'HTML name attribute for the checkbox input',
        },
        onChange: { action: 'changed' },
        onClick: { action: 'clicked' },
    },
};

export default meta;

type Story = StoryObj<typeof Checkbox>;

const CheckboxWithState = (args: any) => {
    const [checked, setChecked] = useState(args.checked || false);
    return (
        <Checkbox
            {...args}
            checked={checked}
            onChange={(e) => {
                setChecked(e.target.checked);
                args.onChange?.(e);
            }}
        />
    );
};

export const Default: Story = {
    render: CheckboxWithState,
    args: {
        children: 'Default checkbox',
        id: 'default-checkbox',
    },
};

export const Checked: Story = {
    render: CheckboxWithState,
    args: {
        children: 'Checked checkbox',
        checked: true,
        id: 'checked-checkbox',
    },
};

export const Disabled: Story = {
    render: CheckboxWithState,
    args: {
        children: 'Disabled checkbox',
        disabled: true,
        id: 'disabled-checkbox',
    },
};

export const DisabledChecked: Story = {
    render: CheckboxWithState,
    args: {
        children: 'Disabled checked checkbox',
        checked: true,
        disabled: true,
        id: 'disabled-checked-checkbox',
    },
};

export const PlusStyle: Story = {
    render: CheckboxWithState,
    args: {
        children: 'Plus/minus style checkbox',
        plusStyle: true,
        id: 'plus-style-checkbox',
    },
};

export const PlusStyleChecked: Story = {
    render: CheckboxWithState,
    args: {
        children: 'Plus/minus style checked',
        plusStyle: true,
        checked: true,
        id: 'plus-style-checked-checkbox',
    },
};

export const WithOverflow: Story = {
    render: CheckboxWithState,
    args: {
        children: 'This is a very long label text that should demonstrate the overflow behavior when the text is too long to fit in the available space',
        overflow: true,
        id: 'overflow-checkbox',
    },
};

export const WithoutLabel: Story = {
    render: CheckboxWithState,
    args: {
        id: 'no-label-checkbox',
    },
};
