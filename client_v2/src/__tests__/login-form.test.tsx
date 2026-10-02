import { render, waitFor } from '@solidjs/testing-library';
import { describe, it, expect } from 'vitest';

import { Form } from 'panel/login/Login/Form';
import { byViewportWidth, mockMatchMedia } from 'panel/__tests__/helpers/matchMedia';

/** Applies a theme the way `setUITheme` does, through the `<html>` attribute. */
const applyTheme = (theme: 'light' | 'dark') => {
    document.documentElement.dataset.theme = theme;
};

/**
 * Renders the form and returns the class list of the wrapper that carries the
 * `onCard` styling around the field with `id`.
 */
const renderForm = () => {
    const { container } = render(() => <Form onSubmit={() => {}} />);

    return (id: string) => container.querySelector(`#${id}`)?.parentElement?.className ?? '';
};

describe('Login form fields', () => {
    it('drops the card styling on mobile', () => {
        mockMatchMedia(byViewportWidth(360));
        applyTheme('dark');
        const fieldClass = renderForm();

        expect(fieldClass('username')).not.toContain('onCard');
        expect(fieldClass('password')).not.toContain('onCard');
    });

    it('uses the card styling from the tablet breakpoint up in the dark theme', () => {
        mockMatchMedia(byViewportWidth(1024));
        applyTheme('dark');
        const fieldClass = renderForm();

        expect(fieldClass('username')).toContain('onCard');
        expect(fieldClass('password')).toContain('onCard');
    });

    it('drops the card styling in the light theme', () => {
        mockMatchMedia(byViewportWidth(1024));
        applyTheme('light');
        const fieldClass = renderForm();

        expect(fieldClass('username')).not.toContain('onCard');
        expect(fieldClass('password')).not.toContain('onCard');
    });

    it('follows the viewport when it is resized', () => {
        const matchMedia = mockMatchMedia(byViewportWidth(360));
        applyTheme('dark');
        const fieldClass = renderForm();

        matchMedia.set(byViewportWidth(1024));

        expect(fieldClass('username')).toContain('onCard');
        expect(fieldClass('password')).toContain('onCard');
    });

    it('follows the theme when it changes', async () => {
        mockMatchMedia(byViewportWidth(1024));
        applyTheme('light');
        const fieldClass = renderForm();

        expect(fieldClass('username')).not.toContain('onCard');

        applyTheme('dark');

        // MutationObserver callbacks are queued, so the update is not synchronous.
        await waitFor(() => expect(fieldClass('username')).toContain('onCard'));
        expect(fieldClass('password')).toContain('onCard');
    });
});
