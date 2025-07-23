import type { Meta, StoryObj } from '@storybook/react-webpack5';
import { Button } from '../../common/ui';

const meta: Meta<typeof Button> = {
    title: 'UI/Button',
    component: Button,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        variant: {
            control: 'select',
            options: ['primary', 'secondary', 'danger', 'ghost'],
            description: 'Button variant',
        },
        size: {
            control: 'select',
            options: ['small', 'medium', 'big'],
            description: 'Button size',
        },
        disabled: {
            control: 'boolean',
            description: 'Disabled state',
        },
        onClick: { action: 'clicked' },
    },
};

export default meta;

type Story = StoryObj<typeof Button>;

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
