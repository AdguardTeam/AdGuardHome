import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@solidjs/testing-library';
import { createSignal } from 'solid-js';

// CSS modules return {} under css:false, so theme class names would be
// undefined. Mock theme with identity class names for deterministic asserts.
vi.mock('panel/lib/theme', () => {
    const make = (names: string[]) => Object.fromEntries(names.map((n) => [n, n]));
    return {
        default: {
            pagination: make([
                'wrapper',
                'pagesContainer',
                'button',
                'button_active',
                'arrow',
                'arrow_left',
                'arrow_right',
                'summary',
                'dropdownText',
                'dropdownShowOnPage',
                'limitContainer',
            ]),
            dropdown: make(['menu', 'item', 'item_active', 'icon', 'flexDropdownWrap', 'dropdown']),
        },
    };
});

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string, values?: { value?: number }) =>
            values?.value !== undefined ? `${key}:${values.value}` : key,
    },
}));

vi.mock('panel/common/ui/Dropdown', () => ({
    Dropdown: (props: any) => (
        <>
            <button
                data-testid="page-size-trigger"
                onClick={() => props.onOpenChange?.(!props.open)}
            >
                {props.children}
            </button>
            {props.open && <div data-testid="page-size-menu">{props.menu}</div>}
        </>
    ),
}));

import { generatePageNumbers, Pagination } from '../common/ui/Table/blocks/Pagination/Pagination';

describe('generatePageNumbers', () => {
    it('returns a single page when totalPages is 1', () => {
        expect(generatePageNumbers(0, 1)).toEqual([1]);
    });

    it('shows all pages without ellipsis for a small page count', () => {
        expect(generatePageNumbers(0, 5)).toEqual([1, 2, 3, 4, 5]);
    });

    it('inserts an ellipsis at the end when on the first page of many', () => {
        expect(generatePageNumbers(0, 20)).toEqual([1, 2, 3, 'ellipsis', 20]);
    });

    it('inserts ellipses on both sides when in the middle of many pages', () => {
        expect(generatePageNumbers(9, 20)).toEqual([
            1,
            'ellipsis',
            8,
            9,
            10,
            11,
            12,
            'ellipsis',
            20,
        ]);
    });
});

const baseProps = {
    pageSize: 10,
    totalItems: 100,
    pageSizeOptions: [10, 20, 30],
    onPageChange: vi.fn(),
    onPageSizeChange: vi.fn(),
};

const pageButton = (container: HTMLElement, text: string) =>
    [...container.querySelectorAll('button')].find((b) => b.textContent === text)!;

describe('Pagination', () => {
    it('applies the active class only to the current page button', () => {
        const { container } = render(() => (
            <Pagination {...baseProps} currentPage={2} totalPages={10} />
        ));
        // currentPage is 0-based (2) → 1-based page "3" is active
        expect(pageButton(container, '3').className).toContain('button_active');
        expect(pageButton(container, '4').className).not.toContain('button_active');
        expect(pageButton(container, '1').className).not.toContain('button_active');
    });

    it('moves the active class when currentPage changes reactively', () => {
        const [current, setCurrent] = createSignal(2);
        const { container } = render(() => (
            <Pagination {...baseProps} currentPage={current()} totalPages={10} />
        ));
        expect(pageButton(container, '3').className).toContain('button_active');
        setCurrent(3);
        expect(pageButton(container, '3').className).not.toContain('button_active');
        expect(pageButton(container, '4').className).toContain('button_active');
    });

    it('calls onPageChange with the 0-based index when a page button is clicked', () => {
        const { container } = render(() => (
            <Pagination {...baseProps} currentPage={0} totalPages={5} />
        ));
        fireEvent.click(pageButton(container, '2'));
        expect(baseProps.onPageChange).toHaveBeenCalledWith(1);
    });

    it('prev/next buttons call onPageChange with currentPage - 1 / + 1', () => {
        const { container } = render(() => (
            <Pagination {...baseProps} currentPage={2} totalPages={5} />
        ));
        const buttons = container.querySelectorAll('button');
        // buttons: prev, 1..5, next, page-size-trigger (mock)
        fireEvent.click(buttons[0]); // prev
        expect(baseProps.onPageChange).toHaveBeenLastCalledWith(1);
        fireEvent.click(buttons[buttons.length - 2]); // next (before mock trigger)
        expect(baseProps.onPageChange).toHaveBeenLastCalledWith(3);
    });

    it('disables prev on the first page and next on the last page', () => {
        const { container: first } = render(() => (
            <Pagination {...baseProps} currentPage={0} totalPages={5} />
        ));
        // buttons: prev, 1..5, next, page-size-trigger (mock)
        const firstBtns = first.querySelectorAll('button');
        expect(firstBtns[0]).toBeDisabled();
        expect(firstBtns[firstBtns.length - 2]).not.toBeDisabled();

        const { container: last } = render(() => (
            <Pagination {...baseProps} currentPage={4} totalPages={5} />
        ));
        const lastBtns = last.querySelectorAll('button');
        expect(lastBtns[0]).not.toBeDisabled();
        expect(lastBtns[lastBtns.length - 2]).toBeDisabled();
    });

    it('does not render page buttons when totalPages <= 1', () => {
        const { container } = render(() => (
            <Pagination {...baseProps} currentPage={0} totalPages={1} />
        ));
        // Only the page-size trigger button exists (no prev/next/page buttons).
        const buttons = container.querySelectorAll('button');
        expect(buttons).toHaveLength(1);
    });

    it('calls onPageSizeChange when a page-size option is selected', () => {
        const { getByTestId } = render(() => (
            <Pagination {...baseProps} currentPage={0} totalPages={3} />
        ));
        fireEvent.click(getByTestId('page-size-trigger'));
        fireEvent.click(getByTestId('page-size-menu').querySelectorAll('[class~="item"]')[1]);
        expect(baseProps.onPageSizeChange).toHaveBeenCalledWith(20);
    });
});
