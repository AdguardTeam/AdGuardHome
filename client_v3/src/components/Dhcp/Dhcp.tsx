import { createSignal, createEffect, onMount, Show } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Button } from 'panel/common/ui/Button';
import { PageLoader } from 'panel/common/ui/Loader';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { SwitchGroup } from 'panel/common/ui/SettingsGroup';

import {
    dhcpState,
    getDhcpStatus,
    getDhcpInterfaces,
    findActiveDhcp,
    setDhcpConfig,
    toggleDhcp,
    resetDhcp,
    resetDhcpLeases,
    addStaticLease,
    removeStaticLease,
    updateStaticLease,
    toggleLeaseModal,
} from 'panel/stores/dhcp';

import { InterfaceSelect } from './blocks/InterfaceSelect';
import { Ipv4Settings } from './blocks/Ipv4Settings';
import { Ipv6Settings } from './blocks/Ipv6Settings';
import { StaticLeasesTable } from './blocks/StaticLeasesTable';
import { DynamicLeasesTable } from './blocks/DynamicLeasesTable';
import { StaticLeaseModal } from './blocks/StaticLeaseModal';

import s from './Dhcp.module.pcss';

type LeaseData = {
    mac: string;
    ip: string;
    hostname: string;
};

type V4Config = {
    gateway_ip: string;
    subnet_mask: string;
    range_start: string;
    range_end: string;
    lease_duration: number;
};

type V6Config = {
    range_start: string;
    lease_duration: number;
};

type DhcpConfig = {
    enabled: boolean;
    interface_name: string;
    v4?: V4Config;
    v6?: V6Config;
};

const MODAL_TYPE = {
    ADD_LEASE: 'ADD_LEASE',
    EDIT_LEASE: 'EDIT_LEASE',
    MAKE_STATIC: 'MAKE_STATIC',
};

const MAX_VISIBLE_IPS = 2;

export const Dhcp = () => {
    const [confirmResetSettings, setConfirmResetSettings] = createSignal(false);
    const [confirmResetLeases, setConfirmResetLeases] = createSignal(false);
    const [confirmDeleteLease, setConfirmDeleteLease] = createSignal<LeaseData | null>(null);
    const [showAllIps, setShowAllIps] = createSignal(false);
    const [selectedInterface, setSelectedInterface] = createSignal(dhcpState.interface_name || '');

    // Sync selectedInterface with store
    createEffect(() => {
        if (dhcpState.interface_name) {
            setSelectedInterface(dhcpState.interface_name);
        }
    });

    // Load data on mount
    onMount(async () => {
        await getDhcpStatus();
        if (dhcpState.dhcp_available) {
            await getDhcpInterfaces();
        }
    });

    const handleInterfaceChange = (name: string) => {
        setSelectedInterface(name);
        setShowAllIps(false);
    };

    const handleToggleDhcp = () => {
        if (dhcpState.enabled) {
            toggleDhcp({ enabled: dhcpState.enabled });
        } else {
            const values: DhcpConfig = {
                enabled: dhcpState.enabled,
                interface_name: selectedInterface(),
            };
            const v4 = dhcpState.v4;
            const v6 = dhcpState.v6;
            const enteredSomeV4Value = v4 && Object.values(v4).some(Boolean);
            const enteredSomeV6Value = v6 && Object.values(v6).some(Boolean);
            if (enteredSomeV4Value) {
                values.v4 = v4;
            }
            if (enteredSomeV6Value) {
                values.v6 = v6;
            }
            toggleDhcp(values);
        }
    };

    const handleCheckDhcp = () => {
        findActiveDhcp(selectedInterface());
    };

    const handleSaveV4Config = (values: V4Config) => {
        setDhcpConfig({ interface_name: selectedInterface(), v4: values });
    };

    const handleSaveV6Config = (values: V6Config) => {
        setDhcpConfig({ interface_name: selectedInterface(), v6: values });
    };

    const handleResetSettings = () => {
        resetDhcp();
        getDhcpStatus();
        setConfirmResetSettings(false);
    };

    const handleResetLeases = () => {
        resetDhcpLeases();
        setConfirmResetLeases(false);
    };

    const handleAddStaticLease = () => {
        toggleLeaseModal(MODAL_TYPE.ADD_LEASE);
    };

    const handleEditStaticLease = (lease: LeaseData) => {
        toggleLeaseModal(MODAL_TYPE.EDIT_LEASE, lease);
    };

    const handleDeleteStaticLease = (lease: LeaseData) => {
        setConfirmDeleteLease(lease);
    };

    const handleConfirmDeleteLease = () => {
        const lease = confirmDeleteLease();
        if (lease) {
            removeStaticLease(lease);
            setConfirmDeleteLease(null);
        }
    };

    const handleMakeStatic = (lease: LeaseData) => {
        toggleLeaseModal(MODAL_TYPE.MAKE_STATIC, lease);
    };

    const handleEditDynamicLease = (lease: LeaseData) => {
        toggleLeaseModal(MODAL_TYPE.ADD_LEASE, lease);
    };

    const handleDeleteDynamicLease = (lease: LeaseData) => {
        setConfirmDeleteLease(lease);
    };

    const handleRefreshLeases = () => {
        getDhcpStatus();
    };

    const handleLeaseModalSubmit = (data: LeaseData) => {
        if (dhcpState.modalType === MODAL_TYPE.EDIT_LEASE) {
            updateStaticLease(data);
        } else {
            addStaticLease(data);
        }
    };

    const handleLeaseModalClose = () => {
        toggleLeaseModal();
    };

    const selectedIface = () => {
        const interfaces = dhcpState.interfaces;
        const sel = selectedInterface();
        return interfaces && sel ? interfaces[sel] : null;
    };

    const allIps = () => selectedIface()?.ip_addresses || [];
    const visibleIps = () => (showAllIps() ? allIps() : allIps().slice(0, MAX_VISIBLE_IPS));
    const hiddenIpsCount = () => allIps().length - MAX_VISIBLE_IPS;

    const enteredSomeValue = () => {
        const v4 = dhcpState.v4;
        const v6 = dhcpState.v6;
        return (
            (v4 && Object.values(v4).some(Boolean)) ||
            (v6 && Object.values(v6).some(Boolean)) ||
            selectedInterface()
        );
    };

    const showWarning = () => {
        const check = dhcpState.check;
        return (
            !dhcpState.enabled &&
            check &&
            (check.v4?.other_server?.found === 'yes' || check.v6?.other_server?.found === 'yes')
        );
    };

    return (
        <>
            <Show when={dhcpState.processing || dhcpState.processingInterfaces}>
                <PageLoader />
            </Show>

            <Show when={!dhcpState.processing && !dhcpState.dhcp_available}>
                <div class={theme.layout.container}>
                    <div class={theme.layout.containerIn}>
                        <div class={s.unavailable}>
                            <h2 class={theme.title.h4}>{intl.getMessage('unavailable_dhcp')}</h2>
                            <p class={theme.text.t2}>{intl.getMessage('unavailable_dhcp_desc')}</p>
                        </div>
                    </div>
                </div>
            </Show>

            <Show when={!dhcpState.processing && dhcpState.dhcp_available}>
                <div class={theme.layout.container}>
                    <div class={theme.layout.containerIn}>
                        <h1
                            class={cn(
                                theme.layout.title,
                                theme.title.h4,
                                theme.title.h3_tablet,
                                s.title,
                            )}
                        >
                            {intl.getMessage('dhcp_settings')}
                        </h1>

                        <div class={s.settingsColumn}>
                            <SwitchGroup
                                id="dhcp_toggle"
                                title={intl.getMessage('dhcp_title')}
                                description={intl.getMessage('dhcp_description')}
                                checked={!!dhcpState.enabled}
                                onChange={handleToggleDhcp}
                                disabled={
                                    dhcpState.processingDhcp ||
                                    dhcpState.processingConfig ||
                                    (!dhcpState.enabled && !selectedInterface())
                                }
                            >
                                <div class={s.fieldGroup}>
                                    <InterfaceSelect
                                        interfaces={dhcpState.interfaces}
                                        selectedInterface={selectedInterface()}
                                        enabled={!!dhcpState.enabled}
                                        onChange={handleInterfaceChange}
                                    />
                                </div>

                                <Show when={selectedIface()}>
                                    <div class={s.interfaceInfo}>
                                        <Show when={selectedIface()?.gateway_ip}>
                                            <div class={s.interfaceInfoRow}>
                                                <span
                                                    class={cn(theme.text.t3, s.interfaceInfoLabel)}
                                                >
                                                    {intl.getMessage('dhcp_form_gateway_input')}:
                                                </span>
                                                <span
                                                    class={cn(theme.text.t3, s.interfaceInfoValue)}
                                                >
                                                    {selectedIface()?.gateway_ip}
                                                </span>
                                            </div>
                                        </Show>
                                        <Show when={selectedIface()?.hardware_address}>
                                            <div class={s.interfaceInfoRow}>
                                                <span
                                                    class={cn(theme.text.t3, s.interfaceInfoLabel)}
                                                >
                                                    {intl.getMessage('dhcp_hardware_address')}:
                                                </span>
                                                <span
                                                    class={cn(theme.text.t3, s.interfaceInfoValue)}
                                                >
                                                    {selectedIface()?.hardware_address}
                                                </span>
                                            </div>
                                        </Show>
                                        <Show when={allIps().length > 0}>
                                            <div class={s.interfaceInfoRow}>
                                                <span
                                                    class={cn(theme.text.t3, s.interfaceInfoLabel)}
                                                >
                                                    {intl.getMessage('dhcp_ip_addresses')}:
                                                </span>
                                                <span
                                                    class={cn(theme.text.t3, s.interfaceInfoValue)}
                                                >
                                                    {visibleIps().join(', ')}
                                                </span>
                                                <Show when={!showAllIps() && hiddenIpsCount() > 0}>
                                                    <span
                                                        class={cn(
                                                            theme.text.t3,
                                                            s.interfaceInfoMore,
                                                        )}
                                                        onClick={() => setShowAllIps(true)}
                                                    >
                                                        {intl.getMessage('show_more_count', {
                                                            count: hiddenIpsCount(),
                                                        })}
                                                    </span>
                                                </Show>
                                            </div>
                                        </Show>
                                    </div>
                                </Show>

                                <div class={s.actionLinks}>
                                    <button
                                        type="button"
                                        class={s.actionLinkGreen}
                                        onClick={handleCheckDhcp}
                                        disabled={
                                            !!dhcpState.enabled ||
                                            !selectedInterface() ||
                                            dhcpState.processingConfig ||
                                            dhcpState.processingStatus
                                        }
                                    >
                                        {intl.getMessage('check_dhcp_servers')}
                                    </button>
                                    <button
                                        type="button"
                                        class={s.actionLinkOrange}
                                        onClick={() => setConfirmResetSettings(true)}
                                        disabled={!enteredSomeValue() || dhcpState.processingConfig}
                                    >
                                        {intl.getMessage('reset_settings')}
                                    </button>
                                </div>
                            </SwitchGroup>
                        </div>

                        <Show when={showWarning()}>
                            <div class={s.warning}>
                                <span class={theme.text.t2}>{intl.getMessage('dhcp_warning')}</span>
                            </div>
                        </Show>

                        <div class={s.settingsColumn}>
                            <h2
                                class={cn(
                                    theme.layout.subtitle,
                                    theme.title.h5,
                                    theme.title.h4_tablet,
                                )}
                            >
                                {intl.getMessage('dhcp_ipv4_settings')}
                            </h2>
                            <Ipv4Settings
                                v4={dhcpState.v4}
                                interfaces={dhcpState.interfaces}
                                selectedInterface={selectedInterface()}
                                processingConfig={!!dhcpState.processingConfig}
                                onSave={handleSaveV4Config}
                            />
                        </div>

                        <div class={s.settingsColumn}>
                            <h2
                                class={cn(
                                    theme.layout.subtitle,
                                    theme.title.h5,
                                    theme.title.h4_tablet,
                                )}
                            >
                                {intl.getMessage('dhcp_ipv6_settings')}
                            </h2>
                            <Ipv6Settings
                                v6={dhcpState.v6}
                                interfaces={dhcpState.interfaces}
                                selectedInterface={selectedInterface()}
                                processingConfig={!!dhcpState.processingConfig}
                                onSave={handleSaveV6Config}
                            />
                        </div>

                        <div>
                            <h2
                                class={cn(
                                    theme.layout.subtitle,
                                    theme.title.h5,
                                    theme.title.h4_tablet,
                                )}
                            >
                                {intl.getMessage('dhcp_static_leases')}
                            </h2>
                            <div class={theme.form.group}>
                                <Show
                                    when={
                                        dhcpState.staticLeases && dhcpState.staticLeases.length > 0
                                    }
                                    fallback={
                                        <div class={cn(theme.text.t1, s.emptyTable)}>
                                            {intl.getMessage('static_dhcp_leases_not_found')}
                                        </div>
                                    }
                                >
                                    <StaticLeasesTable
                                        staticLeases={dhcpState.staticLeases || []}
                                        processingDeleting={!!dhcpState.processingDeleting}
                                        processingUpdating={!!dhcpState.processingUpdating}
                                        onEdit={handleEditStaticLease}
                                        onDelete={handleDeleteStaticLease}
                                        onRefresh={handleRefreshLeases}
                                    />
                                </Show>
                            </div>
                            <div class={theme.form.buttonGroup}>
                                <Button
                                    variant="primary"
                                    size="small"
                                    onClick={handleAddStaticLease}
                                    class={theme.form.button}
                                    disabled={!selectedInterface()}
                                >
                                    {intl.getMessage('dhcp_add_static_lease')}
                                </Button>
                                <Button
                                    variant="secondary"
                                    size="small"
                                    onClick={() => setConfirmResetLeases(true)}
                                    class={theme.form.button}
                                    disabled={
                                        !selectedInterface() ||
                                        !dhcpState.staticLeases ||
                                        dhcpState.staticLeases.length === 0
                                    }
                                >
                                    {intl.getMessage('dhcp_reset_leases')}
                                </Button>
                            </div>
                        </div>

                        <h2
                            class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}
                        >
                            {intl.getMessage('dhcp_leases')}
                        </h2>

                        <div class={theme.form.group}>
                            <Show
                                when={dhcpState.leases && dhcpState.leases.length > 0}
                                fallback={
                                    <div class={cn(theme.text.t1, s.emptyTable)}>
                                        {intl.getMessage('dynamic_dhcp_leases_not_found')}
                                    </div>
                                }
                            >
                                <DynamicLeasesTable
                                    leases={dhcpState.leases || []}
                                    onEdit={handleEditDynamicLease}
                                    onDelete={handleDeleteDynamicLease}
                                    onMakeStatic={handleMakeStatic}
                                    onRefresh={handleRefreshLeases}
                                />
                            </Show>
                        </div>

                        <Show when={dhcpState.isModalOpen}>
                            <StaticLeaseModal
                                isOpen={!!dhcpState.isModalOpen}
                                isEdit={dhcpState.modalType === MODAL_TYPE.EDIT_LEASE}
                                isMakeStatic={dhcpState.modalType === MODAL_TYPE.MAKE_STATIC}
                                initialData={dhcpState.leaseModalConfig}
                                processingAdding={!!dhcpState.processingAdding}
                                processingUpdating={!!dhcpState.processingUpdating}
                                staticLeases={dhcpState.staticLeases || []}
                                dhcpConfig={
                                    dhcpState.v4
                                        ? {
                                              gatewayIp: dhcpState.v4.gateway_ip,
                                              subnetMask: dhcpState.v4.subnet_mask,
                                          }
                                        : undefined
                                }
                                onSubmit={handleLeaseModalSubmit}
                                onClose={handleLeaseModalClose}
                            />
                        </Show>

                        <Show when={confirmResetSettings()}>
                            <ConfirmDialog
                                title={intl.getMessage('reset_settings')}
                                text={intl.getMessage('dhcp_reset')}
                                buttonText={intl.getMessage('reset_settings_confirm')}
                                cancelText={intl.getMessage('cancel')}
                                buttonVariant="danger"
                                onConfirm={handleResetSettings}
                                onClose={() => setConfirmResetSettings(false)}
                            />
                        </Show>

                        <Show when={confirmResetLeases()}>
                            <ConfirmDialog
                                title={intl.getMessage('dhcp_reset_leases')}
                                text={intl.getMessage('dhcp_reset_leases_confirm')}
                                buttonText={intl.getMessage('reset_settings_confirm')}
                                cancelText={intl.getMessage('cancel')}
                                buttonVariant="danger"
                                onConfirm={handleResetLeases}
                                onClose={() => setConfirmResetLeases(false)}
                            />
                        </Show>

                        <Show when={confirmDeleteLease()}>
                            <ConfirmDialog
                                title={intl.getMessage('delete_confirm')}
                                text={intl.getMessage('delete_confirm_desc', {
                                    ip: confirmDeleteLease()?.ip,
                                })}
                                buttonText={intl.getMessage('delete_table_action_confirm')}
                                cancelText={intl.getMessage('cancel')}
                                buttonVariant="danger"
                                onConfirm={handleConfirmDeleteLease}
                                onClose={() => setConfirmDeleteLease(null)}
                            />
                        </Show>
                    </div>
                </div>
            </Show>
        </>
    );
};
