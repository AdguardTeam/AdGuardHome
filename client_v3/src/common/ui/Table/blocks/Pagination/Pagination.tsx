import { createSignal, createMemo, For } from 'solid-js';
import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import { Dropdown } from 'panel/common/ui/Dropdown';
import intl from 'panel/common/intl';

type Props = {
    currentPage: number;
    totalPages: number;
    pageSize: number;
    totalItems: number;
    pageSizeOptions: number[];
    onPageChange: (page: number) => void;
    onPageSizeChange: (size: number) => void;
    limitButtonDescription?: string;
};

export const generatePageNumbers = (
    currentPage: number,
    totalPages: number,
): (number | 'ellipsis')[] => {
    const delta = 2;
    const range: number[] = [];

    range.push(1);

    const start = Math.max(2, currentPage + 1 - delta);
    const end = Math.min(totalPages - 1, currentPage + 1 + delta);

    Array.from({ length: Math.max(0, end - start + 1) }, (_, i) => start + i).forEach((page) =>
        range.push(page),
    );

    if (totalPages > 1) {
        range.push(totalPages);
    }

    const uniqueRange = [...new Set(range)].sort((a, b) => a - b);

    const rangeWithDots: (number | 'ellipsis')[] = [];
    uniqueRange.forEach((page, index) => {
        if (index > 0) {
            const prevPage = uniqueRange[index - 1];
            if (page - prevPage === 2) {
                rangeWithDots.push(prevPage + 1);
            } else if (page - prevPage > 2) {
                rangeWithDots.push('ellipsis');
            }
        }
        rangeWithDots.push(page);
    });

    return rangeWithDots;
};

export const Pagination = (props: Props) => {
    const [limitMenuOpen, setLimitMenuOpen] = createSignal(false);

    const canPreviousPage = () => props.currentPage > 0;
    const canNextPage = () => props.currentPage < props.totalPages - 1;

    const pageNumbers = createMemo(() => generatePageNumbers(props.currentPage, props.totalPages));

    const limitMenu = (
        <div class={theme.dropdown.menu}>
            <For each={props.pageSizeOptions}>
                {(size) => (
                    <div
                        class={cn(theme.dropdown.item, {
                            [theme.dropdown.item_active]: props.pageSize === size,
                        })}
                        onClick={() => {
                            props.onPageSizeChange(size);
                            setLimitMenuOpen(false);
                        }}
                    >
                        {intl.getMessage('rows_per_page', { value: size })}
                    </div>
                )}
            </For>
        </div>
    );

    const renderPages = () => {
        if (props.totalPages <= 1) {
            return null;
        }

        return (
            <div class={theme.pagination.pagesContainer}>
                <button
                    type="button"
                    aria-label={intl.getMessage('aria_previous_page')}
                    onClick={() => props.onPageChange(props.currentPage - 1)}
                    disabled={!canPreviousPage()}
                    class={theme.pagination.button}
                >
                    <Icon
                        icon="arrow"
                        class={cn(theme.pagination.arrow, theme.pagination.arrow_left)}
                    />
                </button>

                <For each={pageNumbers()}>
                    {(pageNum) => {
                        if (pageNum === 'ellipsis') {
                            return <span class={theme.pagination.summary}>...</span>;
                        }

                        return (
                            <button
                                type="button"
                                onClick={() => props.onPageChange(pageNum - 1)}
                                class={cn(
                                    theme.pagination.button,
                                    pageNum === props.currentPage + 1 &&
                                        theme.pagination.button_active,
                                )}
                            >
                                {pageNum}
                            </button>
                        );
                    }}
                </For>

                <button
                    type="button"
                    aria-label={intl.getMessage('aria_next_page')}
                    onClick={() => props.onPageChange(props.currentPage + 1)}
                    disabled={!canNextPage()}
                    class={theme.pagination.button}
                >
                    <Icon
                        icon="arrow"
                        class={cn(theme.pagination.arrow, theme.pagination.arrow_right)}
                    />
                </button>
            </div>
        );
    };

    return (
        <div class={theme.pagination.wrapper}>
            {renderPages()}

            <div class={theme.pagination.limitContainer}>
                <Dropdown
                    trigger="click"
                    position="bottomRight"
                    menu={limitMenu}
                    open={limitMenuOpen()}
                    onOpenChange={setLimitMenuOpen}
                    iconClass={theme.dropdown.icon}
                    class={theme.dropdown.flexDropdownWrap}
                    wrapClass={cn(theme.dropdown.dropdown, theme.pagination.dropdownShowOnPage)}
                >
                    <span class={theme.pagination.dropdownText}>
                        {intl.getMessage('rows_per_page', { value: props.pageSize })}
                    </span>
                </Dropdown>
            </div>
        </div>
    );
};
