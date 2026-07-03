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
    processingSettings: boolean;
    enabled: boolean;
    addRewritesList: () => void;
    deleteRewrite: (rewrite: Rewrite) => void;
    editRewrite: (rewrite: Rewrite) => void;
    toggleRewrite: (rewrite: Rewrite) => void;
    toggleAllRewrites: (enabled: boolean) => void;
};

export const RewritesTable = (props: Props) => {
    const [sortDirection, setSortDirection] = createSignal<'asc' | 'desc'>('asc');

    const pageSize = createMemo(
        () => LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.BLOCKLIST_PAGE_SIZE) || undefined,
    );

    const sortedList = createMemo(() => {
        const items = [...(props.list || [])];

        // eslint-disable-next-line solid/reactivity -- sort callback inside createMemo; reads already tracked
        items.sort((a, b) => {
            const aDomain = a.domain.toLowerCase();
            const bDomain = b.domain.toLowerCase();

            if (aDomain < bDomain) {
                return sortDirection() === 'asc' ? -1 : 1;
            }

            if (aDomain > bDomain) {
                return sortDirection() === 'asc' ? 1 : -1;
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
            key: 'enabled',
            header: {
                text: '',
                className: s.headerCell,
                render: () => (
                    <Switch
                        id="rewrite_global_enabled"
                        data-testid="rewrite-global-toggle"
                        checked={props.enabled}
                        onChange={() => props.toggleAllRewrites(!props.enabled)}
                        disabled={props.processingSettings}
                    />
                ),
            },
            accessor: 'enabled',
            sortable: false,
            width: 64,
            className: s.cellNameToggleOuter,
            render: (value: boolean, row: Rewrite) => {
                const { domain, enabled } = row;
                const id = `rewrite_${domain}`;

                return (
                    <div class={theme.table.cell}>
                        <span class={s.cellNameLabel}>{domain}</span>

                        <div class={s.cellValueToggle}>
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
            key: 'domain',
            header: {
                text: intl.getMessage('domain'),
                className: s.headerCell,
            },
            accessor: 'domain',
            sortable: true,
            className: s.nameDesktopOnly,
            render: (value: string) => (
                <div class={theme.table.cell}>
                    <span class={theme.table.cellLabel}>{intl.getMessage('name_label')}</span>

                    <div class={theme.table.cellValueText}>
                        <span class={theme.common.textOverflow}>{value}</span>
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
                <div class={theme.table.cell}>
                    <span class={theme.table.cellLabel}>{intl.getMessage('result')}</span>

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

            <div class={s.allDomainsMobile}>
                {intl.getMessage('all_domains')}

                <Switch
                    id="rewrite_global_enabled_mobile"
                    data-testid="rewrite-global-toggle-mobile"
                    checked={props.enabled}
                    onChange={() => props.toggleAllRewrites(!props.enabled)}
                    disabled={props.processingSettings}
                />
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
