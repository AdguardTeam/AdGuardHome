import type { Meta, StoryObj } from 'storybook-solidjs-vite';

import { Button } from 'panel/common/ui/Button';

import { Tooltip } from './Tooltip';

const positions = ['bottomRight', 'bottomLeft', 'topRight', 'topLeft', 'top'] as const;

const meta = {
    title: 'UI/Tooltip',
    component: Tooltip,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        content: { control: 'text' },
        position: {
            control: 'select',
            options: [...positions],
        },
        disabled: { control: 'boolean' },
        class: { control: false },
        overlayClass: { control: false },
        triggerAttrs: { control: false },
    },
    args: {
        content: 'Tooltip content',
    },
    render: (args) => (
        <Tooltip {...args}>
            <Button variant="secondary">Hover me</Button>
        </Tooltip>
    ),
} satisfies Meta<typeof Tooltip>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
    args: {},
};

export const Top: Story = {
    args: {
        position: 'top',
        content: 'Shown above the trigger',
    },
};

export const BottomLeft: Story = {
    args: {
        position: 'bottomLeft',
        content: 'Shown below and to the left',
    },
};

export const LongContent: Story = {
    args: {
        content:
            'A longer tooltip that wraps onto several lines so the overlay width can be checked',
        position: 'bottomLeft',
    },
};

export const Disabled: Story = {
    args: {
        disabled: true,
        content: 'This tooltip never opens',
    },
};
