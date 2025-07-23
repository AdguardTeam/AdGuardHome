import type { Meta, StoryObj } from '@storybook/react-webpack5';
import { Textarea } from '../../common/controls';

const meta: Meta<typeof Textarea> = {
    title: 'Controls/Textarea',
    component: Textarea,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        label: {
            control: 'text',
            description: 'Label text displayed above the textarea field',
        },
        placeholder: {
            control: 'text',
            description: 'Placeholder text shown when textarea is empty',
        },
        value: {
            control: 'text',
            description: 'Current value of the textarea field',
        },
        disabled: {
            control: 'boolean',
            description: 'Whether the textarea is disabled and cannot be interacted with',
        },
        errorMessage: {
            control: 'text',
            description: 'Error message displayed below the textarea when in error state',
        },
        rows: {
            control: 'number',
            description: 'Number of visible text lines for the textarea',
        },
        cols: {
            control: 'number',
            description: 'Visible width of the textarea in characters',
        },
        maxLength: {
            control: 'number',
            description: 'Maximum number of characters allowed in the textarea',
        },
        autoFocus: {
            control: 'boolean',
            description: 'Whether the textarea should automatically focus when mounted',
        },
        onChange: { action: 'changed' },
        onBlur: { action: 'blurred' },
        onFocus: { action: 'focused' },
    },
};

export default meta;

type Story = StoryObj<typeof Textarea>;

export const Default: Story = {
    args: {
        placeholder: 'Enter text...',
    },
};

export const WithLabel: Story = {
    args: {
        label: 'Label',
        placeholder: 'Enter text...',
    },
};

export const WithValue: Story = {
    args: {
        label: 'Textarea with value',
        value: 'Example text',
    },
};

export const Disabled: Story = {
    args: {
        label: 'Disabled textarea',
        placeholder: 'Cannot edit this field',
        disabled: true,
    },
};

export const WithError: Story = {
    args: {
        label: 'Textarea with error',
        value: 'Invalid value',
        errorMessage: 'This field has an error',
    },
};
