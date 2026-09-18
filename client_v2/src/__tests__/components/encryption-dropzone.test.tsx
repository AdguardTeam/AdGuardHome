import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, fireEvent, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { createSignal } from 'solid-js';

const { themeMock, encryptionStateMock } = vi.hoisted(() => {
    const proxy: any = new Proxy(
        {},
        {
            get: (_target, prop) => {
                if (prop === Symbol.toPrimitive || prop === 'toString') {
                    return () => '';
                }
                return proxy;
            },
        },
    );

    return {
        themeMock: proxy,
        encryptionStateMock: {
            enabled: false,
            private_key_saved: false,
            server_name: '',
            force_https: false,
            port_https: 443,
            port_dns_over_tls: 853,
            port_dns_over_quic: 784,
            processingConfig: false,
            processingValidate: false,
        } as Record<string, unknown>,
    };
});

vi.mock('panel/lib/theme', () => ({
    default: themeMock,
}));

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string, _values?: Record<string, unknown>) => key,
    },
}));

vi.mock('panel/stores/encryption', () => ({
    encryptionState: encryptionStateMock,
    setTlsConfig: vi.fn(),
    validateTlsConfig: vi.fn(async () => ({ valid_cert: true, valid_key: true, valid_pair: true })),
}));

import { Dropzone } from 'panel/common/ui/Dropzone';
import { Textarea } from 'panel/common/controls/Textarea';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';
import { TlsSetupWizard } from 'panel/components/Encryption/blocks/SetupWizard/TlsSetupWizard';

const FILE_CONTENT = '-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----';
const CERT_PEM = `-----BEGIN CERTIFICATE-----\nMIIBxTCCAS6gAwIBAgIUNW1eQ0p0q5p0YfGq3Gq3Gq3Gq3EwCgYIKoZIzj0EAwIw\n-----END CERTIFICATE-----`;
const KEY_PEM = '-----BEGIN PRIVATE KEY-----\nMIIB\n-----END PRIVATE KEY-----';

const readAsText = vi.fn();

// Minimal FileReader stand-in: resolves synchronously so assertions stay simple.
class MockFileReader {
    onload: (() => void) | null = null;

    result: string | ArrayBuffer | null = null;

    readAsText(file: File) {
        readAsText(file);
        this.result = FILE_CONTENT;
        this.onload?.();
    }
}

/** jsdom's DragEvent support varies, so build the event by hand. */
const fireDragEvent = (el: Element, type: string, init: Record<string, unknown> = {}) => {
    const event = new Event(type, { bubbles: true, cancelable: true });
    Object.assign(event, init);
    fireEvent(el, event);
};

const makeFile = (name = 'cert.pem') => new File([FILE_CONTENT], name, { type: 'text/plain' });

describe('Dropzone', () => {
    beforeEach(() => {
        vi.stubGlobal('FileReader', MockFileReader);
        readAsText.mockClear();
    });

    afterEach(() => {
        vi.unstubAllGlobals();
    });

    const renderZone = (onFileSelect = vi.fn()) => {
        const utils = render(() => (
            <Dropzone onFileSelect={onFileSelect} hint="drop hint" testId="test-zone" />
        ));

        return {
            ...utils,
            onFileSelect,
            zone: screen.getByTestId('test-zone'),
            input: utils.container.querySelector('input[type="file"]') as HTMLInputElement,
        };
    };

    it('renders the hint and the browse affordance at rest', () => {
        const { zone } = renderZone();

        expect(zone.tagName).toBe('BUTTON');
        expect(zone.getAttribute('type')).toBe('button');
        expect(screen.getByText('drop hint')).toBeInTheDocument();
        expect(screen.getByText('browse')).toBeInTheDocument();
    });

    it('swaps the browse link for the download icon while dragging', () => {
        const { zone } = renderZone();

        fireDragEvent(zone, 'dragover');

        expect(screen.queryByText('browse')).not.toBeInTheDocument();
        expect(zone.querySelector('svg use')?.getAttribute('href')).toBe('#download');
        expect(zone.className).toContain('dragOver');

        fireDragEvent(zone, 'dragleave', { relatedTarget: document.body });

        expect(screen.getByText('browse')).toBeInTheDocument();
        expect(screen.queryByText('drop hint')).toBeInTheDocument();
        expect(zone.className).not.toContain('dragOver');
    });

    it('applies the consumer spacing class to the zone', () => {
        render(() => (
            <Dropzone
                onFileSelect={vi.fn()}
                hint="drop hint"
                testId="test-zone"
                class="custom-gap"
            />
        ));

        expect(screen.getByTestId('test-zone').className).toContain('custom-gap');
    });

    it('stays active when the pointer moves onto a child node', () => {
        const { zone } = renderZone();
        const hint = screen.getByText('drop hint');

        fireDragEvent(zone, 'dragover');
        fireDragEvent(zone, 'dragleave', { relatedTarget: hint });

        expect(zone.querySelector('svg use')?.getAttribute('href')).toBe('#download');
    });

    it('clears the drag state after a drop', () => {
        const { zone } = renderZone();

        fireDragEvent(zone, 'dragover');
        fireDragEvent(zone, 'drop', { dataTransfer: { files: [makeFile()] } });

        expect(screen.getByText('browse')).toBeInTheDocument();
    });

    it('opens the file picker on click', () => {
        const { zone, input } = renderZone();
        const clickSpy = vi.spyOn(input, 'click');

        fireEvent.click(zone);

        expect(clickSpy).toHaveBeenCalledOnce();
    });

    it('opens the file picker with the keyboard', async () => {
        const user = userEvent.setup();
        const { zone, input } = renderZone();
        const clickSpy = vi.spyOn(input, 'click');

        zone.focus();
        await user.keyboard('{Enter}');

        expect(clickSpy).toHaveBeenCalledOnce();
    });

    it('reads a dropped file and emits its contents', () => {
        const { zone, onFileSelect } = renderZone();
        const file = makeFile();

        fireDragEvent(zone, 'drop', { dataTransfer: { files: [file] } });

        expect(readAsText).toHaveBeenCalledWith(file);
        expect(onFileSelect).toHaveBeenCalledWith(FILE_CONTENT);
    });

    it('reads a file chosen through the dialog and resets the input', async () => {
        const { input, onFileSelect } = renderZone();

        await userEvent.upload(input, makeFile());

        expect(onFileSelect).toHaveBeenCalledWith(FILE_CONTENT);
        expect(input.value).toBe('');
    });

    it('does not emit anything when no file was chosen', () => {
        const { zone, input, onFileSelect } = renderZone();

        fireDragEvent(zone, 'drop', { dataTransfer: { files: [] } });
        fireEvent.change(input, { target: { files: [] } });

        expect(readAsText).not.toHaveBeenCalled();
        expect(onFileSelect).not.toHaveBeenCalled();
    });
});

describe('Textarea isClearable', () => {
    const renderTextarea = (options: {
        value: string;
        isClearable?: boolean;
        disabled?: boolean;
        onClear?: () => void;
    }) => {
        const [value, setValue] = createSignal(options.value);
        const onChange = vi.fn();

        render(() => (
            <Textarea
                value={value()}
                isClearable={options.isClearable}
                disabled={options.disabled}
                onClear={options.onClear}
                onChange={(e) => {
                    onChange(e);
                    setValue(e.currentTarget.value);
                }}
            />
        ));

        return {
            onChange,
            box: screen.getByRole('textbox') as HTMLTextAreaElement,
            clear: () => screen.getByTestId('textarea-clear-button'),
            hasClear: () => screen.queryByTestId('textarea-clear-button') !== null,
            value,
        };
    };

    it('hides the clear button when the field is empty', () => {
        const { hasClear } = renderTextarea({ value: '', isClearable: true });

        expect(hasClear()).toBe(false);
    });

    it('hides the clear button when isClearable is not set', () => {
        const { hasClear } = renderTextarea({ value: 'pem' });

        expect(hasClear()).toBe(false);
    });

    it('hides the clear button when the field is disabled', () => {
        const { hasClear } = renderTextarea({ value: 'pem', isClearable: true, disabled: true });

        expect(hasClear()).toBe(false);
    });

    it('empties the field and notifies the parent when cleared', () => {
        const onClear = vi.fn();
        const { box, clear, onChange, value } = renderTextarea({
            value: 'pem',
            isClearable: true,
            onClear,
        });
        const button = clear();

        expect(box.value).toBe('pem');
        expect(button.getAttribute('aria-label')).toBe('aria_clear_input');

        fireEvent.click(button);

        expect(value()).toBe('');
        expect(box.value).toBe('');
        expect(onClear).toHaveBeenCalledOnce();
        expect(onChange).toHaveBeenCalledOnce();
        expect(onChange.mock.calls[0][0].currentTarget.value).toBe('');
    });
});

describe('TLS setup wizard dropzones', () => {
    const openWizard = () => {
        const utils = render(() => <TlsSetupWizard open={true} onClose={vi.fn()} />);
        return utils;
    };

    const setTextarea = (selector: string, next: string) => {
        const el = document.querySelector(selector) as HTMLTextAreaElement;
        fireEvent.change(el, { target: { value: next } });
        return el;
    };

    it('shows the certificate dropzone and hides it once the textarea is filled', () => {
        openWizard();

        const box = document.querySelector('#tls_setup_certificate_chain') as HTMLTextAreaElement;
        const zone = screen.getByTestId('tls-setup-cert-dropzone');

        expect(zone).toBeInTheDocument();
        expect(screen.getByText('encryption_cert_dropzone_hint')).toBeInTheDocument();
        // 16px gap between the paste field and the zone (consumer-owned class).
        expect(zone.className).toContain('dropzoneGap');
        // 116px while the zone is visible, 200px once it is gone.
        expect(box.className).toContain('compact');
        expect(box.className).not.toContain('large');

        setTextarea('#tls_setup_certificate_chain', CERT_PEM);

        expect(screen.queryByTestId('tls-setup-cert-dropzone')).not.toBeInTheDocument();
        expect(box.className).toContain('large');
        expect(box.className).not.toContain('compact');
    });

    it('restores the dropzone when the certificate textarea is cleared', () => {
        openWizard();

        const box = setTextarea('#tls_setup_certificate_chain', CERT_PEM);
        expect(box.value).toBe(CERT_PEM);
        expect(box.className).toContain('large');

        fireEvent.click(screen.getByTestId('textarea-clear-button'));

        expect(box.value).toBe('');
        expect(screen.getByTestId('tls-setup-cert-dropzone')).toBeInTheDocument();
        expect(box.className).toContain('compact');
    });

    it('shows the key dropzone on step 2 and hides it once the key is filled', async () => {
        const user = userEvent.setup();
        openWizard();

        setTextarea('#tls_setup_certificate_chain', CERT_PEM);
        await user.click(screen.getByTestId('tls-setup-add'));

        const box = document.querySelector('#tls_setup_private_key') as HTMLTextAreaElement;
        expect(screen.getByTestId('tls-setup-key-dropzone')).toBeInTheDocument();
        expect(box.className).toContain('compact');

        setTextarea('#tls_setup_private_key', KEY_PEM);

        expect(screen.queryByTestId('tls-setup-key-dropzone')).not.toBeInTheDocument();
        expect(box.className).toContain('large');
    });

    it('has no browse suffix on the certificate path input', async () => {
        const user = userEvent.setup();
        openWizard();

        const contentRadio = document.querySelector(
            '#tls_setup_certificate_source-content',
        ) as HTMLInputElement;
        expect(contentRadio.checked).toBe(true);

        const pathRadio = document.querySelector(
            `#tls_setup_certificate_source-${ENCRYPTION_SOURCE.PATH}`,
        ) as HTMLInputElement;
        await user.click(pathRadio);

        expect(document.querySelector('#tls_setup_certificate_path')).toBeInTheDocument();
        expect(screen.queryByText('browse')).not.toBeInTheDocument();
        expect(screen.queryByTestId('tls-setup-cert-dropzone')).not.toBeInTheDocument();
    });
});
