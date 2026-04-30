import React, { useMemo, useState } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Table, TableColumn } from 'panel/common/ui/Table';
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
    onEdit: (lease: DynamicLease) => void;
    onDelete: (lease: DynamicLease) => void;
    onMakeStatic: (lease: DynamicLease) => void;
    onRefresh: () => void;
};

const pageSize = 10;

export const DynamicLeasesTable = ({ leases, onEdit, onDelete, onMakeStatic, onRefresh }: Props) => {
    const [openMenuId, setOpenMenuId] = useState<string | null>(null);

    const handleEdit = (row: DynamicLease) => {
        onEdit(row);
        setOpenMenuId(null);
    };

    const handleMakeStatic = (row: DynamicLease) => {
        onMakeStatic(row);
        setOpenMenuId(null);
    };

    const handleRefresh = () => {
        onRefresh();
        setOpenMenuId(null);
    };

    const handleDelete = (row: DynamicLease) => {
        onDelete(row);
        setOpenMenuId(null);
    };

    const columns: TableColumn<DynamicLease>[] = useMemo(
        () => [
            {
                key: 'mac',
                header: {
                    text: intl.getMessage('dhcp_table_mac_address'),
                    className: s.headerCell,
                },
                accessor: 'mac',
                sortable: true,
                render: (value: string) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('dhcp_table_mac_address')}</span>
                        <div className={s.cellValue}>
                            <span className={theme.common.textOverflow}>{value}</span>
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
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('dhcp_table_ip_address')}</span>
                        <div className={s.cellValue}>
                            <span>{value}</span>
                        </div>
                    </div>
                ),
            },
            {
                key: 'hostname',
                header: {
                    text: intl.getMessage('dhcp_table_hostname_v2'),
                    className: s.headerCell,
                },
                accessor: 'hostname',
                sortable: true,
                render: (value: string) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('dhcp_table_hostname_v2')}</span>
                        <div className={s.cellValue}>
                            <span className={theme.common.textOverflow}>{value}</span>
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
                render: (_value: any, row: DynamicLease) => {
                    const rowId = `${row.mac}-${row.ip}`;
                    return (
                        <div className={s.cell}>
                            <span className={s.cellLabel}>{intl.getMessage('actions_table_header_v2')}</span>
                            <div className={s.cellValue}>
                                <div className={s.cellActions}>
                                    <Dropdown
                                        menu={
                                            <div className={theme.dropdown.menu}>
                                                <div
                                                    className={theme.dropdown.item}
                                                    onClick={() => handleEdit(row)}
                                                >
                                                    {intl.getMessage('edit_table_action_v2')}
                                                </div>
                                                <div
                                                    className={theme.dropdown.item}
                                                    onClick={() => handleMakeStatic(row)}
                                                >
                                                    {intl.getMessage('make_static_v2')}
                                                </div>
                                                <div
                                                    className={theme.dropdown.item}
                                                    onClick={() => handleRefresh()}
                                                >
                                                    {intl.getMessage('refresh_btn_v2')}
                                                </div>
                                                <div
                                                    className={cn(theme.dropdown.item, theme.dropdown.item_danger)}
                                                    onClick={() => handleDelete(row)}
                                                >
                                                    {intl.getMessage('delete_table_action_v2')}
                                                </div>
                                            </div>
                                        }
                                        trigger="click"
                                        position="bottomRight"
                                        noIcon
                                        open={openMenuId === rowId}
                                        onOpenChange={(isOpen) => setOpenMenuId(isOpen ? rowId : null)}
                                    >
                                        <button type="button" className={s.actionButton}>
                                            <Icon icon="bullets" color="gray" />
                                        </button>
                                    </Dropdown>
                                </div>
                            </div>
                        </div>
                    );
                },
            },
        ],
        [onEdit, onDelete],
    );

    return (
        <Table
            data={leases}
            className={s.dynamicTable}
            columns={columns}
            emptyTable={
                <div className={s.emptyTableContent}>
                    <Icon icon="not_found_search" color="gray" className={s.emptyTableIcon} />
                    <div className={cn(theme.text.t3, s.emptyTableDesc)}>
                        {intl.getMessage('dhcp_leases_not_found_v2')}
                    </div>
                </div>
            }
            pageSize={pageSize}
            getRowId={(row: DynamicLease) => `${row.mac}-${row.ip}`}
        />
    );
};
