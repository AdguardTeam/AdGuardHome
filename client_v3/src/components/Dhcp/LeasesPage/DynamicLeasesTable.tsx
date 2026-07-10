import { createSignal, createMemo } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Table, type TableColumn } from 'panel/common/ui/Table';
import { Icon } from 'panel/common/ui/Icon';
import { Dropdown } from 'panel/common/ui/Dropdown';

import s from './LeasesTable.module.pcss';

type DynamicLease = {
    mac: string;
    ip: string;
    hostname: string;
    expires?: string;
};

type Props = {
    leases: DynamicLease[];
    processingUpdating: boolean;
    processingDeleting: boolean;
    onEdit: (lease: DynamicLease) => void;
    onDelete: (lease: DynamicLease) => void;
    onMakeStatic: (lease: DynamicLease) => void;
    onRefresh: () => void;
};

export const DynamicLeasesTable = (props: Props) => {
    const [openMenuId, setOpenMenuId] = createSignal<string | null>(null);

    const handleEdit = (row: DynamicLease) => {
        props.onEdit(row);
        setOpenMenuId(null);
    };

    const handleMakeStatic = (row: DynamicLease) => {
        props.onMakeStatic(row);
        setOpenMenuId(null);
    };

    const handleRefresh = () => {
        props.onRefresh();
        setOpenMenuId(null);
    };

    const handleDelete = (row: DynamicLease) => {
        props.onDelete(row);
        setOpenMenuId(null);
    };

    const columns = createMemo<TableColumn<DynamicLease>[]>(() => [
        {
            key: 'mac',
            header: {
                text: intl.getMessage('dhcp_table_mac_address'),
                className: s.headerCell,
            },
            accessor: 'mac',
            sortable: true,
            render: (value: string) => (
                <div class={theme.table.cell}>
                    <span class={theme.table.cellLabel}>
                        {intl.getMessage('dhcp_table_mac_address')}
                    </span>
                    <div class={theme.table.cellValueText}>
                        <span class={theme.common.textOverflow}>{value}</span>
                    </div>
                </div>
            ),
        },
        {
            key: 'ip',
            header: {
                text: intl.getMessage('dhcp_table_ip_address'),
                className: s.headerCell,
            },
            accessor: 'ip',
            sortable: true,
            render: (value: string) => (
                <div class={theme.table.cell}>
                    <span class={theme.table.cellLabel}>
                        {intl.getMessage('dhcp_table_ip_address')}
                    </span>
                    <div class={theme.table.cellValueText}>
                        <span>{value}</span>
                    </div>
                </div>
            ),
        },
        {
            key: 'hostname',
            header: {
                text: intl.getMessage('dhcp_table_hostname'),
                className: s.headerCell,
            },
            accessor: 'hostname',
            sortable: true,
            render: (value: string) => (
                <div class={theme.table.cell}>
                    <span class={theme.table.cellLabel}>
                        {intl.getMessage('dhcp_table_hostname')}
                    </span>
                    <div class={theme.table.cellValueText}>
                        <span class={theme.common.textOverflow}>{value}</span>
                    </div>
                </div>
            ),
        },
        {
            key: 'actions',
            header: {
                text: '',
                className: s.headerCell,
            },
            accessor: 'mac',
            sortable: false,
            width: 48,
            render: (_value: unknown, row: DynamicLease) => {
                const rowId = `${row.mac}-${row.ip}`;
                return (
                    <div class={theme.table.cell}>
                        <div class={theme.table.cellValue}>
                            <div class={cn(theme.table.cellActions, s.mobileActions)}>
                                <button
                                    type="button"
                                    onClick={() => props.onEdit(row)}
                                    disabled={props.processingUpdating}
                                    class={theme.table.action}
                                    title={intl.getMessage('edit_table_action')}
                                    aria-label={intl.getMessage('edit_table_action')}
                                    data-testid="dynamic-lease-edit-button"
                                    data-table-action
                                >
                                    <Icon icon="edit" color="gray" />
                                    <span class={theme.table.actionLabel}>
                                        {intl.getMessage('edit_table_action')}
                                    </span>
                                </button>

                                <button
                                    type="button"
                                    onClick={() => props.onRefresh()}
                                    class={theme.table.action}
                                    title={intl.getMessage('refresh_btn')}
                                    aria-label={intl.getMessage('refresh_btn')}
                                    data-testid="dynamic-lease-refresh-button"
                                    data-table-action
                                >
                                    <Icon icon="refresh" color="gray" />
                                    <span class={theme.table.actionLabel}>
                                        {intl.getMessage('refresh_btn')}
                                    </span>
                                </button>

                                <button
                                    type="button"
                                    onClick={() => props.onMakeStatic(row)}
                                    class={theme.table.action}
                                    title={intl.getMessage('make_static')}
                                    aria-label={intl.getMessage('make_static')}
                                    data-testid="dynamic-lease-make-static-button"
                                    data-table-action
                                >
                                    <Icon icon="plus" color="gray" />
                                    <span class={theme.table.actionLabel}>
                                        {intl.getMessage('make_static')}
                                    </span>
                                </button>

                                <button
                                    type="button"
                                    onClick={() => props.onDelete(row)}
                                    disabled={props.processingDeleting}
                                    class={cn(theme.table.action, theme.table.action_danger)}
                                    title={intl.getMessage('delete_table_action')}
                                    aria-label={intl.getMessage('delete_table_action')}
                                    data-testid="dynamic-lease-delete-button"
                                    data-table-action
                                >
                                    <Icon icon="delete" color="red" />
                                    <span class={theme.table.actionLabel}>
                                        {intl.getMessage('delete_table_action')}
                                    </span>
                                </button>
                            </div>

                            <div class={s.desktopActions}>
                                <Dropdown
                                    menu={
                                        <div class={theme.dropdown.menu}>
                                            <div
                                                class={theme.dropdown.item}
                                                onClick={() => handleEdit(row)}
                                            >
                                                {intl.getMessage('edit_table_action')}
                                            </div>
                                            <div
                                                class={theme.dropdown.item}
                                                onClick={() => handleRefresh()}
                                            >
                                                {intl.getMessage('refresh_btn')}
                                            </div>
                                            <div
                                                class={theme.dropdown.item}
                                                onClick={() => handleMakeStatic(row)}
                                            >
                                                {intl.getMessage('make_static')}
                                            </div>
                                            <div
                                                class={cn(
                                                    theme.dropdown.item,
                                                    theme.dropdown.item_danger,
                                                )}
                                                onClick={() => handleDelete(row)}
                                            >
                                                {intl.getMessage('delete_table_action')}
                                            </div>
                                        </div>
                                    }
                                    trigger="click"
                                    position="bottomRight"
                                    noIcon
                                    open={openMenuId() === rowId}
                                    onOpenChange={(isOpen: boolean) =>
                                        setOpenMenuId(isOpen ? rowId : null)
                                    }
                                >
                                    <button
                                        type="button"
                                        class={cn(theme.table.action, s.actionButton)}
                                        data-testid="dynamic-lease-actions-dropdown"
                                        data-table-action
                                    >
                                        <Icon icon="bullets" color="gray" />
                                    </button>
                                </Dropdown>
                            </div>
                        </div>
                    </div>
                );
            },
        },
    ]);

    return (
        <Table
            data={props.leases}
            class={s.dynamicTable}
            columns={columns()}
            getRowId={(row: DynamicLease) => `${row.mac}-${row.ip}`}
        />
    );
};
