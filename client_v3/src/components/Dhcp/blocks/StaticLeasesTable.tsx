import { createSignal, createMemo } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Table, type TableColumn } from 'panel/common/ui/Table';
import { Icon } from 'panel/common/ui/Icon';
import { Dropdown } from 'panel/common/ui/Dropdown';

import s from './LeasesTable.module.pcss';

type StaticLease = {
    mac: string;
    ip: string;
    hostname: string;
};

type Props = {
    staticLeases: StaticLease[];
    processingDeleting: boolean;
    processingUpdating: boolean;
    onEdit: (lease: StaticLease) => void;
    onDelete: (lease: StaticLease) => void;
    onRefresh: () => void;
};

const pageSize = 7;

export const StaticLeasesTable = (props: Props) => {
    const [openMenuId, setOpenMenuId] = createSignal<string | null>(null);

    const handleEdit = (row: StaticLease) => {
        props.onEdit(row);
        setOpenMenuId(null);
    };

    const handleRefresh = () => {
        props.onRefresh();
        setOpenMenuId(null);
    };

    const handleDelete = (row: StaticLease) => {
        props.onDelete(row);
        setOpenMenuId(null);
    };

    const columns = createMemo<TableColumn<StaticLease>[]>(() => [
        {
            key: 'mac',
            header: {
                text: intl.getMessage('dhcp_table_mac_address'),
                className: s.headerCell,
            },
            accessor: 'mac',
            sortable: true,
            render: (value: string) => (
                <div class={s.cell}>
                    <span class={s.cellLabel}>
                        {intl.getMessage('dhcp_table_mac_address')}
                    </span>
                    <div class={s.cellValue}>
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
                <div class={s.cell}>
                    <span class={s.cellLabel}>
                        {intl.getMessage('dhcp_table_ip_address')}
                    </span>
                    <div class={s.cellValue}>
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
                <div class={s.cell}>
                    <span class={s.cellLabel}>
                        {intl.getMessage('dhcp_table_hostname')}
                    </span>
                    <div class={s.cellValue}>
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
            fitContent: true,
            render: (_value: any, row: StaticLease) => {
                const rowId = `${row.mac}-${row.ip}`;
                return (
                    <div class={s.cell}>
                        <span class={s.cellLabel}>
                            {intl.getMessage('actions_table_header')}
                        </span>
                        <div class={s.cellValue}>
                            <div class={s.cellActions}>
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
                                    <button type="button" class={s.actionButton}>
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
            data={props.staticLeases}
            class={s.staticTable}
            columns={columns()}
            emptyTable={
                <div class={s.emptyTableContent}>
                    <Icon icon="not_found_search" color="gray" class={s.emptyTableIcon} />
                    <div class={cn(theme.text.t3, s.emptyTableDesc)}>
                        {intl.getMessage('dhcp_static_leases_not_found')}
                    </div>
                </div>
            }
            pageSize={pageSize}
            getRowId={(row: StaticLease) => `${row.mac}-${row.ip}`}
        />
    );
};
