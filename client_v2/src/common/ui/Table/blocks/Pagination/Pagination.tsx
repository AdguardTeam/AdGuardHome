import React, { useState, useMemo } from 'react';
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

const generatePageNumbers = (currentPage: number, totalPages: number): (number | 'ellipsis')[] => {
    const delta = 2; // Number of pages to show around current page
    const range: number[] = [];

    // Always show first page
    range.push(1);

    // Add pages around current page
    const start = Math.max(2, currentPage + 1 - delta);
    const end = Math.min(totalPages - 1, currentPage + 1 + delta);

    Array.from({ length: Math.max(0, end - start + 1) }, (_, i) => start + i).forEach((page) => range.push(page));

    // Always show last page (if more than 1 page)
    if (totalPages > 1) {
        range.push(totalPages);
    }

    // Remove duplicates and sort
    const uniqueRange = [...new Set(range)].sort((a, b) => a - b);

    // Add ellipsis where needed
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

export const Pagination = ({
    currentPage,
    totalPages,
    pageSize,
    pageSizeOptions,
    onPageChange,
    onPageSizeChange,
}: Props) => {
    const [limitMenuOpen, setLimitMenuOpen] = useState(false);

    const canPreviousPage = currentPage > 0;
    const canNextPage = currentPage < totalPages - 1;

    const pageNumbers = useMemo(() => generatePageNumbers(currentPage, totalPages), [currentPage, totalPages]);

    const limitMenu = (
        <div className={theme.dropdown.menu}>
            {pageSizeOptions.map((size) => (
                <div
                    key={size}
                    className={cn(theme.dropdown.item, {
                        [theme.dropdown.item_active]: pageSize === size,
                    })}
                    onClick={() => {
                        onPageSizeChange(size);
                        setLimitMenuOpen(false);
                    }}
                >
                    {intl.getMessage('rows_per_page', { value: size })}
                </div>
            ))}
        </div>
    );

    return (
        <div className={theme.pagination.wrapper}>
            <div className={theme.pagination.pagesContainer}>
                <button
                    type="button"
                    onClick={() => onPageChange(currentPage - 1)}
                    disabled={!canPreviousPage}
                    className={theme.pagination.button}
                >
                    <Icon icon="arrow" className={cn(theme.pagination.arrow, theme.pagination.arrow_left)} />
                </button>

                {pageNumbers.map((pageNum, index) => {
                    if (pageNum === 'ellipsis') {
                        return (
                            <span key={`ellipsis-${index}`} className={theme.pagination.summary}>
                                ...
                            </span>
                        );
                    }

                    const isCurrentPage = pageNum === currentPage + 1; // Convert 0-based to 1-based
                    return (
                        <button
                            key={pageNum}
                            type="button"
                            onClick={() => onPageChange(pageNum - 1)} // Convert 1-based to 0-based
                            className={cn(theme.pagination.button, isCurrentPage && theme.pagination.button_active)}
                        >
                            {pageNum}
                        </button>
                    );
                })}

                <button
                    type="button"
                    onClick={() => onPageChange(currentPage + 1)}
                    disabled={!canNextPage}
                    className={theme.pagination.button}
                >
                    <Icon icon="arrow" className={cn(theme.pagination.arrow, theme.pagination.arrow_right)} />
                </button>
            </div>

            <Dropdown
                trigger="click"
                position="bottomRight"
                menu={limitMenu}
                open={limitMenuOpen}
                onOpenChange={setLimitMenuOpen}
                iconClassName={theme.dropdown.icon}
                className={theme.dropdown.flexDropdownWrap}
                wrapClassName={cn(theme.dropdown.dropdown, theme.pagination.dropdownShowOnPage)}
            >
                <span className={theme.pagination.dropdownText}>
                    {intl.getMessage('rows_per_page', { value: pageSize })}
                </span>
            </Dropdown>
        </div>
    );
};
