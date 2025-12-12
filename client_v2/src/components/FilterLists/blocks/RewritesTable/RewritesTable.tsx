import React, { useMemo, useState } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { Table as ReactTable, TableColumn } from 'panel/common/ui/Table';
import { Icon } from 'panel/common/ui/Icon';
import { SortDropdown } from 'panel/common/ui/SortDropdown';
import theme from 'panel/lib/theme';
import { Switch } from 'panel/common/controls/Switch';

import { Rewrite } from '../../DNSRewrites';
import s from '../ListsTable/ListsTable.module.pcss';

const DEFAULT_PAGE_SIZE = 10;

type Props = {
    list: Rewrite[];
    processing: boolean;
    processingAdd: boolean;
    processingDelete: boolean;
    processingUpdate: boolean;
    processingSettings: boolean;
    enabled: boolean;
    addRewritesList: () => void;
    deleteRewrite: (rewrite: Rewrite) => void;
    editRewrite: (rewrite: Rewrite) => void;
    toggleRewrite: (rewrite: Rewrite) => void;
    toggleAllRewrites: (enabled: boolean) => void;
};

export const RewritesTable = ({
    list,
    processing,
    processingAdd,
    processingDelete,
    processingUpdate,
    processingSettings,
    enabled,
    addRewritesList,
    deleteRewrite,
    editRewrite,
    toggleRewrite,
    toggleAllRewrites,
}: Props) => {
    const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

    const pageSize = useMemo(
        () => LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.BLOCKLIST_PAGE_SIZE) || DEFAULT_PAGE_SIZE,
        [],
    );

    const sortedList = useMemo(() => {
        const items = [...(list || [])];

        items.sort((a, b) => {
            const aDomain = a.domain.toLowerCase();
            const bDomain = b.domain.toLowerCase();

            if (aDomain < bDomain) {
                return sortDirection === 'asc' ? -1 : 1;
            }

            if (aDomain > bDomain) {
                return sortDirection === 'asc' ? 1 : -1;
            }

            return 0;
        });

        return items;
    }, [list, sortDirection]);

    const columns: TableColumn<Rewrite>[] = useMemo(
        () => [
            {
                key: 'enabled',
                header: {
                    text: '',
                    className: s.headerCell,
                    render: () => (
                        <Switch
                            id="rewrite_global_enabled"
                            data-testid="rewrite-global-toggle"
                            checked={enabled}
                            onChange={() => toggleAllRewrites(!enabled)}
                            disabled={processingSettings}
                        />
                    ),
                },
                accessor: 'enabled',
                sortable: false,
                fitContent: true,
                render: (value: boolean, row: Rewrite) => {
                    const { domain, enabled } = row;
                    const id = `rewrite_${domain}`;

                    return (
                        <div className={s.cell}>
                            <span className={s.cellLabel}>{intl.getMessage('enabled_table_header')}</span>

                            <div className={s.cellValue}>
                                <Switch
                                    id={id}
                                    data-testid={`rewrite-toggle-${domain}`}
                                    checked={enabled}
                                    onChange={() => toggleRewrite(row)}
                                    disabled={processingUpdate}
                                />
                            </div>
                        </div>
                    );
                },
            },
            {
                key: 'domain',
                header: {
                    text: intl.getMessage('domain'),
                    className: s.headerCell,
                },
                accessor: 'domain',
                sortable: true,
                render: (value: string) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('name_label')}</span>

                        <div className={s.cellValue}>
                            <span className={theme.common.textOverflow}>{value}</span>
                        </div>
                    </div>
                ),
            },
            {
                key: 'answer',
                header: {
                    text: intl.getMessage('result'),
                    className: s.headerCell,
                },
                accessor: 'answer',
                sortable: true,
                render: (value: string) => (
                    <div className={s.cell}>
                        <span className={s.cellLabel}>{intl.getMessage('result')}</span>

                        <div className={s.cellValue}>
                            <span className={theme.common.textOverflow}>{value}</span>
                        </div>
                    </div>
                ),
            },
            {
                key: 'actions',
                header: {
                    text: intl.getMessage('actions_label'),
                    className: s.headerCell,
                },
                sortable: false,
                render: (value: any, row: Rewrite) => {
                    const currentRewrite = {
                        answer: row.answer,
                        domain: row.domain,
                        enabled: row.enabled,
                    };

                    return (
                        <div className={s.cell}>
                            <span className={s.cellLabel}>{intl.getMessage('actions_label')}</span>

                            <div className={s.cellValue}>
                                <div className={s.cellActions}>
                                    <button
                                        type="button"
                                        onClick={() => editRewrite(currentRewrite)}
                                        disabled={processingUpdate}
                                        className={s.action}
                                        data-testid={`edit-rewrite-${row.domain}`}
                                    >
                                        <Icon icon="edit" color="gray" />
                                    </button>

                                    <button
                                        type="button"
                                        onClick={() => deleteRewrite(currentRewrite)}
                                        disabled={processingDelete}
                                        className={s.action}
                                        data-testid={`delete-rewrite-${row.domain}`}
                                    >
                                        <Icon icon="delete" color="red" />
                                    </button>
                                </div>
                            </div>
                        </div>
                    );
                },
            },
        ],
        [processingDelete, processingUpdate, processingSettings, enabled, toggleAllRewrites],
    );

    const handlePageSizeChange = (newSize: number) => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.BLOCKLIST_PAGE_SIZE, newSize);
    };

    const emptyTableContent = () => {
        return (
            <div className={s.emptyTableContent}>
                <Icon icon="not_found_search" color="gray" className={s.emptyTableIcon} />

                <div className={cn(theme.text.t3, s.emptyTableDesc)}>
                    {intl.getMessage('rewrites_empty', {
                        button: (text: string) => (
                            <button className={cn(theme.text.t3, theme.link.link)} type="button" onClick={addRewritesList}>
                                {text}
                            </button>
                        ),
                    })}
                </div>
            </div>
        );
    };

    return (
        <>
            <div className={s.sortDropdownMobile}>
                <SortDropdown value={sortDirection} onChange={setSortDirection} />
            </div>

            <ReactTable<Rewrite>
                data={sortedList}
                className={s.tableRewrites}
                columns={columns}
                emptyTable={emptyTableContent()}
                loading={processing || processingAdd || processingDelete}
                pageSize={pageSize}
                onPageSizeChange={handlePageSizeChange}
            />
        </>
    );
};
