import React from 'react';
import type { Meta, StoryObj } from '@storybook/react-webpack5';
import { Dropdown } from 'panel/common/ui/Dropdown';
import theme from 'panel/lib/theme';

const meta: Meta<typeof Dropdown> = {
    title: 'UI/Dropdown',
    component: Dropdown,
    parameters: {
        layout: 'centered',
        docs: {
            story: {
                height: '200px',
            },
        },
    },
    tags: ['autodocs'],
    argTypes: {
        trigger: {
            control: 'select',
            options: ['click', 'hover'],
            description: 'How the dropdown is triggered',
        },
        position: {
            control: 'select',
            options: ['bottomLeft', 'bottomCenter', 'bottomRight', 'topLeft', 'topCenter', 'topRight'],
            description: 'Position of the dropdown overlay',
        },
        disabled: {
            control: 'boolean',
            description: 'Whether the dropdown is disabled',
        },
        noIcon: {
            control: 'boolean',
            description: 'Hide the dropdown arrow icon',
        },
        widthAuto: {
            control: 'boolean',
            description: 'Auto width for the overlay',
        },
        flex: {
            control: 'boolean',
            description: 'Use flex layout',
        },
        autoClose: {
            control: 'boolean',
            description: 'Auto close dropdown after timeout',
        },
        disableAnimation: {
            control: 'boolean',
            description: 'Disable dropdown animation',
        },
        minOverlayWidthMatchTrigger: {
            control: 'boolean',
            description: 'Match overlay width to trigger width',
        },
        className: {
            control: 'text',
            description: 'CSS class for the dropdown wrapper',
        },
        overlayClassName: {
            control: 'text',
            description: 'CSS class for the dropdown overlay',
        },
        menu: {
            control: false,
            description: 'Dropdown menu content',
        },
        children: {
            control: false,
            description: 'Dropdown trigger content',
        },
        onOpenChange: { action: 'openChanged' },
    },
};

export default meta;

type Story = StoryObj<typeof Dropdown>;

const actionMenu = (
    <div className={theme.dropdown.menu}>
        <div className={theme.dropdown.item}>Edit</div>
        <div className={theme.dropdown.item}>Duplicate</div>
        <div className={theme.dropdown.item}>Delete</div>
    </div>
);

export const Default: Story = {
    args: {
        trigger: 'click',
        menu: actionMenu,
        children: <div className={theme.dropdown.trigger}>Click me</div>,
    },
};

export const HoverTrigger: Story = {
    args: {
        trigger: 'hover',
        menu: actionMenu,
        children: <div className={theme.dropdown.trigger}>Hover me</div>,
    },
};

export const WithoutIcon: Story = {
    args: {
        trigger: 'click',
        menu: actionMenu,
        noIcon: true,
        children: <div className={theme.dropdown.trigger}>No arrow icon</div>,
    },
};

export const DifferentPositions: Story = {
    args: {
        trigger: 'click',
        menu: actionMenu,
        position: 'topLeft',
        children: <div className={theme.dropdown.trigger}>Top Left Position</div>,
    },
};

export const AutoClose: Story = {
    args: {
        trigger: 'click',
        menu: actionMenu,
        autoClose: true,
        children: <div className={theme.dropdown.trigger}>Auto Close (1s delay)</div>,
    },
};
