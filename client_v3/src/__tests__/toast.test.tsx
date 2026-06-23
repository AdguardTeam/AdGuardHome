import { describe, it, expect } from 'vitest';
import { render } from '@solidjs/testing-library';
import Toast from '../components/Toasts/Toast';

describe('Toast', () => {
    it('renders the plain message when no options', () => {
        const { getByText } = render(() => (
            <Toast id="1" message="hello" type="success" />
        ));
        expect(getByText('hello')).toBeTruthy();
    });

    it('renders interpolated options.components content', () => {
        const { container } = render(() => (
            <Toast
                id="2"
                message="update_failed"
                type="notice"
                options={{ components: { a: (c: string) => c } }}
            />
        ));
        // Message rendered (not [object Object])
        expect(container.textContent).not.toContain('[object Object]');
    });
});
