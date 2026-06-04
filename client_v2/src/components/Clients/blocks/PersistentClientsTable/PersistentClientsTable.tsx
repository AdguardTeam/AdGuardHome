import React, { useMemo, useCallback } from 'react';
import cn from 'clsx';
import { useDispatch } from 'react-redux';
import copy from 'copy-to-clipboard';

import intl from 'panel/common/intl';
import { Client, NormalizedTopClients } from 'panel/initialState';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { Table as ReactTable, TableColumn } from 'panel/common/ui/Table';
import { Icon } from 'panel/common/ui/Icon';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { addSuccessToast } from 'panel/actions/toasts';
import theme from 'panel/lib/theme';

import { ServiceCell } from './ServiceCell';
import { TagCell } from './TagCell';
import { WebService } from './ServiceIcons';

import { TableEmptyState } from '../TableEmptyState/TableEmptyState';
import s from './PersistentClientsTable.module.pcss';

type Props = {
    clients: Client[];
    normalizedTopClients?: NormalizedTopClients;
    loading?: boolean;
    onEdit: (client: Client) => void;
    onDelete: (name: string) => void;
    editDisabled?: boolean;
    deleteDisabled?: boolean;
    allServices?: WebService[];
};

export const PersistentClientsTable = ({
    clients,
    normalizedTopClients,
    loading = false,
    onEdit,
    onDelete,
    editDisabled = false,
    deleteDisabled = false,
    allServices = [],
}: Props) => {
    const dispatch = useDispatch();
    const pageSize = useMemo(
        () => LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.CLIENTS_PAGE_SIZE) || undefined,
        [],
    );

    const serviceMap = useMemo(() => {
        const map = new Map<string, WebService>();
        allServices.forEach((svc) => {
            map.set(svc.id, svc);
        });
        return map;
    }, [allServices]);

    const handleCopy = useCallback(
        (text: string) => {
            copy(text);
            dispatch(addSuccessToast(intl.getMessage('copied')));
        },
        [dispatch],
    );

    const columns: TableColumn<Client>[] = useMemo(
        () => [
            {
                key: 'ids',
                header: {
                    text: intl.getMessage('client_identifier'),
                    className: s.headerCell,
                },
                accessor: (row: Client) => row.ids.join(','),
                sortable: true,
                render: (_value: string, row: Client) => {
                    const { ids } = row;
                    const firstId = ids[0] || '';
                    const hiddenCount = ids.length - 1;

                    return (
                        <div className={s.cell}>
                            <span className={s.cellLabel}>
                                {intl.getMessage('client_identifier')}
                            </span>

                            <div className={s.cellValue}>
                                <div className={s.idsRow}>
                                    <span className={cn(theme.common.textOverflow, s.idsText)}>
                                        {firstId}
                                        {hiddenCount > 0 && ','}
                                    </span>
                                    {hiddenCount > 0 && (
                                        <Dropdown
                                            trigger="hover"
                                            noIcon
                                            overlayClassName={s.idsTooltipOverlay}
                                            menu={
                                                <div className={s.idsTooltip}>
                                                    {ids.map((id) => (
                                                        <span key={id} className={s.idsTooltipItem}>
                                                            {id}
                                                        </span>
                                                    ))}
                                                    <button
                                                        type="button"
                                                        className={cn(
                                                            s.copyBtn,
                                                            s.copyBtnGreen,
                                                            s.copyBtnTopRight,
                                                        )}
                                                        onClick={() => handleCopy(ids.join(', '))}
                                                        title={intl.getMessage('copy')}
                                                    >
                                                        <Icon icon="copy" color="green" />
                                                    </button>
                                                </div>
                                            }
                                        >
                                            <span className={s.countLabel}>{hiddenCount}</span>
                                        </Dropdown>
                                    )}
                                </div>
                            </div>
                        </div>
                    );
                },
            },
            {
                key: 'name',
                header: {
                    text: intl.getMessage('name'),
                    className: s.headerCell,
                },
                accessor: 'name',
                sortable: true,
                render: (value: string) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('name')}</span>

                        <div className={s.cellValue}>
                            <Dropdown
                                trigger="hover"
                                noIcon
                                overlayClassName={s.nameTooltipOverlay}
                                menu={
                                    <div className={s.nameTooltip}>
                                        <span className={s.nameTooltipText}>{value}</span>
                                        <button
                                            type="button"
                                            className={cn(s.copyBtn, s.copyBtnGreen)}
                                            onClick={() => handleCopy(value)}
                                            title={intl.getMessage('copy')}
                                        >
                                            <Icon icon="copy" color="green" />
                                        </button>
                                    </div>
                                }
                                className={s.nameDropdown}
                                childrenClassName={s.nameDropdownInner}
                            >
                                <span className={cn(theme.common.textOverflow, s.nameTrigger)}>
                                    {value}
                                </span>
                            </Dropdown>
                        </div>
                    </div>
                ),
            },
            {
                key: 'settings',
                header: {
                    text: intl.getMessage('settings'),
                    className: s.headerCell,
                },
                accessor: 'use_global_settings',
                sortable: true,
                render: (value: boolean) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('settings')}</span>

                        <div className={s.cellValue}>
                            <span>
                                {value
                                    ? intl.getMessage('settings_global')
                                    : intl.getMessage('settings_custom')}
                            </span>
                            {!value && (
                                <Icon icon="user" color="gray" className={s.userIconRight} />
                            )}
                        </div>
                    </div>
                ),
            },
            {
                key: 'blocked_services',
                header: {
                    text: intl.getMessage('blocked_services'),
                    className: s.headerCell,
                },
                accessor: 'use_global_blocked_services',
                sortable: true,
                minWidth: 120,
                render: (_value: boolean, row: Client) => (
                    <ServiceCell
                        serviceIds={row.blocked_services || []}
                        useGlobal={row.use_global_blocked_services}
                        serviceMap={serviceMap}
                    />
                ),
            },
            {
                key: 'upstreams',
                header: {
                    text: intl.getMessage('upstreams'),
                    className: s.headerCell,
                },
                accessor: (row: Client) => row.upstreams.length > 0,
                sortable: true,
                render: (_value: boolean, row: Client) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('upstreams')}</span>

                        <div className={s.cellValue}>
                            <span>
                                {row.upstreams.length > 0
                                    ? intl.getMessage('settings_custom')
                                    : intl.getMessage('settings_global')}
                            </span>
                        </div>
                    </div>
                ),
            },
            {
                key: 'tags',
                header: {
                    text: intl.getMessage('tags_title'),
                    className: s.headerCell,
                },
                accessor: 'tags',
                sortable: false,
                render: (value: string[]) => <TagCell tags={value} onCopy={handleCopy} />,
            },
            {
                key: 'requests',
                header: {
                    text: intl.getMessage('requests_table_header'),
                    className: s.headerCell,
                },
                accessor: (row: Client) => normalizedTopClients?.configured[row.name] || 0,
                sortable: true,
                render: (_value: unknown, row: Client) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>
                            {intl.getMessage('requests_table_header')}
                        </span>

                        <div className={s.cellValue}>
                            <span>
                                {(normalizedTopClients?.configured[row.name] || 0).toLocaleString()}
                            </span>
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
                sortable: false,
                width: 80,
                render: (_value: unknown, row: Client) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>
                            {intl.getMessage('actions_table_header')}
                        </span>

                        <div className={s.cellValue}>
                            <div className={s.cellActions}>
                                <button
                                    type="button"
                                    onClick={() => onEdit(row)}
                                    disabled={editDisabled}
                                    className={s.action}
                                    title={intl.getMessage('edit_table_action')}
                                    data-testid="clients-edit-button"
                                >
                                    <Icon icon="edit" color="gray" />
                                </button>

                                <button
                                    type="button"
                                    onClick={() => onDelete(row.name)}
                                    disabled={deleteDisabled}
                                    className={cn(s.action, s.action_danger)}
                                    title={intl.getMessage('delete_table_action')}
                                    data-testid="clients-delete-button"
                                >
                                    <Icon icon="delete" color="red" />
                                </button>
                            </div>
                        </div>
                    </div>
                ),
            },
        ],
        [
            deleteDisabled,
            editDisabled,
            normalizedTopClients,
            onDelete,
            onEdit,
            serviceMap,
            handleCopy,
        ],
    );

    const handlePageSizeChange = (newSize: number) => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.CLIENTS_PAGE_SIZE, newSize);
    };

    const emptyTableContent = <TableEmptyState />;

    return (
        <ReactTable<Client>
            data={clients}
            className={s.table}
            columns={columns}
            emptyTable={emptyTableContent}
            loading={loading}
            pageSize={pageSize}
            onPageSizeChange={handlePageSizeChange}
            getRowId={(row) => row.name}
        />
    );
};
