import React from 'react';
import type { Meta, StoryObj } from '@storybook/react-webpack5';
import theme from '../../lib/theme';

const Typography = () => {
    return (
        <div>
            <div>
                <div className={theme.title.h0}>.h0 Title Example</div>
                <div className={theme.title.h1}>.h1 Title Example</div>
                <div className={theme.title.h2}>.h2 Title Example</div>
                <div className={theme.title.h3}>.h3 Title Example</div>
                <div className={theme.title.h4}>.h4 Title Example</div>
                <div className={theme.title.h5}>.h5 Title Example</div>
                <div className={theme.title.h6}>.h6 Title Example</div>
            </div>

            <div>
                <div>
                    <div className={theme.text.t1}>.t1 Text Example</div>
                    <div className={theme.text.t2}>.t2 Text Example</div>
                    <div className={theme.text.t3}>.t3 Text Example</div>
                    <div className={theme.text.t4}>.t4 Text Example</div>
                </div>
            </div>
        </div>
    );
};

const meta: Meta<typeof Typography> = {
    title: 'Theme/Typography',
    component: Typography,
    parameters: {
        layout: 'padded',
    },
};

export default meta;

type Story = StoryObj<typeof Typography>;

export const TypographyStyles: Story = {
    render: () => <Typography />,
};
