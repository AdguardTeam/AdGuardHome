import { createSignal, untrack } from 'solid-js';
import type { Meta, StoryObj } from 'storybook-solidjs-vite';

import { COMMENT_LINE_TOKENS } from 'panel/helpers/constants';
import s from 'panel/stories/Stories.module.pcss';

import { TEXTAREA_SIZES, Textarea } from './Textarea';

type TextareaArgs = Parameters<typeof Textarea>[0];

const ControlledTextarea = (props: TextareaArgs) => {
    const [value, setValue] = createSignal(String(untrack(() => props.value) ?? ''));

    return (
        <Textarea
            {...props}
            value={value()}
            onChange={(event) => {
                setValue(event.currentTarget.value);
                props.onChange?.(event);
            }}
        />
    );
};

const rulesValue = `# Block ads
||ads.example.com^

# Allow a site
@@||good.example.com^`;

const meta = {
    title: 'Controls/Textarea',
    component: Textarea,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        label: { control: 'text' },
        placeholder: { control: 'text' },
        value: { control: 'text' },
        rows: { control: 'number' },
        size: {
            control: 'select',
            options: [...TEXTAREA_SIZES],
        },
        disabled: { control: 'boolean' },
        readOnly: { control: 'boolean' },
        errorMessage: { control: 'text' },
        onCard: { control: 'boolean' },
        isClearable: { control: 'boolean' },
        highlightComments: { control: 'boolean' },
        resizable: { control: 'boolean' },
        commentPrefixes: { control: false },
        class: { control: false },
        onChange: { action: 'changed' },
        onClear: { action: 'cleared' },
    },
    args: {
        placeholder: 'Enter a value',
    },
    render: (args) => <ControlledTextarea {...args} class={s.column} />,
} satisfies Meta<TextareaArgs>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithLabel: Story = {
    args: {
        label: 'Ignored domains',
        placeholder: 'example.com',
    },
};

export const Medium: Story = {
    args: {
        size: 'medium',
        value: 'example.com',
    },
};

export const Large: Story = {
    args: {
        size: 'large',
        value: 'example.com',
    },
};

export const Compact: Story = {
    args: {
        size: 'compact',
        value: 'example.com',
    },
};

export const Disabled: Story = {
    args: {
        disabled: true,
        value: 'Read-only value',
    },
};

export const ErrorMessage: Story = {
    args: {
        errorMessage: 'Enter at least one domain',
        value: 'not a domain',
    },
};

export const Clearable: Story = {
    args: {
        isClearable: true,
        value: 'example.com',
    },
};

export const OnCard: Story = {
    args: {
        onCard: true,
        size: 'medium',
        value: 'example.com',
    },
};

export const CommentHighlight: Story = {
    args: {
        highlightComments: true,
        commentPrefixes: COMMENT_LINE_TOKENS,
        size: 'large',
        value: rulesValue,
    },
};

export const WithRows: Story = {
    args: {
        rows: 12,
        highlightComments: true,
        commentPrefixes: COMMENT_LINE_TOKENS,
        value: rulesValue,
    },
};

export const Resizable: Story = {
    args: {
        rows: 12,
        highlightComments: true,
        commentPrefixes: COMMENT_LINE_TOKENS,
        value: rulesValue,
        resizable: true,
    },
};
