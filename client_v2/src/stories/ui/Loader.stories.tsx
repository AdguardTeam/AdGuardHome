import React from 'react';
import type { Meta, StoryObj } from '@storybook/react-webpack5';
import { Loader, InlineLoader, Button } from '../../common/ui';

const meta: Meta<typeof Loader> = {
    title: 'UI/Loader',
    component: Loader,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        color: {
            control: 'color',
            description: 'Color of the loader icon',
        },
        className: {
            control: 'text',
            description: 'CSS class name for the loader icon',
        },
        overlay: {
            control: 'boolean',
            description: 'Whether to show the loader with an overlay background',
        },
        overlayClassName: {
            control: 'text',
            description: 'CSS class name for the overlay wrapper',
        },
        icon: {
            control: 'text',
            description: 'Icon type to use for the loader (defaults to "loader")',
        },
    },
};

export default meta;

type Story = StoryObj<typeof Loader>;

export const Default: Story = {
    args: {},
};

export const WithColor: Story = {
    args: {
        color: '#007bff',
    },
};

export const WithOverlay: Story = {
    args: {
        overlay: true,
    },
    render: (args) => (
        <div
            style={{
                position: 'relative',
                width: '300px',
                height: '200px',
                background: '#f5f5f5',
                border: '1px solid #ddd',
            }}>
            <div style={{ padding: '20px' }}>
                <h3>Content behind overlay</h3>
                <p>This content should be covered by the loader overlay.</p>
            </div>
            <Loader {...args} />
        </div>
    ),
};

export const CustomIcon: Story = {
    args: {
        icon: 'refresh' as any,
    },
};

export const CustomClassName: Story = {
    args: {
        className: 'custom-loader-class',
    },
};

export const ColoredOverlay: Story = {
    args: {
        overlay: true,
        color: '#28a745',
    },
    render: (args) => (
        <div
            style={{
                position: 'relative',
                width: '300px',
                height: '200px',
                background: '#f8f9fa',
                border: '1px solid #dee2e6',
            }}>
            <div style={{ padding: '20px' }}>
                <h3>Loading Content</h3>
                <p>Please wait while we load your data...</p>
            </div>
            <Loader {...args} />
        </div>
    ),
};

// InlineLoader stories
const InlineLoaderMeta: Meta<typeof InlineLoader> = {
    title: 'UI/InlineLoader',
    component: InlineLoader,
    parameters: {
        layout: 'centered',
    },
    tags: ['autodocs'],
    argTypes: {
        className: {
            control: 'text',
            description: 'CSS class name for the inline loader',
        },
        icon: {
            control: 'text',
            description: 'Icon type to use for the loader (defaults to "loader")',
        },
    },
};

export { InlineLoaderMeta as InlineLoaderStories };

type InlineLoaderStory = StoryObj<typeof InlineLoader>;

export const InlineDefault: InlineLoaderStory = {
    args: {},
};

export const InlineInText: InlineLoaderStory = {
    render: (args) => (
        <div>
            <p>
                Loading data <InlineLoader {...args} /> please wait...
            </p>
        </div>
    ),
};

export const InlineInButton: InlineLoaderStory = {
    render: (args) => (
        <Button variant="primary" size="medium" leftAddon={<InlineLoader {...args} />}>
            Loading...
        </Button>
    ),
};

export const InlineCustomIcon: InlineLoaderStory = {
    args: {
        icon: 'refresh' as any,
    },
};

export const InlineMultiple: InlineLoaderStory = {
    render: (args) => (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span>Small:</span>
                <InlineLoader {...args} className="small-loader" />
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span>Default:</span>
                <InlineLoader {...args} />
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span>Large:</span>
                <InlineLoader {...args} className="large-loader" />
            </div>
        </div>
    ),
};
