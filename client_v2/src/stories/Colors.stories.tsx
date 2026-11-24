import { For } from 'solid-js';
import cn from 'clsx';
import type { Meta, StoryObj } from 'storybook-solidjs-vite';

import theme from 'panel/lib/theme';

import s from './Colors.module.pcss';

const STEPS = ['0', '10', '20', '30', '40', '50', '60', '70', '80', '90', '100'] as const;

const PALETTES = [
    'product-primary',
    'product-secondary',
    'product-tertiary',
    'gray',
    'orange',
    'red',
] as const;

type TokenGroup = {
    title: string;
    tokens: string[];
};

const SEMANTIC_GROUPS: TokenGroup[] = [
    {
        title: 'Backgrounds',
        tokens: [
            '--default-page-background',
            '--hovered-page-background',
            '--pressed-page-background',
            '--page-background-additional',
            '--page-background-additional-hovered',
            '--page-background-additional-pressed',
            '--default-cards-background',
            '--default-popup-background',
            '--default-footer-background',
            '--default-item-divider',
        ],
    },
    {
        title: 'Text',
        tokens: [
            '--default-main-text',
            '--disabled-main-text',
            '--default-description-text',
            '--default-labels',
        ],
    },
    {
        title: 'Inputs',
        tokens: [
            '--default-input-background',
            '--default-active-input-stroke',
            '--default-inactive-input-stroke',
            '--default-placeholder',
            '--default-input-on-card-background',
            '--disabled-input-on-card-background',
            '--hovered-input-on-card-background',
            '--pressed-input-on-card-background',
        ],
    },
    {
        title: 'Dropdown',
        tokens: [
            '--default-dropdown-menu-background',
            '--hovered-dropdown-menu-background',
            '--pressed-dropdown-menu-background',
        ],
    },
    {
        title: 'Buttons',
        tokens: [
            '--default-main-button',
            '--hovered-main-button',
            '--pressed-main-button',
            '--disabled-main-button',
            '--default-primary-button-text',
            '--disabled-primary-button-text',
            '--default-primary-button-icon',
            '--default-secondary-button',
            '--hovered-secondary-button',
            '--pressed-secondary-button',
            '--disabled-secondary-button',
            '--default-secondary-button-stroke',
            '--disabled-secondary-button-stroke',
            '--default-secondary-card-button',
            '--hovered-secondary-card-button',
            '--pressed-secondary-card-button',
            '--disabled-secondary-card-button',
            '--default-danger-button',
            '--hovered-danger-button',
            '--pressed-danger-button',
            '--disabled-danger-button',
            '--hovered-secondary-danger-button',
            '--default-warning-button',
            '--hovered-warning-button',
            '--pressed-warning-button',
            '--disabled-warning-button',
        ],
    },
    {
        title: 'Links',
        tokens: [
            '--default-link',
            '--hovered-link',
            '--pressed-link',
            '--visited-link',
            '--disabled-link',
            '--default-attention-link',
            '--hovered-attention-link',
            '--pressed-attention-link',
            '--disabled-attention-link',
            '--default-additional-link',
            '--default-error-link',
        ],
    },
    {
        title: 'Icons',
        tokens: [
            '--default-product-icon',
            '--default-black-icons',
            '--default-gray-icons',
            '--disabled-gray-icons',
            '--disabled-product-icon',
            '--default-error-icon',
            '--stroke-icons-white-icons-default',
            '--stroke-icons-tertiary-icon-disabled',
            '--stroke-icons-secondary-icon-disabled',
            '--default-product-secondary-icon-stroke',
            '--default-attention-icon-stroke',
        ],
    },
    {
        title: 'Focus, stats and notifications',
        tokens: [
            '--default-focus-outline',
            '--default-stats-background',
            '--default-red-stat',
            '--default-tertiary-stat',
            '--default-notifications-attention',
            '--default-loaders-background',
            '--default-loaders-background-dark',
            '--default-loaders-primary',
            '--default-breadcrumbs',
        ],
    },
    {
        title: 'Overlays',
        tokens: [
            '--modal-iframe-overlay',
            '--modal-overlay',
            '--dialog-overlay',
            '--backdrop-overlay',
            '--selected-background',
        ],
    },
    {
        title: 'Toplines, promo and logo',
        tokens: [
            '--default-text-toplines-main',
            '--disabled-text-toplines-main',
            '--default-fills-toplines-adblocker',
            '--default-fills-toplines-vpn',
            '--fills-toplines-topline-background',
            '--text-toplines-topline-title',
            '--text-toplines-topline-description',
            '--text-toplines-topline-button-text-default',
            '--fills-toplines-topline-button-default',
            '--fills-toplines-topline-button-hovered',
            '--fills-toplines-topline-button-pressed',
            '--stroke-toplines-button-stroke-default',
            '--stroke-toplines-button-stroke-hovered',
            '--stroke-toplines-button-stroke-pressed',
            '--stroke-toplines-close-icon-default',
            '--stroke-toplines-close-icon-hovered',
            '--stroke-toplines-close-icon-pressed',
            '--fills-backgrounds-recent-activity',
            '--fills-backgrounds-recent-activity-changed',
            '--fills-backgrounds-recent-activity-blocked',
            '--default-logo-key-color',
            '--default-logo-color',
        ],
    },
    {
        title: 'Switch',
        tokens: [
            '--fills-switch-on-default',
            '--fills-switch-on-hovered',
            '--fills-switch-on-disabled',
            '--fills-switch-off-default',
            '--fills-switch-off-hovered',
            '--fills-switch-off-disabled',
            '--fills-switch-knob',
            '--fills-switch-knob-disabled',
        ],
    },
];

const Swatch = (props: { token: string }) => (
    <div class={s.item}>
        <div class={s.swatch} style={{ '--swatch': `var(${props.token})` }} aria-hidden="true" />
        <span class={cn(theme.text.t4, s.name)}>{props.token}</span>
    </div>
);

const PaletteGallery = () => (
    <section class={s.section}>
        <div class={cn(theme.title.h5, s.sectionTitle)}>Palette</div>
        <For each={PALETTES}>
            {(palette) => (
                <div class={s.ramp}>
                    <div class={cn(theme.text.t3, theme.text.semibold, s.groupTitle)}>
                        {`--${palette}`}
                    </div>
                    <div class={s.rampSteps}>
                        <For each={STEPS}>
                            {(step) => (
                                <div
                                    class={s.rampStep}
                                    style={{ '--swatch': `var(--${palette}-${step})` }}
                                    aria-hidden="true"
                                />
                            )}
                        </For>
                    </div>
                    <div class={s.rampLabels}>
                        <For each={STEPS}>
                            {(step) => <span class={cn(theme.text.t4, s.rampLabel)}>{step}</span>}
                        </For>
                    </div>
                </div>
            )}
        </For>
    </section>
);

const SemanticTokensGallery = () => (
    <section class={s.section}>
        <div class={cn(theme.title.h5, s.sectionTitle)}>Semantic tokens</div>
        <div class={cn(theme.text.t3, s.hint)}>
            Switch the theme in the toolbar to compare the light and dark values.
        </div>
        <For each={SEMANTIC_GROUPS}>
            {(group) => (
                <div class={s.group}>
                    <div class={cn(theme.text.t3, theme.text.semibold, s.groupTitle)}>
                        {group.title}
                    </div>
                    <div class={s.grid}>
                        <For each={group.tokens}>{(token) => <Swatch token={token} />}</For>
                    </div>
                </div>
            )}
        </For>
    </section>
);

const meta = {
    title: 'Theme/Colors',
    parameters: {
        layout: 'padded',
    },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const Palette: Story = {
    render: () => <PaletteGallery />,
};

export const SemanticTokens: Story = {
    render: () => <SemanticTokensGallery />,
};
