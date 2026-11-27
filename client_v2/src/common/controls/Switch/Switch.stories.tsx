import { createSignal } from 'solid-js';
import type { Meta, StoryObj } from 'storybook-solidjs-vite';

import { Switch } from './Switch';

const meta = {
    title: 'Controls/Switch',
    component: Switch,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        id: { control: false },
        checked: { control: 'boolean' },
        disabled: { control: 'boolean' },
        busy: { control: 'boolean', description: 'Keeps the normal palette while unavailable' },
        labelClass: { control: false },
        class: { control: false },
        wrapperClass: { control: false },
        onChange: { control: false },
        ref: { control: false },
        children: { control: 'text' },
    },
    args: {
        id: 'switch-story',
        checked: false,
        children: 'Label',
        onChange: () => {},
    },
    render: (args) => {
        const [checked, setChecked] = createSignal(!!args.checked);

        return (
            <Switch
                {...args}
                checked={checked()}
                onChange={() => setChecked((value) => !value)}
            />
        );
    },
} satisfies Meta<typeof Switch>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
    args: {},
};

export const Checked: Story = {
    args: {
        checked: true,
    },
};

export const Disabled: Story = {
    args: {
        disabled: true,
    },
};

export const Busy: Story = {
    args: {
        checked: true,
        disabled: true,
        busy: true,
    },
};
