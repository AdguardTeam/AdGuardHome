import { For } from 'solid-js';
import cn from 'clsx';
import type { Meta, StoryObj } from 'storybook-solidjs-vite';

import { Icon } from 'panel/common/ui/Icon';
import { ICON_VALUES } from 'panel/common/ui/Icons/Icons';
import theme from 'panel/lib/theme';

import s from './Icons.module.pcss';

const IconsGallery = () => (
    <div class={s.gallery}>
        <For each={ICON_VALUES}>
            {(icon) => (
                <div class={s.item}>
                    <Icon icon={icon} />
                    <span class={cn(theme.text.t4, s.name)}>{icon}</span>
                </div>
            )}
        </For>
    </div>
);

const meta = {
    title: 'UI/Icons',
    component: IconsGallery,
    parameters: {
        layout: 'padded',
    },
} satisfies Meta<typeof IconsGallery>;

export default meta;

type Story = StoryObj<typeof meta>;

export const IconGallery: Story = {
    render: () => <IconsGallery />,
};
