import { For } from 'solid-js';
import cn from 'clsx';
import type { Meta, StoryObj } from 'storybook-solidjs-vite';

import theme from 'panel/lib/theme';

import s from './Typography.module.pcss';

const titles = [
    { name: '.h0', class: theme.title.h0 },
    { name: '.h1', class: theme.title.h1 },
    { name: '.h2', class: theme.title.h2 },
    { name: '.h3', class: theme.title.h3 },
    { name: '.h4', class: theme.title.h4 },
    { name: '.h5', class: theme.title.h5 },
    { name: '.h6', class: theme.title.h6 },
];

const texts = [
    { name: '.t1', class: theme.text.t1 },
    { name: '.t2', class: theme.text.t2 },
    { name: '.t3', class: theme.text.t3 },
    { name: '.t4', class: theme.text.t4 },
];

const modifiers = [
    { name: '.t2.condenced', class: cn(theme.text.t2, theme.text.condenced) },
    { name: '.t2.medium', class: cn(theme.text.t2, theme.text.medium) },
    { name: '.t2.semibold', class: cn(theme.text.t2, theme.text.semibold) },
];

const Titles = () => (
    <div class={s.group}>
        <For each={titles}>{(title) => <div class={title.class}>{title.name} Title Example</div>}</For>
    </div>
);

const Texts = () => (
    <div class={s.group}>
        <For each={texts}>{(text) => <div class={text.class}>{text.name} Text Example</div>}</For>
    </div>
);

const Modifiers = () => (
    <div class={s.group}>
        <For each={modifiers}>
            {(modifier) => <div class={modifier.class}>{modifier.name} Text Example</div>}
        </For>
        <span class={cn(theme.text.t4, s.label)}>Secondary description text</span>
    </div>
);

const meta = {
    title: 'Theme/Typography',
    parameters: {
        layout: 'padded',
    },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const TitlesAndTexts: Story = {
    render: () => (
        <>
            <Titles />
            <Texts />
        </>
    ),
};

export const TextModifiers: Story = {
    render: () => <Modifiers />,
};
