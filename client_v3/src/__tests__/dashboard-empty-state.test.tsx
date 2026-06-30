import { render, screen } from '@solidjs/testing-library';
import { describe, it, expect } from 'vitest';
import { HashRouter, Route } from '@solidjs/router';
import type { Component } from 'solid-js';

import { EmptyState } from 'panel/components/Dashboard/blocks/EmptyState/EmptyState';

const renderWithRouter = (component: Component) => {
    return render(() => (
        <HashRouter>
            <Route path="/" component={component} />
        </HashRouter>
    ));
};

describe('Dashboard EmptyState', () => {
    it('renders no_stats_yet in default mode', () => {
        renderWithRouter(() => <EmptyState mode="default" />);
        expect(screen.getByTestId('dashboard-empty-state')).toBeInTheDocument();
        expect(screen.getByText(/No stats yet/)).toBeInTheDocument();
    });

    it('renders period_notify in disabled mode with settings link', () => {
        renderWithRouter(() => <EmptyState mode="disabled" />);
        expect(screen.getByTestId('dashboard-empty-state')).toBeInTheDocument();
        // period_notify: "To change the displayed periods, edit Statistics retention in General settings"
        expect(screen.getByText(/Statistics retention/)).toBeInTheDocument();
        // The "General settings" text is rendered inside a Link
        // Note: period_notify uses &nbsp; — the link name is "General\u00a0settings"
        expect(screen.getByRole('link', { name: /General/ })).toBeInTheDocument();
    });

    it('defaults to default mode when no mode prop provided', () => {
        renderWithRouter(() => <EmptyState />);
        expect(screen.getByText(/No stats yet/)).toBeInTheDocument();
    });
});
