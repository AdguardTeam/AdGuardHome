import { render } from '@solidjs/testing-library';
import { describe, it, expect, vi } from 'vitest';
import { Select } from '../common/controls/Select/Select';

const OPTIONS = [
    { value: '0.0.0.0', label: 'All interfaces' },
    { value: '127.0.0.1', label: 'Loopback' },
];

const MANY_OPTIONS = Array.from({ length: 15 }, (_, i) => ({
    value: `opt-${i}`,
    label: `Option ${i}`,
}));

describe('Select', () => {
    it('renders the indicator inside the trigger button', () => {
        render(() => (
            <Select
                options={OPTIONS}
                value={OPTIONS[0]}
                onChange={() => {}}
                placeholder="Select interface"
            />
        ));

        const trigger = document.querySelector(
            '[data-scope="select"][data-part="trigger"]',
        ) as HTMLElement;
        const indicator = trigger?.querySelector('[data-part="indicator"]');

        // The indicator (arrow icon) should be inside the trigger.
        expect(indicator).toBeTruthy();
    });
});

describe('Select — single-select (searchable combobox)', () => {
    it('shows the selected value via an overlay above the input', () => {
        // react-select parity: the value text is rendered behind the search
        // input (same grid cell) so it stays visible until the user types.
        render(() => (
            <Select
                options={OPTIONS}
                value={OPTIONS[1]}
                onChange={() => {}}
                isSearchable
                placeholder="Select interface"
            />
        ));

        const overlay = document.querySelector('.solid-combobox-single-value');
        expect(overlay).toBeInTheDocument();
        expect(overlay?.textContent).toBe(OPTIONS[1].label);

        // The overlay and the input share the same grid wrapper.
        const wrapper = document.querySelector('.solid-combobox-single-value-wrapper');
        expect(wrapper).toBeInTheDocument();
        expect(
            wrapper?.querySelector('[data-scope="combobox"][data-part="input"]'),
        ).toBeInTheDocument();

        // The input's own placeholder is empty — the overlay shows the value.
        const input = document.querySelector(
            '[data-scope="combobox"][data-part="input"]',
        ) as HTMLInputElement;
        expect(input.placeholder).toBe('');
    });

    it('shows the placeholder overlay when no value is selected', () => {
        render(() => (
            <Select
                options={OPTIONS}
                value={undefined}
                onChange={() => {}}
                isSearchable
                placeholder="Select interface"
            />
        ));

        // No value overlay.
        expect(document.querySelector('.solid-combobox-single-value')).not.toBeInTheDocument();

        // Placeholder text is rendered instead.
        const placeholder = document.querySelector('.solid-select-placeholder');
        expect(placeholder).toBeInTheDocument();
        expect(placeholder?.textContent).toBe('Select interface');
    });
});

describe('Select — multi-select (searchable combobox)', () => {
    it('places the search input inline inside the multi-value container', () => {
        render(() => (
            <Select
                options={OPTIONS}
                value={[]}
                onChange={() => {}}
                isMulti
                isSearchable
                placeholder="Pick tags"
            />
        ));

        const container = document.querySelector('.solid-select-multi-value-container');
        expect(container).toBeInTheDocument();

        // The search input must be a descendant of the multi-value container so
        // it flows inline after the pills (mimics react-select).
        const inputInside = container?.querySelector('[data-scope="combobox"][data-part="input"]');
        expect(inputInside).toBeInTheDocument();
    });

    it('hides the dropdown arrow and shows the clear button when values exist', () => {
        // NOTE: isClearable is intentionally NOT passed. react-select v2's
        // CustomClearIndicator only checks hasValue, and no consumer passes
        // isClearable, so the clear button must appear for multi-select
        // whenever values exist.
        render(() => (
            <Select
                options={OPTIONS}
                value={[OPTIONS[1]]}
                onChange={() => {}}
                isMulti
                isSearchable
            />
        ));

        // Dropdown arrow is replaced by the clear button (react-select parity).
        const trigger = document.querySelector('[data-scope="combobox"][data-part="trigger"]');
        expect(trigger).not.toBeInTheDocument();

        const clearTrigger = document.querySelector(
            '[data-scope="combobox"][data-part="clear-trigger"]',
        );
        expect(clearTrigger).toBeInTheDocument();
    });
});

describe('Select — multi-select (non-searchable) regression', () => {
    it('keeps the multi-value container that wraps the pills', () => {
        // SelectMultiValue no longer renders its own container div; both the
        // searchable and non-searchable branches must provide it (see C1 fix).
        render(() => (
            <Select
                options={OPTIONS}
                value={[OPTIONS[1]]}
                onChange={() => {}}
                isMulti
                isSearchable={false}
            />
        ));

        const container = document.querySelector('.solid-select-multi-value-container');
        expect(container).toBeInTheDocument();
    });
});

describe('Select — empty state (nothing found)', () => {
    it('renders a nothing-found message when the search yields no options', () => {
        render(() => (
            <Select options={[]} onChange={() => {}} isSearchable menuIsOpen placeholder="Search" />
        ));

        const empty = document.querySelector('[data-scope="combobox"][data-part="empty"]');
        expect(empty).toBeInTheDocument();
        expect(empty?.textContent).toContain('Nothing found');
    });
});

describe('Select — menuFooter', () => {
    const Footer = () => <div data-testid="test-footer">Footer content</div>;

    it('renders menuFooter in non-searchable mode', () => {
        render(() => (
            <Select
                options={OPTIONS}
                value={OPTIONS[0]}
                onChange={() => {}}
                isSearchable={false}
                menuIsOpen
                menuFooter={<Footer />}
            />
        ));

        // The footer must appear inside the dropdown content.
        const content = document.querySelector('[data-scope="select"][data-part="content"]');
        expect(content).toBeInTheDocument();
        expect(content?.querySelector('[data-testid="test-footer"]')).toBeInTheDocument();
    });

    it('renders menuFooter in searchable non-lazy mode (Bug #2 regression)', () => {
        // Previously, menuFooter was only rendered in the lazy branch for the
        // searchable Combobox — the non-lazy branch omitted it entirely.
        render(() => (
            <Select
                options={OPTIONS}
                value={OPTIONS[0]}
                onChange={() => {}}
                isSearchable
                menuIsOpen
                menuFooter={<Footer />}
            />
        ));

        const content = document.querySelector('[data-scope="combobox"][data-part="content"]');
        expect(content).toBeInTheDocument();
        expect(content?.querySelector('[data-testid="test-footer"]')).toBeInTheDocument();
    });

    it('renders menuFooter in searchable lazy mode', () => {
        render(() => (
            <Select
                options={OPTIONS}
                value={OPTIONS[0]}
                onChange={() => {}}
                isSearchable
                lazyList
                menuIsOpen
                menuFooter={<Footer />}
            />
        ));

        const content = document.querySelector('[data-scope="combobox"][data-part="content"]');
        expect(content).toBeInTheDocument();
        expect(content?.querySelector('[data-testid="test-footer"]')).toBeInTheDocument();
    });
});

describe('Select — autoFocus', () => {
    it('calls focus on the trigger in non-searchable mode', () => {
        // jsdom may not actually move focus, but we can verify that
        // the ref callback calls `el.focus()` when `autoFocus` is set.
        const focusSpy = vi.spyOn(HTMLElement.prototype, 'focus');

        render(() => (
            <Select
                options={OPTIONS}
                value={OPTIONS[0]}
                onChange={() => {}}
                isSearchable={false}
                autoFocus
            />
        ));

        const trigger = document.querySelector('[data-scope="select"][data-part="trigger"]');
        expect(trigger).toBeTruthy();
        expect(focusSpy).toHaveBeenCalled();

        focusSpy.mockRestore();
    });
});

describe('Select — onMenuScrollToBottom', () => {
    it('fires when the lazy menu list is scrolled (Bug #3 regression)', () => {
        // Bug #3: in lazy mode, the scrollable inner div (data-part="menu-list")
        // had no onScroll handler — scroll events don't bubble to the Ark UI
        // Content element, so the callback never fired. The fix attaches
        // onScroll to both Content and the inner div.
        const onScrollToBottom = vi.fn();

        render(() => (
            <Select
                options={OPTIONS}
                value={OPTIONS[0]}
                onChange={() => {}}
                isSearchable
                lazyList
                menuIsOpen
                onMenuScrollToBottom={onScrollToBottom}
            />
        ));

        // The lazy list renders a scrollable inner div identified by
        // data-part="menu-list" (CSS module class names are not available in
        // the test environment with css: false).
        const menuList = document.querySelector('[data-part="menu-list"]') as HTMLElement;
        expect(menuList).toBeInTheDocument();

        // In jsdom, all scroll dimensions are 0 so the "near bottom" condition
        // (scrollHeight - scrollTop <= clientHeight + 50) is always true.
        menuList.dispatchEvent(new Event('scroll'));

        expect(onScrollToBottom).toHaveBeenCalledTimes(1);
    });
});

describe('Select — auto-searchable threshold', () => {
    it('uses non-searchable ArkSelect for <= 10 options by default', () => {
        render(() => <Select options={OPTIONS} value={OPTIONS[0]} onChange={() => {}} />);

        // ArkSelect renders data-scope="select".
        expect(document.querySelector('[data-scope="select"]')).toBeInTheDocument();
        expect(document.querySelector('[data-scope="combobox"]')).not.toBeInTheDocument();
    });

    it('auto-enables search for > 10 options by default (single-select)', () => {
        render(() => <Select options={MANY_OPTIONS} onChange={() => {}} />);

        expect(document.querySelector('[data-scope="combobox"]')).toBeInTheDocument();
        expect(document.querySelector('[data-scope="select"]')).not.toBeInTheDocument();
    });
});
