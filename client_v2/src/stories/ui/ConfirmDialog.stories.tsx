import React, { useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react-webpack5';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { Button } from 'panel/common/ui/Button';

const meta: Meta<typeof ConfirmDialog> = {
    title: 'UI/ConfirmDialog',
    component: ConfirmDialog,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        title: {
            control: 'text',
            description: 'Dialog title',
        },
        text: {
            control: 'text',
            description: 'Dialog body text',
        },
        buttonText: {
            control: 'text',
            description: 'Confirm button text',
        },
        cancelText: {
            control: 'text',
            description: 'Cancel button text',
        },
        buttonVariant: {
            control: 'select',
            options: ['primary', 'secondary', 'danger', 'ghost'],
            description: 'Confirm button variant',
        },
        submitId: {
            control: 'text',
            description: 'HTML id for the confirm button',
        },
        cancelId: {
            control: 'text',
            description: 'HTML id for the cancel button',
        },
        wrapClassName: {
            control: 'text',
            description: 'CSS class name for the dialog wrapper',
        },
        customFooter: {
            control: false,
            description: 'Custom footer content to replace default buttons',
        },
        onClose: { action: 'closed' },
        onConfirm: { action: 'confirmed' },
    },
};

export default meta;

type Story = StoryObj<typeof ConfirmDialog>;

const ConfirmDialogWithTrigger = (args: any) => {
    const [isOpen, setIsOpen] = useState(false);

    return (
        <div>
            <Button onClick={() => setIsOpen(true)}>Open Dialog</Button>
            {isOpen && (
                <ConfirmDialog
                    {...args}
                    onClose={() => {
                        setIsOpen(false);
                        args.onClose?.();
                    }}
                    onConfirm={() => {
                        setIsOpen(false);
                        args.onConfirm?.();
                    }}
                />
            )}
        </div>
    );
};

export const Default: Story = {
    render: ConfirmDialogWithTrigger,
    args: {
        title: 'Confirm Action',
        text: 'Are you sure you want to perform this action?',
        buttonText: 'Confirm',
        cancelText: 'Cancel',
    },
};

export const DeleteConfirmation: Story = {
    render: ConfirmDialogWithTrigger,
    args: {
        title: 'Delete Item',
        text: 'This action cannot be undone. Are you sure you want to delete this item?',
        buttonText: 'Delete',
        cancelText: 'Cancel',
        buttonVariant: 'danger',
    },
};

export const WithoutTitle: Story = {
    render: ConfirmDialogWithTrigger,
    args: {
        text: 'Are you sure you want to continue?',
        buttonText: 'Yes',
        cancelText: 'No',
    },
};

export const WithoutText: Story = {
    render: ConfirmDialogWithTrigger,
    args: {
        title: 'Confirm',
        buttonText: 'OK',
        cancelText: 'Cancel',
    },
};

export const LongContent: Story = {
    render: ConfirmDialogWithTrigger,
    args: {
        title: 'Important Notice',
        text: 'This is a longer confirmation dialog with more detailed information. It explains the consequences of the action and provides additional context to help the user make an informed decision. The text can span multiple lines and contain important details about what will happen when the user confirms the action.',
        buttonText: 'I Understand',
        cancelText: 'Cancel',
    },
};

export const CustomButtons: Story = {
    render: ConfirmDialogWithTrigger,
    args: {
        title: 'Save Changes',
        text: 'You have unsaved changes. What would you like to do?',
        buttonText: 'Save',
        cancelText: 'Discard',
        buttonVariant: 'primary',
    },
};
