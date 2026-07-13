import { createMemo, createSignal, onMount, Show } from 'solid-js';
import { useSearchParams } from '@solidjs/router';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Breadcrumbs } from 'panel/common/ui/Breadcrumbs';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Icon } from 'panel/common/ui/Icon';
import { Tabs } from 'panel/common/ui/Tabs';
import { RoutePath } from 'panel/components/Routes/Paths';
import {
    dhcpState,
    addStaticLease,
    updateStaticLease,
    resetDhcpLeases,
    toggleLeaseModal,
    getDhcpStatus,
} from 'panel/stores/dhcp';

import { StaticLeasesTab } from './StaticLeasesTab';
import { DynamicLeasesTab } from './DynamicLeasesTab';
import { StaticLeaseModal } from './StaticLeaseModal';

import s from './LeasesPage.module.pcss';

const LEASE_TABS = {
    STATIC: 'static',
    DYNAMIC: 'dynamic',
} as const;

type LeaseData = {
    mac: string;
    ip: string;
    hostname: string;
};

export const LeasesPage = () => {
    const [searchParams, setSearchParams] = useSearchParams<{
        tab?: string;
    }>();

    const [menuOpen, setMenuOpen] = createSignal(false);
    const [confirmResetLeases, setConfirmResetLeases] = createSignal(false);

    const activeTab = createMemo(() =>
        searchParams.tab === LEASE_TABS.DYNAMIC ? LEASE_TABS.DYNAMIC : LEASE_TABS.STATIC,
    );

    const handleTabChange = (tabId: string) => {
        setSearchParams({ tab: tabId }, { replace: true });
    };

    onMount(() => {
        getDhcpStatus();
    });

    const handleLeaseModalSubmit = (data: LeaseData) => {
        if (dhcpState.modalType === 'EDIT_LEASE') {
            updateStaticLease(data);
        } else {
            addStaticLease(data);
        }
    };

    const handleLeaseModalClose = () => {
        toggleLeaseModal();
    };

    const handleResetClick = () => {
        setMenuOpen(false);
        setConfirmResetLeases(true);
    };

    const handleResetLeases = () => {
        resetDhcpLeases();
        setConfirmResetLeases(false);
    };

    const resetMenu = (
        <div
            class={cn(theme.dropdown.item, theme.dropdown.item_danger, theme.dropdown.item_large)}
            onClick={handleResetClick}
        >
            {intl.getMessage('dhcp_reset_leases')}
        </div>
    );

    return (
        <div class={cn(theme.layout.container, theme.layout.container_compact)}>
            <div class={theme.layout.containerIn}>
                <div class={s.breadcrumbs}>
                    <Breadcrumbs
                        parentLinks={[
                            {
                                path: RoutePath.Dhcp,
                                title: intl.getMessage('dhcp'),
                            },
                        ]}
                        currentTitle={intl.getMessage('dhcp_leases_title')}
                    />
                </div>

                <div class={s.header}>
                    <h1 class={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                        {intl.getMessage('dhcp_leases_title')}
                    </h1>
                    <div class={s.headerActions}>
                        <Dropdown
                            trigger="click"
                            position="bottomRight"
                            noIcon
                            open={menuOpen()}
                            onOpenChange={setMenuOpen}
                            menu={resetMenu}
                        >
                            <button
                                type="button"
                                class={s.menuButton}
                                aria-label={intl.getMessage('dhcp_reset_leases')}
                            >
                                <Icon icon="bullets" />
                            </button>
                        </Dropdown>
                    </div>
                </div>

                <Tabs
                    activeTab={activeTab()}
                    onTabChange={handleTabChange}
                    variant="filled"
                    class={s.tabs}
                    contentClass={cn(s.tabContent, {
                        [s.tabContent_static]: activeTab() === LEASE_TABS.STATIC,
                    })}
                    fullWidth
                    tabs={[
                        {
                            id: LEASE_TABS.STATIC,
                            label: intl.getMessage('dhcp_static_leases'),
                            content: <StaticLeasesTab />,
                        },
                        {
                            id: LEASE_TABS.DYNAMIC,
                            label: intl.getMessage('dhcp_leases'),
                            content: <DynamicLeasesTab />,
                        },
                    ]}
                />

                <Show when={dhcpState.isModalOpen}>
                    <StaticLeaseModal
                        isOpen={!!dhcpState.isModalOpen}
                        isEdit={dhcpState.modalType === 'EDIT_LEASE'}
                        isMakeStatic={dhcpState.modalType === 'MAKE_STATIC'}
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
            </div>
        </div>
    );
};
