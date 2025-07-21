import React, { useState } from 'react';
import { withTranslation } from 'react-i18next';
import cn from 'clsx';

import { ICON_VALUES } from 'panel/common/ui/Icons';
import { Switch, Select, Textarea, Input, Radio, Checkbox } from 'panel/common/controls';
import { Dialog, ConfirmDialog, Link, Dropdown, Breadcrumbs, Button, Icon } from 'panel/common/ui';
import { CustomMultiValue } from 'panel/common/controls/Select';
import theme from 'panel/lib/theme';
import { RoutePath } from 'panel/components/Routes/Paths';

import styles from './styles.module.pcss';

// List of all CSS variable names from light.css
export const COLOR_VARIABLES = [
    '--default-page-background',
    '--hovered-page-background',
    '--pressed-page-background',
    '--fills-backgrounds-page-background-additional',
    '--default-cards-background',
    '--default-popup-background',
    '--default-footer-background',
    '--default-item-divider',
    '--default-main-text',
    '--disabled-main-text',
    '--default-description-text',
    '--default-labels',
    '--default-input-background',
    '--default-active-input-stroke',
    '--default-inactive-input-stroke',
    '--default-placeholder',
    '--default-input-on-card-background',
    '--disabled-input-on-card-background',
    '--default-dropdown-menu-background',
    '--hovered-dropdown-menu-background',
    '--pressed-dropdown-menu-background',
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
    '--default-link',
    '--hovered-link',
    '--pressed-link',
    '--visited-link',
    '--default-attention-link',
    '--hovered-attention-link',
    '--pressed-attention-link',
    '--disabled-attention-link',
    '--default-error-link',
    '--default-product-icon',
    '--default-black-icons',
    '--default-gray-icons',
    '--disabled-gray-icons',
    '--default-error-icon',
    '--stroke-icons-white-icons-default',
    '--stroke-icons-tertiary-icon-disabled',
    '--stroke-icons-secondary-icon-disabled',
    '--default-stats-background',
    '--default-red-stat',
    '--modal-iframe-overlay',
    '--modal-overlay',
    '--default-notifications-attention',
    '--default-logo-key-color',
    '--default-loaders-background',
    '--default-loaders-background-dark',
    '--default-loaders-primary',
    '--default-text-toplines-main',
    '--disabled-text-toplines-main',
    '--default-fills-toplines-adblocker',
    '--default-fills-toplines-vpn',
    '--default-breadcrumbs',
    '--fills-toplines-topline-background',
    '--text-toplines-topline-title',
    '--text-toplines-topline-description',
    '--text-toplines-topline-button-text-default',
    '--fills-toplines-topline-background-image',
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
    '--fills-switch-on-default',
    '--fills-switch-on-hovered',
    '--fills-switch-on-disabled',
    '--fills-switch-off-default',
    '--fills-switch-off-hovered',
    '--fills-switch-off-disabled',
    '--fills-switch-knob',
    '--fills-switch-knob-disabled',
];

const Expo = () => {
    const [checked, setChecked] = useState(false);
    const [switchChecked, setSwitchChecked] = useState(false);
    const [inputValue, setInputValue] = useState('');
    const [textareaValue, setTextareaValue] = useState('');
    const [radioValue, setRadioValue] = useState('1');
    const [dialogOpen, setDialogOpen] = useState(false);
    const [confirmOpen, setConfirmOpen] = useState(false);

    return (
        <div className={theme.layout.container}>
            <h1 className={cn(theme.title.h3, styles.title)}>Components</h1>

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Checkbox</h3>
            <div className={styles.contolsList}>
                <Checkbox onChange={(e) => setChecked(e.target.checked)} checked={checked}>
                    <span className={styles.label}>Test checkbox</span>
                </Checkbox>
                <Checkbox onChange={(e) => setChecked(e.target.checked)} checked={checked} disabled>
                    <span className={styles.label}>Disabled checkbox</span>
                </Checkbox>
            </div>

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Switch</h3>
            <div className={styles.contolsList}>
                <Switch handleChange={(e) => setSwitchChecked(e.target.checked)} checked={switchChecked} id="switch">
                    Test switch
                </Switch>
                <Switch
                    handleChange={(e) => setSwitchChecked(e.target.checked)}
                    checked={switchChecked}
                    id="switch2"
                    disabled>
                    Disabled switch
                </Switch>
            </div>

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Radio</h3>
            <Radio
                handleChange={(v) => setRadioValue(v)}
                value={radioValue}
                options={[
                    { text: 'Option 1', value: '1' },
                    { text: 'Option 2', value: '2' },
                    { text: 'Option 3', value: '3' },
                ]}
            />

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Buttons</h3>
            <div className={styles.buttonsContainer}>
                <Button variant="primary">Primary</Button>
                <Button variant="secondary">Secondary</Button>
                <Button variant="ghost">Ghost</Button>
                <Button variant="primary" size="small">
                    Small
                </Button>
                <Button variant="primary" size="medium">
                    Medium
                </Button>
                <Button variant="primary" size="big">
                    Big
                </Button>
                <Button variant="primary" leftAddon={<Icon icon="lang" />} rightAddon={<Icon icon="lang" />}>
                    Button with icon
                </Button>
                <Button variant="primary" disabled>
                    Disabled button
                </Button>
                <Button variant="secondary" disabled>
                    Disabled secondary button
                </Button>
            </div>

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Input</h3>
            <Input
                id="input1"
                type="text"
                value={inputValue}
                label="Label"
                onChange={(e) => setInputValue(e.target.value)}
                placeholder="Enter text"
                suffixIcon={<Icon icon="lang" />}
                prefixIcon={<Icon icon="lang" />}
            />

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Textarea</h3>
            <Textarea
                id="textarea"
                value={textareaValue}
                onChange={(e) => setTextareaValue(e.target.value)}
                placeholder="Enter text"
                label="Label"
            />

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Dialog</h3>
            <Button variant="primary" size="small" onClick={() => setDialogOpen(true)} style={{ marginBottom: 8 }}>
                Open Dialog
            </Button>
            {dialogOpen && (
                <Dialog visible title="Dialog Title" onClose={() => setDialogOpen(false)}>
                    <div className={theme.dialog.body}>Dialog content goes here.</div>
                </Dialog>
            )}

            <h3 className={cn(theme.title.h5, styles.subtitle)}>ConfirmDialog</h3>
            <Button variant="primary" size="small" onClick={() => setConfirmOpen(true)} style={{ marginBottom: 8 }}>
                Open ConfirmDialog
            </Button>
            {confirmOpen && (
                <ConfirmDialog
                    title="Decrease log rotation interval?"
                    text="This will delete all logs older than 6 hours"
                    onClose={() => setConfirmOpen(false)}
                    onConfirm={() => setConfirmOpen(false)}
                    buttonVariant="danger"
                    buttonText="Yes, decrease"
                    cancelText="Cancel"
                />
            )}

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Link</h3>
            <Link to={RoutePath.SettingsPage} className={theme.link.link}>
                Go to settings
            </Link>

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Dropdown</h3>
            <Dropdown
                position="bottomLeft"
                trigger="click"
                menu={
                    <div className={theme.dropdown.menu}>
                        <button type="button" className={theme.dropdown.item}>
                            Item 1
                        </button>
                        <button type="button" className={theme.dropdown.item}>
                            Item 2
                        </button>
                    </div>
                }>
                <span className={theme.dropdown.text}>Open Dropdown</span>
            </Dropdown>

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Breadcrumbs</h3>
            <Breadcrumbs
                parentLinks={[
                    { path: RoutePath.Dashboard, title: 'Dashboard' },
                    { path: RoutePath.Logs, title: 'Logs' },
                ]}
                currentTitle="Current Page"
            />

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Select</h3>
            <div className={styles.selectExamples}>
                <div className={styles.selectExample}>
                    <h4 className={styles.exampleTitle}>Basic Select</h4>
                    <Select
                        options={[
                            { value: 'option1', label: 'Option 1' },
                            { value: 'option2', label: 'Option 2' },
                            { value: 'option3', label: 'Option 3' },
                        ]}
                        onChange={(selected) => console.log('Selected:', selected)}
                        placeholder="Select value"
                    />
                </div>

                <div className={styles.selectExample}>
                    <h4 className={styles.exampleTitle}>Multi-select</h4>
                    <Select
                        isMulti
                        options={[
                            { value: 'option1', label: 'Option 1' },
                            { value: 'option2', label: 'Option 2' },
                            { value: 'option3', label: 'Option 3' },
                            { value: 'option4', label: 'Option 4' },
                        ]}
                        components={{ MultiValue: CustomMultiValue }}
                        onChange={(selected) => console.log('Selected:', selected)}
                        placeholder="Select multiple options"
                    />
                </div>

                <div className={styles.selectExample}>
                    <h4 className={styles.exampleTitle}>Disabled</h4>
                    <Select
                        isDisabled={true}
                        options={[
                            { value: 'option1', label: 'Option 1' },
                            { value: 'option2', label: 'Option 2' },
                        ]}
                        onChange={(selected) => console.log('Selected:', selected)}
                        placeholder="Disabled select"
                    />
                </div>

                <div className={styles.selectExample}>
                    <h4 className={styles.exampleTitle}>Clearable</h4>
                    <Select
                        isClearable={true}
                        options={[
                            { value: 'option1', label: 'Option 1' },
                            { value: 'option2', label: 'Option 2' },
                            { value: 'option3', label: 'Option 3' },
                        ]}
                        onChange={(selected) => console.log('Selected:', selected)}
                        placeholder="Clearable select"
                    />
                </div>

                <div className={styles.selectExample}>
                    <h4 className={styles.exampleTitle}>Group Options</h4>
                    <Select<string>
                        options={[
                            {
                                label: 'Group 1',
                                options: [
                                    { value: 'option1', label: 'Option 1' },
                                    { value: 'option2', label: 'Option 2' },
                                ],
                            },
                            {
                                label: 'Group 2',
                                options: [
                                    { value: 'option3', label: 'Option 3' },
                                    { value: 'option4', label: 'Option 4' },
                                ],
                            },
                        ]}
                        onChange={(selected) => console.log('Selected:', selected)}
                        placeholder="Grouped options"
                    />
                </div>
            </div>

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Icons</h3>
            <div className={styles.iconsContainer}>
                {ICON_VALUES.map((icon) => (
                    <div key={icon} className={styles.iconItem}>
                        <Icon icon={icon} className={styles.icon} />
                        <div className={styles.iconName}>{icon}</div>
                    </div>
                ))}
            </div>

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Typography (theme.title & theme.text)</h3>
            <div className={styles.typographySection}>
                <div className={styles.typographyBlock}>
                    <div className={styles.typographyHeading}>Title classes:</div>
                    <div className={styles.typographyList}>
                        <div className={theme.title.h0}>.h0 Title Example</div>
                        <div className={theme.title.h1}>.h1 Title Example</div>
                        <div className={theme.title.h2}>.h2 Title Example</div>
                        <div className={theme.title.h3}>.h3 Title Example</div>
                        <div className={theme.title.h4}>.h4 Title Example</div>
                        <div className={theme.title.h5}>.h5 Title Example</div>
                        <div className={theme.title.h6}>.h6 Title Example</div>
                    </div>
                </div>
                <div className={styles.typographyBlock}>
                    <div className={styles.typographyHeading}>Text classes:</div>
                    <div className={styles.typographyList}>
                        <div className={theme.text.t1}>.t1 Text Example</div>
                        <div className={theme.text.t2}>.t2 Text Example</div>
                        <div className={theme.text.t3}>.t3 Text Example</div>
                        <div className={theme.text.t4}>.t4 Text Example</div>
                    </div>
                </div>
            </div>

            <h3 className={cn(theme.title.h5, styles.subtitle)}>Colors</h3>
            <ul className={styles.colorsContainer}>
                {COLOR_VARIABLES.map((varName) => (
                    <li key={varName} className={styles.colorSwatch}>
                        <div className={styles.colorBox} style={{ background: `var(${varName})` }} />
                        <div className={styles.colorName}>{varName}</div>
                    </li>
                ))}
            </ul>
        </div>
    );
};

export default withTranslation()(Expo);
