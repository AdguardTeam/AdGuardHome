import { createSignal, untrack } from 'solid-js';
import type { Meta, StoryObj } from 'storybook-solidjs-vite';

import { Icon } from 'panel/common/ui/Icon';
import s from 'panel/stories/Stories.module.pcss';

import { INPUT_SIZES, Input } from './Input';
import { PasswordInput } from './PasswordInput';

type InputArgs = Parameters<typeof Input>[0];

const ControlledInput = (props: InputArgs) => {
    const [value, setValue] = createSignal(String(untrack(() => props.value) ?? ''));

    return (
        <Input
            {...props}
            value={value()}
            onChange={(event) => {
                setValue(event.currentTarget.value);
                props.onChange?.(event);
            }}
        />
    );
};

const PasswordExample = () => {
    const [value, setValue] = createSignal('secret-password');

    return (
        <PasswordInput
            class={s.column}
            label="Password"
            value={value()}
            onChange={setValue}
        />
    );
};

const meta = {
    title: 'Controls/Input',
    component: Input,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        label: { control: 'text', description: 'Label rendered above the field' },
        placeholder: { control: 'text', description: 'Placeholder text' },
        value: { control: 'text', description: 'Controlled value' },
        disabled: { control: 'boolean' },
        error: { control: 'boolean', description: 'Error styling' },
        errorMessage: { control: 'text', description: 'Error text under the field' },
        invalid: { control: 'boolean', description: 'Invalid styling without a message' },
        size: {
            control: 'select',
            options: [...INPUT_SIZES],
        },
        borderless: { control: 'boolean' },
        isClearable: { control: 'boolean', description: 'Shows the clear button' },
        maxLength: { control: 'number' },
        type: {
            control: 'select',
            options: ['text', 'number', 'email', 'url'],
        },
        prefixIcon: { control: false },
        suffixIcon: { control: false },
        class: { control: false },
        onChange: { action: 'changed' },
        onClear: { action: 'cleared' },
    },
    render: (args) => <ControlledInput {...args} class={s.column} />,
} satisfies Meta<InputArgs>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
    args: {
        placeholder: 'Enter a value',
    },
};

export const WithLabel: Story = {
    args: {
        label: 'Upstream DNS server',
        placeholder: 'tls://dns.example.com',
    },
};

export const Small: Story = {
    args: {
        size: 'small',
        value: '1.1.1.1',
    },
};

export const Medium: Story = {
    args: {
        size: 'medium',
        value: '1.1.1.1',
    },
};

export const Large: Story = {
    args: {
        size: 'large',
        value: '1.1.1.1',
    },
};

export const Disabled: Story = {
    args: {
        disabled: true,
        value: 'Read-only value',
    },
};

export const Invalid: Story = {
    args: {
        invalid: true,
        value: 'not-an-ip',
    },
};

export const ErrorMessage: Story = {
    args: {
        error: true,
        errorMessage: 'Enter a valid IP address',
        value: 'not-an-ip',
    },
};

export const Clearable: Story = {
    args: {
        isClearable: true,
        value: 'example.com',
    },
};

export const PrefixIcon: Story = {
    args: {
        placeholder: 'Search',
    },
    render: (args) => (
        <ControlledInput {...args} class={s.column} prefixIcon={<Icon icon="search" />} />
    ),
};

export const SuffixIcon: Story = {
    args: {
        value: 'example.com',
    },
    render: (args) => (
        <ControlledInput {...args} class={s.column} suffixIcon={<Icon icon="copy" />} />
    ),
};

export const Password: Story = {
    render: () => <PasswordExample />,
};
