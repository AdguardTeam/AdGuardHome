import { createSignal, Show } from 'solid-js';

import intl from 'panel/common/intl';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { DynamicLeasesTable } from './DynamicLeasesTable';
import { dhcpState, removeStaticLease, toggleLeaseModal, getDhcpStatus } from 'panel/stores/dhcp';

type LeaseData = {
    mac: string;
    ip: string;
    hostname: string;
};

export const DynamicLeasesTab = () => {
    const [confirmDeleteLease, setConfirmDeleteLease] = createSignal<LeaseData | null>(null);

    const handleEditDynamicLease = (lease: LeaseData) => {
        toggleLeaseModal('ADD_LEASE', lease);
    };

    const handleDeleteDynamicLease = (lease: LeaseData) => {
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
        toggleLeaseModal('MAKE_STATIC', lease);
    };

    const handleRefreshLeases = () => {
        getDhcpStatus();
    };

    return (
        <>
            <Show when={dhcpState.leases && dhcpState.leases.length > 0}>
                <DynamicLeasesTable
                    leases={dhcpState.leases || []}
                    processingUpdating={!!dhcpState.processingUpdating}
                    processingDeleting={!!dhcpState.processingDeleting}
                    onEdit={handleEditDynamicLease}
                    onDelete={handleDeleteDynamicLease}
                    onMakeStatic={handleMakeStatic}
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
