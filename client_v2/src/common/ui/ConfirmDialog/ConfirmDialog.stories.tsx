import { createSignal, Show } from 'solid-js';
import { fn } from 'storybook/test';
import type { Meta, StoryObj } from 'storybook-solidjs-vite';

import { Button, BUTTON_VARIANTS } from 'panel/common/ui/Button';

import { ConfirmDialog } from './ConfirmDialog';

type DialogArgs = Parameters<typeof ConfirmDialog>[0];

const DialogExample = (props: DialogArgs) => {
    const [open, setOpen] = createSignal(true);

    return (
        <>
            <Button variant="secondary" onClick={() => setOpen(true)}>
                Open dialog
            </Button>
            <Show when={open()}>
                <ConfirmDialog
                    {...props}
                    onClose={() => {
                        setOpen(false);
                        props.onClose();
                    }}
                />
            </Show>
        </>
    );
};

const meta = {
    title: 'UI/ConfirmDialog',
    component: ConfirmDialog,
    parameters: {
        layout: 'centered',
        docs: {
            // The dialog renders a fixed backdrop, which would cover the whole
            // docs page when several stories are inlined.
            story: {
                inline: false,
                height: '320px',
            },
        },
    },
    tags: ['autodocs'],
    argTypes: {
        title: { control: 'text' },
        text: { control: 'text' },
        buttonText: { control: 'text' },
        cancelText: { control: 'text' },
        submitDisabled: { control: 'boolean' },
        buttonVariant: {
            control: 'select',
            options: [...BUTTON_VARIANTS],
        },
        customFooter: { control: false },
        onConfirm: { action: 'confirmed' },
    },
    args: {
        buttonText: 'Confirm',
        cancelText: 'Cancel',
        onClose: fn(),
    },
    render: (args) => <DialogExample {...args} />,
} satisfies Meta<typeof ConfirmDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
    args: {
        buttonText: 'Confirm',
        cancelText: 'Cancel',
    },
};

export const WithTitleAndText: Story = {
    args: {
        title: 'Delete client',
        text: 'The client will be removed from the list. This cannot be undone.',
        buttonText: 'Delete',
        cancelText: 'Cancel',
    },
};

export const Danger: Story = {
    args: {
        title: 'Delete client',
        text: 'The client will be removed from the list.',
        buttonText: 'Delete',
        cancelText: 'Cancel',
        buttonVariant: 'danger',
    },
};

export const SubmitDisabled: Story = {
    args: {
        title: 'Save changes',
        text: 'Nothing to save yet.',
        buttonText: 'Save',
        cancelText: 'Cancel',
        submitDisabled: true,
    },
};

export const CustomFooter: Story = {
    args: {
        title: 'Custom actions',
    },
    render: (args) => (
        <DialogExample
            {...args}
            customFooter={
                <div>
                    <Button variant="primary">Use defaults</Button>
                </div>
            }
        />
    ),
};
