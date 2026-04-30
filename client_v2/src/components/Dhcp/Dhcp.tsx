import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch, shallowEqual } from 'react-redux';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { RootState } from 'panel/initialState';
import { Button } from 'panel/common/ui/Button';
import { Loader } from 'panel/common/ui/Loader';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { SwitchGroup } from 'panel/common/ui/SettingsGroup';

import {
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
} from 'panel/actions';

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
    const dispatch = useDispatch();
    const dhcp = useSelector((state: RootState) => state.dhcp, shallowEqual);

    const [confirmResetSettings, setConfirmResetSettings] = useState(false);
    const [confirmResetLeases, setConfirmResetLeases] = useState(false);
    const [confirmDeleteLease, setConfirmDeleteLease] = useState<LeaseData | null>(null);
    const [showAllIps, setShowAllIps] = useState(false);

    const {
        processing,
        processingInterfaces,
        processingStatus,
        processingConfig,
        processingDhcp,
        processingAdding,
        processingDeleting,
        processingUpdating,
        enabled,
        interface_name: interfaceName,
        interfaces,
        check,
        v4,
        v6,
        leases,
        staticLeases,
        isModalOpen,
        modalType,
        leaseModalConfig,
        dhcp_available,
    } = dhcp;

    useEffect(() => {
        dispatch(getDhcpStatus());
    }, [dispatch]);

    useEffect(() => {
        if (dhcp_available) {
            dispatch(getDhcpInterfaces());
        }
    }, [dhcp_available, dispatch]);

    const [selectedInterface, setSelectedInterface] = useState(interfaceName || '');

    useEffect(() => {
        if (interfaceName) {
            setSelectedInterface(interfaceName);
        }
    }, [interfaceName]);

    const handleInterfaceChange = (name: string) => {
        setSelectedInterface(name);
        setShowAllIps(false);
    };

    const handleToggleDhcp = () => {
        if (enabled) {
            dispatch(toggleDhcp({ enabled }));
        } else {
            const values: DhcpConfig = {
                enabled,
                interface_name: selectedInterface,
            };
            const enteredSomeV4Value = v4 && Object.values(v4).some(Boolean);
            const enteredSomeV6Value = v6 && Object.values(v6).some(Boolean);
            if (enteredSomeV4Value) {
                values.v4 = v4;
            }
            if (enteredSomeV6Value) {
                values.v6 = v6;
            }
            dispatch(toggleDhcp(values));
        }
    };

    const handleCheckDhcp = () => {
        dispatch(findActiveDhcp(selectedInterface));
    };

    const handleSaveV4Config = (values: V4Config) => {
        dispatch(setDhcpConfig({ interface_name: selectedInterface, v4: values }));
    };

    const handleSaveV6Config = (values: V6Config) => {
        dispatch(setDhcpConfig({ interface_name: selectedInterface, v6: values }));
    };

    const handleResetSettings = () => {
        dispatch(resetDhcp());
        dispatch(getDhcpStatus());
        setConfirmResetSettings(false);
    };

    const handleResetLeases = () => {
        dispatch(resetDhcpLeases());
        setConfirmResetLeases(false);
    };

    const handleAddStaticLease = () => {
        dispatch(toggleLeaseModal({ type: MODAL_TYPE.ADD_LEASE }));
    };

    const handleEditStaticLease = (lease: LeaseData) => {
        dispatch(toggleLeaseModal({ type: MODAL_TYPE.EDIT_LEASE, config: lease }));
    };

    const handleDeleteStaticLease = (lease: LeaseData) => {
        setConfirmDeleteLease(lease);
    };

    const handleConfirmDeleteLease = () => {
        if (confirmDeleteLease) {
            dispatch(removeStaticLease(confirmDeleteLease));
            setConfirmDeleteLease(null);
        }
    };

    const handleMakeStatic = (lease: LeaseData) => {
        dispatch(toggleLeaseModal({ type: MODAL_TYPE.MAKE_STATIC, config: lease }));
    };

    const handleEditDynamicLease = (lease: LeaseData) => {
        dispatch(toggleLeaseModal({ type: MODAL_TYPE.ADD_LEASE, config: lease }));
    };

    const handleDeleteDynamicLease = (lease: LeaseData) => {
        setConfirmDeleteLease(lease);
    };

    const handleRefreshLeases = () => {
        dispatch(getDhcpStatus());
    };

    const handleLeaseModalSubmit = (data: LeaseData) => {
        if (modalType === MODAL_TYPE.EDIT_LEASE) {
            dispatch(updateStaticLease(data));
        } else {
            dispatch(addStaticLease(data));
        }
    };

    const handleLeaseModalClose = () => {
        dispatch(toggleLeaseModal());
    };

    if (processing || processingInterfaces) {
        return (
            <div className={theme.layout.container}>
                <div className={theme.layout.containerIn}>
                    <div className={s.loader}>
                        <Loader />
                    </div>
                </div>
            </div>
        );
    }

    if (!processing && !dhcp_available) {
        return (
            <div className={theme.layout.container}>
                <div className={theme.layout.containerIn}>
                    <div className={s.unavailable}>
                        <h2 className={theme.title.h4}>{intl.getMessage('unavailable_dhcp')}</h2>
                        <p className={theme.text.t2}>{intl.getMessage('unavailable_dhcp_desc')}</p>
                    </div>
                </div>
            </div>
        );
    }

    const enteredSomeValue =
        (v4 && Object.values(v4).some(Boolean)) || (v6 && Object.values(v6).some(Boolean)) || interfaceName;

    const selectedIface = interfaces && interfaces[selectedInterface];
    const allIps: string[] = selectedIface?.ip_addresses || [];
    const visibleIps = showAllIps ? allIps : allIps.slice(0, MAX_VISIBLE_IPS);
    const hiddenIpsCount = allIps.length - MAX_VISIBLE_IPS;

    return (
        <div className={theme.layout.container}>
            <div className={theme.layout.containerIn}>
                <h1 className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                    {intl.getMessage('dhcp_settings')}
                </h1>

                <div className={s.settingsColumn}>
                    <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                        {intl.getMessage('dhcp_title')}
                    </h2>

                    <SwitchGroup
                        id="dhcp_toggle"
                        title={intl.getMessage('dhcp_title')}
                        description={intl.getMessage('dhcp_description')}
                        checked={!!enabled}
                        onChange={handleToggleDhcp}
                        disabled={processingDhcp || processingConfig || (!enabled && !selectedInterface)}
                    >
                        {/* Interface select */}
                        <div className={s.fieldGroup}>
                        <InterfaceSelect
                                interfaces={interfaces}
                                selectedInterface={selectedInterface}
                                enabled={!!enabled}
                            onChange={handleInterfaceChange}
                        />
                    </div>

                    {selectedIface && (
                        <div className={s.interfaceInfo}>
                            {selectedIface.gateway_ip && (
                                <div className={s.interfaceInfoRow}>
                                    <span className={cn(theme.text.t3, s.interfaceInfoLabel)}>
                                        {intl.getMessage('dhcp_form_gateway_input')}:
                                    </span>
                                    <span className={cn(theme.text.t3, s.interfaceInfoValue)}>
                                        {selectedIface.gateway_ip}
                                    </span>
                                </div>
                            )}
                            {selectedIface.hardware_address && (
                                <div className={s.interfaceInfoRow}>
                                    <span className={cn(theme.text.t3, s.interfaceInfoLabel)}>
                                        {intl.getMessage('dhcp_hardware_address')}:
                                    </span>
                                    <span className={cn(theme.text.t3, s.interfaceInfoValue)}>
                                        {selectedIface.hardware_address}
                                    </span>
                                </div>
                            )}
                            {allIps.length > 0 && (
                                <div className={s.interfaceInfoRow}>
                                    <span className={cn(theme.text.t3, s.interfaceInfoLabel)}>
                                        {intl.getMessage('dhcp_ip_addresses')}:
                                    </span>
                                    <span className={cn(theme.text.t3, s.interfaceInfoValue)}>
                                        {visibleIps.join(', ')}
                                    </span>
                                    {!showAllIps && hiddenIpsCount > 0 && (
                                        <span
                                            className={cn(theme.text.t3, s.interfaceInfoMore)}
                                            onClick={() => setShowAllIps(true)}
                                        >
                                            {intl.getMessage('show_more_count', { count: hiddenIpsCount })}
                                        </span>
                                    )}
                                </div>
                            )}
                        </div>
                    )}

                    <div className={s.actionLinks}>
                        <button
                            type="button"
                            className={s.actionLinkGreen}
                            onClick={handleCheckDhcp}
                            disabled={!!enabled || !selectedInterface || processingConfig || processingStatus}
                        >
                            {intl.getMessage('check_dhcp_servers')}
                        </button>
                        <button
                            type="button"
                            className={s.actionLinkOrange}
                            onClick={() => setConfirmResetSettings(true)}
                            disabled={!enteredSomeValue || processingConfig}
                        >
                            {intl.getMessage('reset_settings')}
                        </button>
                    </div>
                    </SwitchGroup>
                </div>

                {!enabled && check && (
                    (check.v4?.other_server?.found === 'yes' || check.v6?.other_server?.found === 'yes') && (
                        <div className={s.warning}>
                            <span className={theme.text.t2}>{intl.getMessage('dhcp_warning')}</span>
                        </div>
                    )
                )}

                <div className={s.settingsColumn}>
                    <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                        {intl.getMessage('dhcp_ipv4_settings_v2')}
                    </h2>
                    <Ipv4Settings
                        v4={v4}
                        interfaces={interfaces}
                        selectedInterface={selectedInterface}
                        processingConfig={!!processingConfig}
                        onSave={handleSaveV4Config}
                    />
                </div>

                <div className={s.settingsColumn}>
                    <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                        {intl.getMessage('dhcp_ipv6_settings')}
                    </h2>
                    <Ipv6Settings
                        v6={v6}
                        interfaces={interfaces}
                        selectedInterface={selectedInterface}
                        processingConfig={!!processingConfig}
                        onSave={handleSaveV6Config}
                    />
                </div>

                <div>
                    <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                        {intl.getMessage('dhcp_static_leases')}
                    </h2>
                    <div className={theme.form.group}>
                        <StaticLeasesTable
                        staticLeases={staticLeases || []}
                        processingDeleting={!!processingDeleting}
                        processingUpdating={!!processingUpdating}
                        onEdit={handleEditStaticLease}
                        onDelete={handleDeleteStaticLease}
                        onRefresh={handleRefreshLeases}
                    />
                    </div>
                    <div className={theme.form.buttonGroup}>
                        <Button
                            variant="primary"
                            size="small"
                            onClick={handleAddStaticLease}
                            className={theme.form.button}
                        >
                            {intl.getMessage('dhcp_add_static_lease')}
                        </Button>
                        <Button
                            variant="secondary"
                            size="small"
                            onClick={() => setConfirmResetLeases(true)}
                            className={theme.form.button}
                        >
                            {intl.getMessage('dhcp_reset_leases')}
                        </Button>
                    </div>
                </div>

                {enabled && (
                    <div>
                        <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                            {intl.getMessage('dhcp_leases')}
                        </h2>
                        <DynamicLeasesTable
                            leases={leases || []}
                            onEdit={handleEditDynamicLease}
                            onDelete={handleDeleteDynamicLease}
                            onMakeStatic={handleMakeStatic}
                            onRefresh={handleRefreshLeases}
                        />
                    </div>
                )}

                {isModalOpen && (
                    <StaticLeaseModal
                        isOpen={!!isModalOpen}
                        isEdit={modalType === MODAL_TYPE.EDIT_LEASE}
                        isMakeStatic={modalType === MODAL_TYPE.MAKE_STATIC}
                        initialData={leaseModalConfig}
                        processingAdding={!!processingAdding}
                        processingUpdating={!!processingUpdating}
                        staticLeases={staticLeases || []}
                        onSubmit={handleLeaseModalSubmit}
                        onClose={handleLeaseModalClose}
                    />
                )}

                {confirmResetSettings && (
                    <ConfirmDialog
                        title={intl.getMessage('reset_settings')}
                        text={intl.getMessage('dhcp_reset')}
                        buttonText={intl.getMessage('reset_settings_confirm')}
                        cancelText={intl.getMessage('cancel')}
                        buttonVariant="danger"
                        onConfirm={handleResetSettings}
                        onClose={() => setConfirmResetSettings(false)}
                    />
                )}

                {confirmResetLeases && (
                    <ConfirmDialog
                        title={intl.getMessage('dhcp_reset_leases')}
                        text={intl.getMessage('dhcp_reset_leases_confirm')}
                        buttonText={intl.getMessage('reset_settings_confirm')}
                        cancelText={intl.getMessage('cancel')}
                        buttonVariant="danger"
                        onConfirm={handleResetLeases}
                        onClose={() => setConfirmResetLeases(false)}
                    />
                )}

                {confirmDeleteLease && (
                    <ConfirmDialog
                        title={intl.getMessage('delete_confirm')}
                        text={intl.getMessage('delete_confirm_desc', { ip: confirmDeleteLease.ip })}
                        buttonText={intl.getMessage('delete_table_action_confirm')}
                        cancelText={intl.getMessage('cancel')}
                        buttonVariant="danger"
                        onConfirm={handleConfirmDeleteLease}
                        onClose={() => setConfirmDeleteLease(null)}
                    />
                )}
            </div>
        </div>
    );
};
