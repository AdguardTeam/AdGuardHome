import type { Meta, StoryObj } from 'storybook-solidjs-vite';

import { Button } from 'panel/common/ui/Button';
import { Icon } from 'panel/common/ui/Icon';

const meta = {
    title: 'UI/Button',
    component: Button,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        variant: {
            control: 'select',
            options: ['primary', 'secondary', 'danger', 'secondary-danger', 'warning', 'ghost'],
            description: 'Button variant',
        },
        size: {
            control: 'select',
            options: ['very-small', 'small', 'medium', 'big'],
            description: 'Button size',
        },
        disabled: {
            control: 'boolean',
            description: 'Disabled state',
        },
        onCard: {
            control: 'boolean',
            description: 'Styles the button for a card surface',
        },
        compact: {
            control: 'boolean',
            description: 'Hides the empty addon slots',
        },
        children: {
            control: 'text',
            description: 'Button label',
        },
        leftAddon: { control: false },
        rightAddon: { control: false },
        onClick: { action: 'clicked' },
    },
} satisfies Meta<typeof Button>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Primary: Story = {
    args: {
        variant: 'primary',
        children: 'Primary Button',
    },
};

export const Secondary: Story = {
    args: {
        variant: 'secondary',
        children: 'Secondary Button',
    },
};

export const Danger: Story = {
    args: {
        variant: 'danger',
        children: 'Danger Button',
    },
};

export const SecondaryDanger: Story = {
    args: {
        variant: 'secondary-danger',
        children: 'Secondary Danger Button',
    },
};

export const Warning: Story = {
    args: {
        variant: 'warning',
        children: 'Warning Button',
    },
};

export const Ghost: Story = {
    args: {
        variant: 'ghost',
        children: 'Ghost Button',
    },
};

export const Disabled: Story = {
    args: {
        disabled: true,
        children: 'Disabled Button',
    },
};

export const Small: Story = {
    args: {
        size: 'small',
        children: 'Small Button',
    },
};

export const Big: Story = {
    args: {
        size: 'big',
        children: 'Big Button',
    },
};

export const WithLeftAddon: Story = {
    args: {
        children: 'Add filter',
    },
    render: (args) => <Button {...args} leftAddon={<Icon icon="plus" />} />,
};

export const Compact: Story = {
    args: {
        compact: true,
        children: 'Compact Button',
    },
};

export const OnCard: Story = {
    args: {
        onCard: true,
        variant: 'secondary',
        children: 'Button on a card',
    },
};
