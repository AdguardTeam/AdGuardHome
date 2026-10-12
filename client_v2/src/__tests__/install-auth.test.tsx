import { render, screen, fireEvent, waitFor } from '@solidjs/testing-library';
import { describe, expect, it, vi } from 'vitest';

import { Auth } from 'panel/install/Setup/Auth';
import { copy } from 'panel/__tests__/helpers/copy';

vi.mock('panel/stores/install', () => ({
    installState: { step: 2 },
    nextStep: vi.fn(),
    prevStep: vi.fn(),
}));

const setup = () => {
    const onAuthSubmit = vi.fn();
    render(() => <Auth onAuthSubmit={onAuthSubmit} />);

    return { onAuthSubmit };
};

const typeInto = (id: string, value: string) => {
    const input = document.querySelector(`#${id}`) as HTMLInputElement;

    // A real user typing fires `input` on every keystroke and `change` on blur;
    // modular-forms reads the value from `input`, Auth's own handler from `change`.
    fireEvent.input(input, { target: { value } });
    fireEvent.change(input, { target: { value } });
};

/** The inline error text rendered under the field with `id`, if any. */
const errorTextFor = (id: string) => {
    const input = document.querySelector(`#${id}`) as HTMLInputElement;
    const wrapper = input.closest('[class*="inputWrapper"]') as HTMLElement;

    return wrapper.parentElement?.querySelector('[class*="inputError"]')?.textContent ?? '';
};

const nextButton = () => document.querySelector('#install_next') as HTMLButtonElement;

const acceptConsent = () => {
    const checkbox = screen.getByRole('checkbox') as HTMLInputElement;

    // `fireEvent.click` double-toggles a checkbox inside a <label> under jsdom.
    fireEvent.change(checkbox, { target: { checked: true } });
};

const submitForm = () => {
    fireEvent.submit(document.querySelector('form') as HTMLFormElement);
};

const fillValidForm = (password = 'P@ssw0rd🔑') => {
    typeInto('install_username', 'admin');
    typeInto('install_password', password);
    typeInto('install_confirm_password', password);
};

describe('install wizard Auth step', () => {
    it('accepts a password containing emoji', async () => {
        const { onAuthSubmit } = setup();
        fillValidForm();
        acceptConsent();

        await waitFor(() => expect(nextButton().disabled).toBe(false));
        submitForm();

        await waitFor(() =>
            expect(onAuthSubmit).toHaveBeenCalledWith(
                {
                    username: 'admin',
                    password: 'P@ssw0rd🔑',
                    confirm_password: 'P@ssw0rd🔑',
                    privacy_consent: true,
                },
                // modular-forms passes the submit event as the second argument.
                expect.anything(),
            ),
        );
    });

    it('accepts a password with Cyrillic letters and emoji', async () => {
        const { onAuthSubmit } = setup();
        fillValidForm('Пароль🔑🔒');
        acceptConsent();

        await waitFor(() => expect(nextButton().disabled).toBe(false));
        submitForm();

        await waitFor(() => expect(onAuthSubmit).toHaveBeenCalledTimes(1));
        expect(onAuthSubmit.mock.calls[0][0].password).toBe('Пароль🔑🔒');
    });

    it('rejects a password without an uppercase letter', async () => {
        const { onAuthSubmit } = setup();
        fillValidForm('пароль🔑🔒');
        acceptConsent();

        expect(nextButton().disabled).toBe(true);

        submitForm();
        await waitFor(() =>
            expect(errorTextFor('install_password')).toBe(
                copy('password_requirements_uppercase'),
            ),
        );
        expect(onAuthSubmit).not.toHaveBeenCalled();
    });

    it('rejects a password without numbers or special characters', async () => {
        setup();
        fillValidForm('ПарольПароль');

        submitForm();
        await waitFor(() =>
            expect(errorTextFor('install_password')).toBe(copy('password_requirements_special')),
        );
    });

    it('rejects a too-short password', async () => {
        setup();
        fillValidForm('Aa1!');

        submitForm();
        await waitFor(() =>
            expect(errorTextFor('install_password')).toBe(
                copy('password_requirements_characters'),
            ),
        );
    });

    it('rejects a mismatched confirmation', async () => {
        setup();
        typeInto('install_username', 'admin');
        typeInto('install_password', 'P@ssw0rd🔑');
        typeInto('install_confirm_password', 'P@ssw0rd🔒');

        submitForm();
        await waitFor(() =>
            expect(errorTextFor('install_confirm_password')).toBe(copy('form_error_password')),
        );
    });

    it('keeps Next disabled until every requirement and the consent are met', async () => {
        setup();
        fillValidForm();

        // The consent checkbox is not checked yet.
        expect(nextButton().disabled).toBe(true);

        acceptConsent();
        await waitFor(() => expect(nextButton().disabled).toBe(false));

        typeInto('install_password', 'ПарольПароль');
        await waitFor(() => expect(nextButton().disabled).toBe(true));
    });

    it('renders all five requirement labels from the design', () => {
        setup();

        const labels = [
            copy('password_requirements_characters'),
            copy('password_requirements_special'),
            copy('password_requirements_lowercase'),
            copy('password_requirements_uppercase'),
            copy('password_requirements_match'),
        ];

        for (const label of labels) {
            expect(screen.getAllByText(label).length).toBeGreaterThan(0);
        }
    });
});
