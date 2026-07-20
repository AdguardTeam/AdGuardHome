import { createSignal, createEffect, onMount, Show } from 'solid-js';
import cn from 'clsx';
import { useNavigate } from '@solidjs/router';

import { Dropdown } from 'panel/common/ui/Dropdown';
import { Icon } from 'panel/common/ui/Icon';
import { PageLoader } from 'panel/common/ui/Loader';
import { SettingRow } from 'panel/common/ui/SettingRow';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { useDialog } from 'panel/hooks/useDialog';
import { Paths } from 'panel/components/Routes/Paths';
import {
    dhcpState,
    getDhcpStatus,
    getDhcpInterfaces,
    setDhcpConfig,
    resetDhcp,
} from 'panel/stores/dhcp';
import { addWarningToast } from 'panel/stores/toasts';

import { DhcpToggle } from './blocks/DhcpToggle';
import { InterfaceSelector } from './blocks/InterfaceSelector';
import { DhcpV4Modal } from './blocks/DhcpV4Modal';
import { DhcpV6Modal } from './blocks/DhcpV6Modal';
import type { V4Config } from './blocks/DhcpV4Modal';
import type { V6Config } from './blocks/DhcpV6Modal';
import s from './styles.module.pcss';

export const Dhcp = () => {
    const navigate = useNavigate();
    const v4Dialog = useDialog();
    const v6Dialog = useDialog();
    const resetDialog = useDialog();

    const [selectedInterface, setSelectedInterface] = createSignal(dhcpState.interface_name || '');
    const [showAllIps, setShowAllIps] = createSignal(false);
    const [menuOpen, setMenuOpen] = createSignal(false);

    createEffect(() => {
        if (dhcpState.interface_name) {
            setSelectedInterface(dhcpState.interface_name);
        }
    });

    onMount(async () => {
        await getDhcpStatus();

        if (dhcpState.dhcp_available) {
            await getDhcpInterfaces();
            if (!selectedInterface() && dhcpState.interfaces) {
                const firstIface = Object.keys(dhcpState.interfaces)[0];
                if (firstIface) setSelectedInterface(firstIface);
            }
        }
    });

    const handleSaveV4Config = (values: V4Config) => {
        setDhcpConfig({ interface_name: selectedInterface(), v4: values });
        v4Dialog.closeDialog();
    };

    const handleSaveV6Config = (values: V6Config) => {
        setDhcpConfig({ interface_name: selectedInterface(), v6: values });
        v6Dialog.closeDialog();
    };

    const handleInterfaceChange = (name: string) => {
        setSelectedInterface(name);
        setShowAllIps(false);
    };

    createEffect(() => {
        const check = dhcpState.check;
        if (
            !dhcpState.enabled &&
            check &&
            (check.v4?.other_server?.found === 'yes' || check.v6?.other_server?.found === 'yes')
        ) {
            addWarningToast({ error: intl.getMessage('dhcp_warning') });
        }
    });

    const hasIpv4 = () =>
        !!(dhcpState.interfaces && dhcpState.interfaces[selectedInterface()]?.ipv4_addresses);

    const hasIpv6 = () =>
        !!(dhcpState.interfaces && dhcpState.interfaces[selectedInterface()]?.ipv6_addresses);

    const handleResetClick = () => {
        setMenuOpen(false);
        resetDialog.openDialog();
    };

    const resetMenu = (
        <div
            class={cn(theme.dropdown.item, theme.dropdown.item_danger, theme.dropdown.item_large)}
            onClick={handleResetClick}
        >
            {intl.getMessage('reset_dhcp_settings')}
        </div>
    );

    const isLoaded = () => !dhcpState.processing && !dhcpState.processingInterfaces;

    return (
        <div class={theme.layout.container}>
            <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <Show when={isLoaded()} fallback={<PageLoader />}>
                    <Show when={!dhcpState.dhcp_available}>
                        <div class={s.unavailable}>
                            <h1 class={cn(theme.title.h4, theme.title.h3_tablet)}>
                                {intl.getMessage('unavailable_dhcp')}
                            </h1>
                            <div class={theme.text.t2}>
                                {intl.getMessage('unavailable_dhcp_desc')}
                            </div>
                        </div>
                    </Show>

                    <Show when={dhcpState.dhcp_available}>
                        <div class={s.header}>
                            <h1
                                class={cn(
                                    theme.layout.title,
                                    theme.title.h4,
                                    theme.title.h3_tablet,
                                    s.title,
                                )}
                            >
                                {intl.getMessage('dhcp')}
                            </h1>
                            <Dropdown
                                position="bottomRight"
                                noIcon
                                open={menuOpen()}
                                onOpenChange={setMenuOpen}
                                menu={resetMenu}
                            >
                                <button
                                    type="button"
                                    class={cn(theme.form.action, s.menuButton)}
                                    aria-label={intl.getMessage('reset_dhcp_settings')}
                                >
                                    <Icon icon="bullets" />
                                </button>
                            </Dropdown>
                        </div>

                        <DhcpToggle
                            selectedInterface={selectedInterface}
                            onToggleOn={v4Dialog.openDialog}
                        />

                        <div class={s.interfaceSection}>
                            <InterfaceSelector
                                selectedInterface={selectedInterface}
                                onInterfaceChange={handleInterfaceChange}
                                showAllIps={showAllIps}
                                onShowAllIps={() => setShowAllIps(true)}
                            />
                        </div>

                        <div class={s.settingsColumn}>
                            <SettingRow
                                id="dhcp_v4"
                                variant="link"
                                title={intl.getMessage('dhcp_ipv4_settings')}
                                value={dhcpState.v4?.gateway_ip || ''}
                                disabled={!hasIpv4()}
                                onClick={v4Dialog.openDialog}
                            />
                            <SettingRow
                                id="dhcp_v6"
                                variant="link"
                                title={intl.getMessage('dhcp_ipv6_settings')}
                                value={
                                    hasIpv6()
                                        ? dhcpState.v6?.range_start ||
                                          intl.getMessage('dhcp_form_range_start')
                                        : intl.getMessage('dhcp_v6_unavailable')
                                }
                                disabled={!hasIpv6()}
                                onClick={() => hasIpv6() && v6Dialog.openDialog()}
                            />
                            <SettingRow
                                id="dhcp_leases_link"
                                variant="link"
                                title={intl.getMessage('dhcp_leases_title')}
                                onClick={() => navigate(Paths.DhcpLeases)}
                            />
                        </div>

                        <DhcpV4Modal
                            open={v4Dialog.open()}
                            selectedInterface={selectedInterface}
                            onClose={v4Dialog.closeDialog}
                            onSave={handleSaveV4Config}
                        />

                        <DhcpV6Modal
                            open={v6Dialog.open()}
                            selectedInterface={selectedInterface}
                            onClose={v6Dialog.closeDialog}
                            onSave={handleSaveV6Config}
                        />

                        <Show when={resetDialog.open()}>
                            <ConfirmDialog
                                title={intl.getMessage('reset_settings')}
                                text={intl.getMessage('dhcp_reset')}
                                buttonText={intl.getMessage('reset_settings_confirm')}
                                cancelText={intl.getMessage('cancel')}
                                buttonVariant="danger"
                                submitDisabled={!!dhcpState.processingReset}
                                onConfirm={() => {
                                    resetDhcp();
                                    getDhcpStatus();
                                    resetDialog.closeDialog();
                                }}
                                onClose={resetDialog.closeDialog}
                            />
                        </Show>
                    </Show>
                </Show>
            </div>
        </div>
    );
};
