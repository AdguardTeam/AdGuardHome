import { describe, it, expect } from 'vitest';
import { render } from '@solidjs/testing-library';
import Toast from '../components/Toasts/Toast';

describe('Toast', () => {
    it('renders the plain message when no options', () => {
        const { getByText } = render(() => <Toast id="1" message="hello" type="success" />);
        expect(getByText('hello')).toBeTruthy();
    });
});
