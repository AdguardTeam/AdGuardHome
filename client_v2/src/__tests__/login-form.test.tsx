import { render } from '@solidjs/testing-library';
import { describe, it, expect } from 'vitest';

import { Form } from 'panel/login/Login/Form';

/**
 * Renders the form and returns the class list of the wrapper around the field
 * with `id`.
 */
const renderForm = () => {
    const { container } = render(() => <Form onSubmit={() => {}} />);

    return (id: string) => container.querySelector(`#${id}`)?.parentElement?.className ?? '';
};

describe('Login form fields', () => {
    it('uses the page surface instead of the on-card styling', () => {
        const fieldClass = renderForm();

        expect(fieldClass('username')).not.toContain('onCard');
        expect(fieldClass('password')).not.toContain('onCard');
    });
});
