import { render } from '@solidjs/testing-library';
import { describe, it, expect } from 'vitest';

import { Form } from 'panel/login/Login/Form';
import { byViewportWidth, mockMatchMedia } from 'panel/__tests__/helpers/matchMedia';

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
        const fieldClass = renderForm();

        expect(fieldClass('username')).not.toContain('onCard');
        expect(fieldClass('password')).not.toContain('onCard');
    });

    it('uses the card styling from the tablet breakpoint up', () => {
        mockMatchMedia(byViewportWidth(1024));
        const fieldClass = renderForm();

        expect(fieldClass('username')).toContain('onCard');
        expect(fieldClass('password')).toContain('onCard');
    });

    it('follows the viewport when it is resized', () => {
        const matchMedia = mockMatchMedia(byViewportWidth(360));
        const fieldClass = renderForm();

        matchMedia.set(byViewportWidth(1024));

        expect(fieldClass('username')).toContain('onCard');
        expect(fieldClass('password')).toContain('onCard');
    });
});
