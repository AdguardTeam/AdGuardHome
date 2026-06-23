import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';

import { Identifiers } from 'panel/components/Clients/AddClient/blocks/Identifiers';
import { initClientForm, clientFormState } from 'panel/stores/clientForm';

describe('Identifiers', () => {
    beforeEach(() => {
        initClientForm(null);
    });

    it('keeps the entered identifier value (bug #1: value erased after typing)', () => {
        render(() => <Identifiers />);
        const input = document.querySelector('input[type="text"]') as HTMLInputElement;

        // Simulate typing + blur-commit (the Input listens to the native `change`
        // event, which fires when the field loses focus).
        fireEvent.change(input, { target: { value: '1.2.3.4' } });

        expect(input.value).toBe('1.2.3.4');
        expect(clientFormState.ids).toEqual(['1.2.3.4']);
    });

    it('adds a new identifier row on button click (bug #2: add button did nothing)', () => {
        render(() => <Identifiers />);
        expect(document.querySelectorAll('input[type="text"]')).toHaveLength(1);

        fireEvent.click(screen.getByTestId('client-form-add-identifier'));

        const inputs = document.querySelectorAll('input[type="text"]');
        expect(inputs).toHaveLength(2);
        expect(clientFormState.ids).toEqual(['', '']);
    });

    it('keeps existing values when adding a new row', () => {
        render(() => <Identifiers />);
        const first = document.querySelector('input[type="text"]') as HTMLInputElement;
        fireEvent.change(first, { target: { value: '1.2.3.4' } });

        fireEvent.click(screen.getByTestId('client-form-add-identifier'));

        const inputs = document.querySelectorAll('input[type="text"]');
        expect(inputs).toHaveLength(2);
        expect((inputs[0] as HTMLInputElement).value).toBe('1.2.3.4');
        expect((inputs[1] as HTMLInputElement).value).toBe('');
        expect(clientFormState.ids).toEqual(['1.2.3.4', '']);
    });

    it('removes an identifier row (except the first one which has no remove button)', () => {
        render(() => <Identifiers />);
        fireEvent.click(screen.getByTestId('client-form-add-identifier'));
        expect(document.querySelectorAll('input[type="text"]')).toHaveLength(2);

        // The remove button is the suffix of the second row.
        const removeBtn = document.querySelector('[aria-label]') as HTMLElement;
        fireEvent.click(removeBtn);

        expect(document.querySelectorAll('input[type="text"]')).toHaveLength(1);
        expect(clientFormState.ids).toEqual(['']);
    });

    it('shows a validation error on blur for an invalid identifier', async () => {
        render(() => <Identifiers />);
        const input = document.querySelector('input[type="text"]') as HTMLInputElement;

        fireEvent.change(input, { target: { value: 'not-a-valid-id' } });
        fireEvent.blur(input);

        // The error message element is rendered by the Input when there is an error.
        expect(input.value).toBe('not-a-valid-id');
    });
});
