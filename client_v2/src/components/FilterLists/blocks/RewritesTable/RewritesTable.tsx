import { createSignal, createMemo } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { Table, type TableColumn } from 'panel/common/ui/Table';
import { Icon } from 'panel/common/ui/Icon';
import { SortSelect } from 'panel/common/ui/SortSelect';
import theme from 'panel/lib/theme';
import { Switch } from 'panel/common/controls/Switch';

import type { Rewrite } from '../../DNSRewrites';
import s from '../ListsTable/ListsTable.module.pcss';

type Props = {
    list: Rewrite[];
    processing: boolean;
    processingAdd: boolean;
    processingDelete: boolean;
    processingUpdate: boolean;
    addRewritesList: () => void;
    deleteRewrite: (rewrite: Rewrite) => void;
    editRewrite: (rewrite: Rewrite) => void;
    toggleRewrite: (rewrite: Rewrite) => void;
};

export const RewritesTable = (props: Props) => {
    const [sortDirection, setSortDirection] = createSignal<'asc' | 'desc'>('asc');

    const pageSize = createMemo(
        () => LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.BLOCKLIST_PAGE_SIZE) || undefined,
    );

    const sortedList = createMemo(() => {
        const items = [...(props.list || [])];

        const direction = sortDirection();
        items.sort((a, b) => {
            const aDomain = a.domain.toLowerCase();
            const bDomain = b.domain.toLowerCase();

            if (aDomain < bDomain) {
                return direction === 'asc' ? -1 : 1;
            }

            if (aDomain > bDomain) {
                return direction === 'asc' ? 1 : -1;
            }

            return 0;
        });

        return items;
    });

    const handleEdit = (rewrite: Rewrite) => {
        props.editRewrite(rewrite);
    };

    const handleDelete = (rewrite: Rewrite) => {
        props.deleteRewrite(rewrite);
    };

    const columns = createMemo<TableColumn<Rewrite>[]>(() => [
        {
            key: 'domain',
            header: {
                text: intl.getMessage('domain'),
                className: s.headerCell,
            },
            accessor: 'domain',
            sortable: true,
            render: (value: string, row: Rewrite) => {
                const { domain, enabled } = row;
                const id = `rewrite_${domain}`;

                return (
                    <div class={theme.table.cell}>
                        <div class={cn(theme.table.cellValueText, s.domainCellValue)}>
                            <span class={theme.common.textOverflow}>{value}</span>
                            <Switch
                                id={id}
                                data-testid={`rewrite-toggle-${domain}`}
                                checked={enabled}
                                onChange={() => props.toggleRewrite(row)}
                                disabled={props.processingUpdate}
                            />
                        </div>
                    </div>
                );
            },
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
                <div class={theme.table.cell}>
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
            sortable: false,
            width: 80,
            render: (value: any, row: Rewrite) => {
                const currentRewrite = {
                    answer: row.answer,
                    domain: row.domain,
                    enabled: row.enabled,
                };

                return (
                    <div class={theme.table.cell}>
                        <div class={theme.table.cellValue}>
                            <div class={theme.table.cellActions}>
                                <button
                                    type="button"
                                    onClick={() => handleEdit(currentRewrite)}
                                    disabled={props.processingUpdate}
                                    class={theme.table.action}
                                    title={intl.getMessage('edit_table_action')}
                                    aria-label={intl.getMessage('edit_table_action')}
                                    data-testid={`edit-rewrite-${row.domain}`}
                                    data-table-action
                                >
                                    <Icon icon="edit" color="gray" />
                                    <span class={theme.table.actionLabel}>
                                        {intl.getMessage('edit_table_action')}
                                    </span>
                                </button>

                                <button
                                    type="button"
                                    onClick={() => handleDelete(currentRewrite)}
                                    disabled={props.processingDelete}
                                    class={cn(theme.table.action, theme.table.action_danger)}
                                    title={intl.getMessage('delete_table_action')}
                                    aria-label={intl.getMessage('delete_table_action')}
                                    data-testid={`delete-rewrite-${row.domain}`}
                                    data-table-action
                                >
                                    <Icon icon="delete" color="red" />
                                    <span class={theme.table.actionLabel}>
                                        {intl.getMessage('delete_table_action')}
                                    </span>
                                </button>
                            </div>
                        </div>
                    </div>
                );
            },
        },
    ]);

    const handlePageSizeChange = (newSize: number) => {
        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.BLOCKLIST_PAGE_SIZE, newSize);
    };

    const emptyTableContent = () => {
        return (
            <div class={s.emptyTableContent}>
                <Icon icon="not_found_search" color="gray" class={s.emptyTableIcon} />

                <div class={cn(theme.text.t3, s.emptyTableDesc)}>
                    {intl.getMessage('rewrites_empty', {
                        button: (text: string) => (
                            <button
                                class={cn(theme.text.t3, theme.link.link)}
                                type="button"
                                onClick={() => props.addRewritesList()}
                            >
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
            <div
                class={cn(
                    theme.pagination.wrapper,
                    s.sortDropdownMobile,
                    s.sortDropdownMobileRewrites,
                )}
            >
                <SortSelect value={sortDirection()} onChange={setSortDirection} />
            </div>

            <Table<Rewrite>
                data={sortedList()}
                class={s.tableRewrites}
                columns={columns()}
                emptyTable={emptyTableContent()}
                loading={props.processing || props.processingAdd || props.processingDelete}
                pageSize={pageSize()}
                onPageSizeChange={handlePageSizeChange}
            />
        </>
    );
};
