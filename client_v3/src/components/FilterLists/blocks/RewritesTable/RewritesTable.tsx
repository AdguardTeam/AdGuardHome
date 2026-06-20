import { createSignal, createMemo, untrack } from 'solid-js';
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
            fitContent: true,
            className: s.cellNameToggleOuter,
            render: (value: boolean, row: Rewrite) => {
                const { domain, enabled } = row;
                const id = `rewrite_${domain}`;

                return (
                    <div class={s.cell}>
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
                <div class={s.cell}>
                    <span class={s.cellLabel}>{intl.getMessage('name_label')}</span>

                    <div class={s.cellValue}>
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
                <div class={s.cell}>
                    <span class={s.cellLabel}>{intl.getMessage('result')}</span>

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
            sortable: false,
            width: 80,
            render: (value: any, row: Rewrite) => {
                const currentRewrite = {
                    answer: row.answer,
                    domain: row.domain,
                    enabled: row.enabled,
                };

                return (
                    <div class={s.cell}>
                        <div class={s.cellActions}>
                            <button
                                type="button"
                                onClick={() => untrack(() => props).editRewrite(currentRewrite)}
                                disabled={props.processingUpdate}
                                class={s.editAction}
                                data-testid={`edit-rewrite-${row.domain}`}
                            >
                                <span class={cn(s.editActionLabel, theme.text.t2)}>
                                    {intl.getMessage('edit_table_action')}
                                </span>
                                <span class={s.editActionIcon}>
                                    <Icon icon="edit" color="gray" />
                                </span>
                            </button>

                            <button
                                type="button"
                                onClick={() => untrack(() => props).deleteRewrite(currentRewrite)}
                                disabled={props.processingDelete}
                                class={s.action}
                                data-testid={`delete-rewrite-${row.domain}`}
                            >
                                <Icon icon="delete" color="red" />
                            </button>
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
