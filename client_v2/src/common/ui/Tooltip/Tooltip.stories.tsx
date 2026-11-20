import type { Meta, StoryObj } from 'storybook-solidjs-vite';

import { Button } from 'panel/common/ui/Button';
import s from 'panel/stories/Stories.module.pcss';

import { TOOLTIP_POSITIONS, Tooltip } from './Tooltip';

const meta = {
    title: 'UI/Tooltip',
    component: Tooltip,
    parameters: {
        layout: 'centered',
        docs: {
            // The tooltip is positioned inside the docs story box, so it needs
            // room to open there without making that box scroll.
            story: {
                height: '320px',
            },
        },
    },
    tags: ['autodocs'],
    argTypes: {
        content: { control: 'text' },
        position: {
            control: 'select',
            options: [...TOOLTIP_POSITIONS],
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
        <Tooltip {...args} content={<div class={s.tooltipContent}>{args.content}</div>}>
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
