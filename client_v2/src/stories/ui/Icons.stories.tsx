import React from 'react';
import type { Meta, StoryObj } from '@storybook/react-webpack5';
import { ICON_VALUES } from 'panel/common/ui/Icons';
import { Icon } from 'panel/common/ui/Icon';

import s from './Icons.module.pcss';

const Icons = () => {
    const handleIconClick = (iconName: string) => {
        navigator.clipboard.writeText(iconName);
    };

    return (
        <div className={s.container}>
            <h2 className={s.title}>Icon Library</h2>
            <p className={s.description}>
                Click any icon to copy its name to clipboard. Total: {ICON_VALUES.length} icons
            </p>
            <div className={s.grid}>
                {ICON_VALUES.map((icon) => (
                    <div key={icon} className={s.iconItem} onClick={() => handleIconClick(icon)}>
                        <div className={s.iconContainer}>
                            <Icon icon={icon} />
                        </div>
                        <div className={s.iconName}>{icon}</div>
                    </div>
                ))}
            </div>
        </div>
    );
};

const meta: Meta<typeof Icons> = {
    title: 'UI/Icons',
    component: Icons,
    parameters: {
        layout: 'padded',
    },
};

export default meta;

type Story = StoryObj<typeof Icons>;

export const IconGallery: Story = {
    render: () => <Icons />,
};
