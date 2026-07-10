import { createSignal, Show } from 'solid-js';

import intl from 'panel/common/intl';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { PlusButton } from 'panel/common/ui/PlusButton';
import { StaticLeasesTable } from './StaticLeasesTable';
import { dhcpState, removeStaticLease, toggleLeaseModal, getDhcpStatus } from 'panel/stores/dhcp';

import s from './LeasesPage.module.pcss';

type LeaseData = {
    mac: string;
    ip: string;
    hostname: string;
};

export const StaticLeasesTab = () => {
    const [confirmDeleteLease, setConfirmDeleteLease] = createSignal<LeaseData | null>(null);

    const handleAddStaticLease = () => {
        toggleLeaseModal('ADD_LEASE');
    };

    const handleEditStaticLease = (lease: LeaseData) => {
        toggleLeaseModal('EDIT_LEASE', lease);
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

    const handleRefreshLeases = () => {
        getDhcpStatus();
    };

    return (
        <>
            <div class={s.addButton}>
                <PlusButton
                    onClick={handleAddStaticLease}
                    disabled={!dhcpState.enabled || !dhcpState.v4?.range_start}
                >
                    {intl.getMessage('dhcp_add_static_lease')}
                </PlusButton>
            </div>

            <Show when={dhcpState.staticLeases && dhcpState.staticLeases.length > 0}>
                <StaticLeasesTable
                    staticLeases={dhcpState.staticLeases || []}
                    processingDeleting={!!dhcpState.processingDeleting}
                    processingUpdating={!!dhcpState.processingUpdating}
                    onEdit={handleEditStaticLease}
                    onDelete={handleDeleteStaticLease}
                    onRefresh={handleRefreshLeases}
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
        </>
    );
};
