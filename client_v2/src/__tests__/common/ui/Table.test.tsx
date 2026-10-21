import { createSignal, For } from 'solid-js';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import { MemoryRouter, Route, createMemoryHistory } from '@solidjs/router';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { Table, type TableColumn } from 'panel/common/ui/Table';

type Row = {
    name: string | null;
    age: number;
    city?: string;
};

const mocks = vi.hoisted(() => ({
    onSortChange: vi.fn(),
    onPageSizeChange: vi.fn(),
    onRowClick: vi.fn(),
}));

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

vi.mock('panel/common/controls/Select', () => ({
    Select: (props: any) => (
        <select
            data-testid="page-size-select"
            value={props.value?.value}
            onChange={(e: Event) => {
                const select = e.currentTarget as HTMLSelectElement;
                const option = props.options.find(
                    (item: any) => String(item.value) === select.value,
                );
                props.onChange(option);
            }}
        >
            <For each={props.options}>
                {(option: any) => <option value={option.value}>{option.label}</option>}
            </For>
        </select>
    ),
}));

const columns: TableColumn<Row>[] = [
    {
        key: 'name',
        header: { text: 'Name header', class: 'name-header' },
        accessor: 'name',
        sortable: true,
        class: 'name-column-cell',
        render: (value: string | null) => (
            <span data-testid="name-cell">{String(value ?? '')}</span>
        ),
    },
    {
        key: 'age',
        header: { text: 'Age header', class: 'age-header' },
        accessor: 'age',
        sortable: true,
        render: (value: number) => <span data-testid="age-cell">{value}</span>,
    },
    {
        key: 'city',
        header: { text: 'City header', class: 'city-header' },
        accessor: 'city',
        sortable: false,
        render: (value: string | undefined) => <span data-testid="city-cell">{value ?? ''}</span>,
    },
];

const makeRows = (count: number): Row[] =>
    Array.from({ length: count }, (_, index) => ({
        name: `row-${String(index).padStart(2, '0')}`,
        age: index,
        city: 'City',
    }));

const getNames = () => screen.getAllByTestId('name-cell').map((cell) => cell.textContent);

const renderTable = (options: {
    data: Row[];
    columns?: TableColumn<Row>[];
    [key: string]: any;
}) => {
    const { data, columns: tableColumns = columns, ...rest } = options;

    return render(() => <Table<Row> data={data} columns={tableColumns} {...rest} />);
};

const getHeader = (container: HTMLElement, selector: string) => {
    const header = container.querySelector(selector);
    if (!(header instanceof HTMLElement)) {
        throw new Error(`Table header not found: ${selector}`);
    }

    return header;
};

describe('Table', () => {
    beforeEach(() => vi.clearAllMocks());

    describe('rendering', () => {
        it('renders headers and values from accessors', () => {
            renderTable({ data: [{ name: 'Alice', age: 30, city: 'Paris' }] });

            expect(screen.getByText('Name header')).toBeDefined();
            expect(screen.getByText('Age header')).toBeDefined();
            expect(screen.getByText('City header')).toBeDefined();
            expect(screen.getByText('Alice')).toBeDefined();
            expect(screen.getByText('30')).toBeDefined();
            expect(screen.getByText('Paris')).toBeDefined();
        });

        it('renders custom cell content with value, row and index', () => {
            const renderCell = vi.fn((value: string, row: Row, index: number) => (
                <span data-testid="custom-cell">{`${index}:${row.name}:${value}`}</span>
            ));

            renderTable({
                data: [
                    { name: 'Alice', age: 30 },
                    { name: 'Bob', age: 40 },
                ],
                columns: [
                    {
                        key: 'name',
                        header: { text: 'Name' },
                        accessor: 'name',
                        render: renderCell,
                    },
                ],
            });

            expect(renderCell).toHaveBeenNthCalledWith(1, 'Alice', { name: 'Alice', age: 30 }, 0);
            expect(renderCell).toHaveBeenNthCalledWith(2, 'Bob', { name: 'Bob', age: 40 }, 1);
            expect(screen.getByText('0:Alice:Alice')).toBeDefined();
            expect(screen.getByText('1:Bob:Bob')).toBeDefined();
        });

        it('renders values from function accessors', () => {
            renderTable({
                data: [{ name: 'Alice', age: 30 }],
                columns: [
                    {
                        key: 'name',
                        header: { text: 'Name' },
                        accessor: (row) => row.name?.toUpperCase(),
                    },
                ],
            });

            expect(screen.getByText('ALICE')).toBeDefined();
        });

        it('renders an empty cell when the accessor value is missing', () => {
            const { container } = renderTable({
                data: [{ name: 'Alice', age: 30 }],
                columns: [{ key: 'city', header: { text: 'City' }, accessor: 'city' }],
                tableRowClass: 'test-row',
            });

            expect(container.querySelector('.test-row')?.textContent).toBe('');
        });

        it('renders a custom column header', () => {
            renderTable({
                data: [{ name: 'Alice', age: 30 }],
                columns: [
                    {
                        key: 'name',
                        header: {
                            text: 'Name',
                            render: () => <span data-testid="custom-header">Custom name</span>,
                        },
                        accessor: 'name',
                    },
                ],
            });

            expect(screen.getByTestId('custom-header')).toBeDefined();
        });
    });

    describe('empty state', () => {
        it('shows the empty state when there are no rows', () => {
            renderTable({ data: [], emptyTable: <div>Nothing here</div> });

            expect(screen.getByText('Nothing here')).toBeDefined();
        });

        it('does not show the empty state when rows exist', () => {
            renderTable({
                data: [{ name: 'Alice', age: 30 }],
                emptyTable: <div>Nothing here</div>,
            });

            expect(screen.queryByText('Nothing here')).toBeNull();
        });
    });

    describe('loading', () => {
        it('shows a loader instead of the table while loading', () => {
            const { container } = renderTable({
                data: [{ name: 'Alice', age: 30 }],
                loading: true,
            });

            expect(screen.queryByText('Name header')).toBeNull();
            expect(screen.queryByText('Alice')).toBeNull();
            expect(container.querySelector('svg')).not.toBeNull();
        });
    });

    describe('sorting', () => {
        const unsortedData: Row[] = [
            { name: 'Charlie', age: 3 },
            { name: 'Alice', age: 1 },
            { name: 'Bob', age: 2 },
        ];

        it('applies the default sort on mount', () => {
            renderTable({
                data: unsortedData,
                defaultSort: { key: 'name', direction: 'asc' },
            });

            expect(getNames()).toEqual(['Alice', 'Bob', 'Charlie']);
        });

        it('applies a descending default sort on mount', () => {
            renderTable({
                data: unsortedData,
                defaultSort: { key: 'age', direction: 'desc' },
            });

            expect(getNames()).toEqual(['Charlie', 'Bob', 'Alice']);
        });

        it('toggles between ascending and descending when a header is clicked', () => {
            const { container } = renderTable({ data: unsortedData });

            fireEvent.click(getHeader(container, '.name-header'));
            expect(getNames()).toEqual(['Alice', 'Bob', 'Charlie']);

            fireEvent.click(getHeader(container, '.name-header'));
            expect(getNames()).toEqual(['Charlie', 'Bob', 'Alice']);
        });

        it('sorts strings case-insensitively', () => {
            const { container } = renderTable({
                data: [
                    { name: 'beta', age: 2 },
                    { name: 'Alpha', age: 1 },
                    { name: 'gamma', age: 3 },
                ],
            });

            fireEvent.click(getHeader(container, '.name-header'));

            expect(getNames()).toEqual(['Alpha', 'beta', 'gamma']);
        });

        it('keeps null values last in both directions', () => {
            const { container } = renderTable({
                data: [
                    { name: 'b', age: 2 },
                    { name: null, age: 3 },
                    { name: 'a', age: 1 },
                ],
            });

            fireEvent.click(getHeader(container, '.name-header'));
            expect(getNames()).toEqual(['a', 'b', '']);

            fireEvent.click(getHeader(container, '.name-header'));
            expect(getNames()).toEqual(['b', 'a', '']);
        });

        it('does not sort a column marked as not sortable', () => {
            const { container } = renderTable({ data: unsortedData });

            fireEvent.click(getHeader(container, '.city-header'));

            expect(getNames()).toEqual(['Charlie', 'Alice', 'Bob']);
        });

        it('does not sort when the table sortable prop is false', () => {
            const { container } = renderTable({ data: unsortedData, sortable: false });

            fireEvent.click(getHeader(container, '.name-header'));

            expect(getNames()).toEqual(['Charlie', 'Alice', 'Bob']);
        });

        it('uses a custom sortFn', () => {
            const { container } = renderTable({
                data: [
                    { name: 'ccc', age: 3 },
                    { name: 'a', age: 1 },
                    { name: 'bb', age: 2 },
                ],
                columns: [
                    {
                        key: 'name',
                        header: { text: 'Name header', class: 'name-header' },
                        accessor: 'name',
                        sortable: true,
                        sortFn: (a: string, b: string) => a.length - b.length,
                        render: (value: string) => <span data-testid="name-cell">{value}</span>,
                    },
                ],
            });

            fireEvent.click(getHeader(container, '.name-header'));

            expect(getNames()).toEqual(['a', 'bb', 'ccc']);
        });

        it('calls onSortChange with the column key and direction', () => {
            const { container } = renderTable({
                data: unsortedData,
                onSortChange: mocks.onSortChange,
            });

            fireEvent.click(getHeader(container, '.name-header'));
            expect(mocks.onSortChange).toHaveBeenLastCalledWith('name', 'asc');

            fireEvent.click(getHeader(container, '.name-header'));
            expect(mocks.onSortChange).toHaveBeenLastCalledWith('name', 'desc');
        });

        it('returns to the first page when sorting', () => {
            const { container } = renderTable({ data: makeRows(15) });

            fireEvent.click(screen.getByRole('button', { name: '2' }));
            expect(getNames()[0]).toBe('row-10');

            fireEvent.click(getHeader(container, '.name-header'));

            expect(getNames()[0]).toBe('row-00');
            expect(screen.queryByText('row-10')).toBeNull();
        });
    });

    describe('pagination', () => {
        it('paginates rows using the default page size', () => {
            renderTable({ data: makeRows(11) });

            expect(getNames()).toHaveLength(10);
            expect(screen.queryByText('row-10')).toBeNull();

            fireEvent.click(screen.getByRole('button', { name: '2' }));

            expect(getNames()).toEqual(['row-10']);
        });

        it('uses the pageSize prop', () => {
            renderTable({ data: makeRows(12), pageSize: 5 });

            expect(getNames()).toEqual(['row-00', 'row-01', 'row-02', 'row-03', 'row-04']);

            fireEvent.click(screen.getByRole('button', { name: '2' }));
            expect(getNames()).toEqual(['row-05', 'row-06', 'row-07', 'row-08', 'row-09']);

            fireEvent.click(screen.getByRole('button', { name: '3' }));
            expect(getNames()).toEqual(['row-10', 'row-11']);
        });

        it('shows all rows and no pagination when pagination is disabled', () => {
            renderTable({ data: makeRows(11), pagination: false });

            expect(getNames()).toHaveLength(11);
            expect(screen.queryByRole('button', { name: '2' })).toBeNull();
            expect(screen.queryByTestId('page-size-select')).toBeNull();
        });

        it('hides the pagination footer when the data fits on one page', () => {
            renderTable({ data: makeRows(10) });

            expect(screen.queryByRole('button', { name: '2' })).toBeNull();
            expect(screen.queryByTestId('page-size-select')).toBeNull();
        });

        it('changes the page size and returns to the first page', () => {
            renderTable({
                data: makeRows(12),
                pageSize: 5,
                onPageSizeChange: mocks.onPageSizeChange,
            });

            fireEvent.click(screen.getByRole('button', { name: '2' }));
            expect(getNames()[0]).toBe('row-05');

            fireEvent.change(screen.getByTestId('page-size-select'), { target: { value: '10' } });

            expect(mocks.onPageSizeChange).toHaveBeenCalledWith(10);
            expect(getNames()[0]).toBe('row-00');
        });

        it('moves to the previous page when the last row of the current page is removed', async () => {
            const [rows, setRows] = createSignal(makeRows(11));

            render(() => (
                <Table<Row>
                    data={rows()}
                    columns={columns}
                    pageSize={10}
                    emptyTable={<div>empty</div>}
                />
            ));

            fireEvent.click(screen.getByRole('button', { name: '2' }));
            expect(screen.getByText('row-10')).toBeDefined();

            setRows(makeRows(10));

            expect(await screen.findByText('row-09')).toBeDefined();
            expect(screen.getByText('row-00')).toBeDefined();
            expect(screen.queryByText('empty')).toBeNull();
            // The clamp must land on page 0, not on a stale index.
            expect(screen.queryByRole('button', { name: '2' })).toBeNull();
        });
    });

    describe('row interactions', () => {
        it('calls onRowClick with the clicked row', () => {
            const row = { name: 'Alice', age: 30 };

            renderTable({ data: [row], onRowClick: mocks.onRowClick });

            fireEvent.click(screen.getByText('Alice'));

            expect(mocks.onRowClick).toHaveBeenCalledWith(row);
        });

        it('renders rows as links when rowLink returns a target', () => {
            const history = createMemoryHistory();
            history.set({ value: '/' });

            render(() => (
                <MemoryRouter history={history}>
                    <Route
                        path="/*"
                        component={() => (
                            <Table<Row>
                                data={[{ name: 'Alice', age: 30 }]}
                                columns={columns}
                                rowLink={() => ({ to: 'DnsRewrites', query: { foo: 'bar' } })}
                            />
                        )}
                    />
                </MemoryRouter>
            ));

            expect(screen.getByRole('link').getAttribute('href')).toBe('/dns_rewrites?foo=bar');
        });
    });

    describe('layout', () => {
        it('applies custom table, header and row classes', () => {
            const { container } = renderTable({
                data: [{ name: 'Alice', age: 30 }],
                class: 'custom-table',
                tableHeaderClass: 'custom-header',
                tableRowClass: 'custom-row',
            });

            expect(container.querySelector('.custom-table')).not.toBeNull();
            expect(container.querySelector('.custom-header')).not.toBeNull();
            expect(container.querySelector('.custom-row')).not.toBeNull();
        });

        it('applies a custom class to body cells', () => {
            const { container } = renderTable({ data: [{ name: 'Alice', age: 30 }] });

            expect(container.querySelector('.name-column-cell')).not.toBeNull();
        });

        it('builds the grid template from column widths', () => {
            const { container } = renderTable({
                data: [{ name: 'Alice', age: 30 }],
                columns: [
                    { key: 'a', header: { text: 'A' }, accessor: 'name', width: 100 },
                    { key: 'b', header: { text: 'B' }, accessor: 'age', width: '20%' },
                    { key: 'c', header: { text: 'C' }, accessor: 'city', fitContent: true },
                    {
                        key: 'd',
                        header: { text: 'D' },
                        accessor: 'name',
                        minWidth: 80,
                        maxWidth: 200,
                    },
                    { key: 'e', header: { text: 'E' }, accessor: 'age' },
                ],
                tableHeaderClass: 'grid-header',
            });

            const header = getHeader(container, '.grid-header');

            expect(header.style.getPropertyValue('--table-columns')).toBe(
                '100px 20% fit-content(100%) minmax(80px, 200px) minmax(0, 1fr)',
            );
        });
    });
});
