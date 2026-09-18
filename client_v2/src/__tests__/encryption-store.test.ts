import { describe, it, expect, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({
    tlsStatus: vi.fn(),
    tlsConfigure: vi.fn(),
    tlsValidate: vi.fn(),
    addErrorToast: vi.fn(),
    addSuccessToast: vi.fn(),
    redirectToCurrentProtocol: vi.fn(),
}));

vi.mock('panel/api/generated', () => ({
    tlsStatus: mocks.tlsStatus,
    tlsConfigure: mocks.tlsConfigure,
    tlsValidate: mocks.tlsValidate,
}));
vi.mock('panel/stores/toasts', () => ({
    addErrorToast: mocks.addErrorToast,
    addSuccessToast: mocks.addSuccessToast,
}));
vi.mock('panel/stores/dashboard', () => ({
    getDnsStatus: vi.fn(),
    dashboardState: { httpPort: 80 },
}));
vi.mock('panel/helpers/helpers', () => ({
    redirectToCurrentProtocol: mocks.redirectToCurrentProtocol,
}));

import {
    setTlsConfig,
    getTlsStatus,
    validateTlsConfig,
    resetValidationStatus,
    encryptionState,
} from 'panel/stores/encryption';
import { copy } from 'panel/__tests__/helpers/copy';

/**
 * The backend marshals a cleared server name and a port that is turned off with
 * `omitempty`, so they are missing from its response instead of empty.
 */
const RESPONSE_WITHOUT_EMPTIES = {
    enabled: false,
    serve_plain_dns: true,
    certificate_chain: '',
    private_key: '',
    certificate_path: '',
    private_key_path: '',
    private_key_saved: false,
};

/**
 * The status fields the backend also marshals with `omitempty`
 * (`tlsConfigStatus` in `internal/home/tls.go`): a resolved warning and the
 * certificate metadata that came with it are dropped from the response
 * entirely instead of being sent empty.
 */
const STATUS_WITHOUT_EMPTIES = {
    valid_chain: true,
    valid_cert: true,
    valid_key: true,
    valid_pair: true,
    not_before: '2026-01-01T00:00:00Z',
    not_after: '2027-01-01T00:00:00Z',
    dns_names: [] as string[],
};

/** The warning the backend reports when the certificate does not cover the name. */
const SERVER_NAME_MISMATCH_WARNING =
    'certificate does not verify: x509: certificate is valid for wrong.example.com, ' +
    'not adguard.example.com';

/** A response from a check that found the name mismatch. */
const RESPONSE_WITH_WARNING = {
    ...RESPONSE_WITHOUT_EMPTIES,
    ...STATUS_WITHOUT_EMPTIES,
    subject: 'CN=wrong.example.com',
    issuer: 'CN=wrong.example.com',
    key_type: 'RSA',
    warning_validation: SERVER_NAME_MISMATCH_WARNING,
};

/** A response from a check that no longer finds anything wrong. */
const RESPONSE_AFTER_FIX = { ...RESPONSE_WITHOUT_EMPTIES, ...STATUS_WITHOUT_EMPTIES };

/** Asserts that no part of the previously reported warning is left behind. */
const expectValidationStatusCleared = () => {
    expect(encryptionState.warning_validation).toBe('');
    expect(encryptionState.subject).toBe('');
    expect(encryptionState.issuer).toBe('');
    expect(encryptionState.key_type).toBeUndefined();
};

describe('setTlsConfig', () => {
    beforeEach(() => vi.clearAllMocks());

    it('empties a server name and port the backend leaves out of the response', async () => {
        // Seed the store the way a save with values does: the backend echoes
        // them back.
        mocks.tlsConfigure.mockResolvedValue({
            ...RESPONSE_WITHOUT_EMPTIES,
            server_name: 'dns.example.com',
            port_https: 8443,
        });
        await setTlsConfig({ certificate_chain: '', private_key: '' });
        expect(encryptionState.server_name).toBe('dns.example.com');
        expect(encryptionState.port_https).toBe(8443);

        // Clearing them makes the backend drop the fields.  A plain merge would
        // keep the previous values, which is what the form would then show.
        mocks.tlsConfigure.mockResolvedValue({ ...RESPONSE_WITHOUT_EMPTIES });
        await setTlsConfig({ server_name: '', port_https: 0 });

        expect(encryptionState.server_name).toBe('');
        expect(encryptionState.port_https).toBe(0);
        expect(encryptionState.port_dns_over_tls).toBe(0);
        expect(encryptionState.port_dns_over_quic).toBe(0);
    });

    it('defaults empty ports to 0', async () => {
        mocks.tlsConfigure.mockImplementation(async (v: any) => ({
            ...v,
            certificate_chain: '',
            private_key: '',
        }));
        await setTlsConfig({
            certificate_chain: '',
            private_key: '',
            port_https: 0,
            port_dns_over_tls: 0,
            port_dns_over_quic: 0,
        });
        const sent = mocks.tlsConfigure.mock.calls[0][0];
        expect(sent.port_https).toBe(0);
        expect(sent.port_dns_over_tls).toBe(0);
        expect(sent.port_dns_over_quic).toBe(0);
    });

    it('clears validation status fields when resetValidationStatus is called', async () => {
        mocks.tlsValidate.mockResolvedValue({
            valid_chain: true,
            valid_cert: true,
            valid_key: true,
            valid_pair: true,
            subject: 'CN=example.com',
            issuer: 'CN=example.com',
            key_type: 'RSA',
            not_after: '2027-01-01T00:00:00Z',
            not_before: '2026-01-01T00:00:00Z',
            dns_names: ['example.com'],
            warning_validation: 'self-signed certificate',
            certificate_chain: '',
            private_key: '',
        });

        await validateTlsConfig({
            enabled: true,
            certificate_chain: 'x',
            private_key: 'y',
        });

        expect(encryptionState.valid_cert).toBe(true);
        expect(encryptionState.warning_validation).toBe('self-signed certificate');

        resetValidationStatus();

        expect(encryptionState.valid_cert).toBe(false);
        expect(encryptionState.valid_chain).toBe(false);
        expect(encryptionState.valid_key).toBe(false);
        expect(encryptionState.valid_pair).toBe(false);
        expect(encryptionState.warning_validation).toBe('');
        expect(encryptionState.subject).toBe('');
        expect(encryptionState.issuer).toBe('');
        expect(encryptionState.key_type).toBeUndefined();
        expect(encryptionState.dns_names).toBeNull();
    });

    it('calls redirectToCurrentProtocol when enabled+force_https on http: origin', async () => {
        Object.defineProperty(window, 'location', {
            value: { protocol: 'http:' },
            writable: true,
        });
        mocks.tlsConfigure.mockImplementation(async (v: any) => ({
            ...v,
            certificate_chain: '',
            private_key: '',
        }));
        await setTlsConfig({
            certificate_chain: '',
            private_key: '',
            enabled: true,
            force_https: true,
            port_https: 443,
        });
        expect(mocks.redirectToCurrentProtocol).toHaveBeenCalledWith(
            expect.objectContaining({
                enabled: true,
                force_https: true,
                port_https: 443,
            }),
            expect.any(Number),
        );
    });

    it('calls redirectToCurrentProtocol when disabling encryption on https: origin', async () => {
        Object.defineProperty(window, 'location', {
            value: { protocol: 'https:' },
            writable: true,
        });
        mocks.tlsConfigure.mockImplementation(async (v: any) => ({
            ...v,
            certificate_chain: '',
            private_key: '',
        }));
        await setTlsConfig({
            certificate_chain: '',
            private_key: '',
            enabled: false,
        });
        expect(mocks.redirectToCurrentProtocol).toHaveBeenCalledWith(
            expect.objectContaining({ enabled: false }),
            expect.any(Number),
        );
    });

    it('calls redirectToCurrentProtocol when changing port_https on https: origin', async () => {
        Object.defineProperty(window, 'location', {
            value: { protocol: 'https:' },
            writable: true,
        });
        mocks.tlsConfigure.mockImplementation(async (v: any) => ({
            ...v,
            certificate_chain: '',
            private_key: '',
        }));
        await setTlsConfig({
            certificate_chain: '',
            private_key: '',
            enabled: true,
            force_https: true,
            port_https: 8443,
        });
        expect(mocks.redirectToCurrentProtocol).toHaveBeenCalledWith(
            expect.objectContaining({
                enabled: true,
                force_https: true,
                port_https: 8443,
            }),
            expect.any(Number),
        );
    });

    it('resolves { ok: true } on success', async () => {
        mocks.tlsConfigure.mockImplementation(async (v: any) => ({
            ...v,
            certificate_chain: '',
            private_key: '',
        }));

        await expect(setTlsConfig({ certificate_chain: '', private_key: '' })).resolves.toEqual({
            ok: true,
        });
    });

    it('resolves { ok: false, error } and still toasts on failure by default', async () => {
        mocks.tlsConfigure.mockRejectedValue(
            new Error('http://127.0.0.1/control/tls/configure | saving: boom | 500'),
        );

        await expect(setTlsConfig({ certificate_chain: '', private_key: '' })).resolves.toEqual({
            ok: false,
            error: 'saving: boom',
        });
        expect(mocks.addErrorToast).toHaveBeenCalledTimes(1);
        expect(encryptionState.processingConfig).toBe(false);
    });

    it('suppresses the error toast when requested so callers can render it inline', async () => {
        mocks.tlsConfigure.mockRejectedValue(
            new Error('http://127.0.0.1/control/tls/configure | saving: boom | 500'),
        );

        const res = await setTlsConfig(
            { certificate_chain: '', private_key: '' },
            { suppressErrorToast: true },
        );

        expect(res).toEqual({ ok: false, error: 'saving: boom' });
        expect(mocks.addErrorToast).not.toHaveBeenCalled();
    });
});

describe('setTlsConfig — enabling encryption', () => {
    beforeEach(() => vi.clearAllMocks());

    /**
     * Saves `values` on top of a known state and forgets the toast that save
     * produced, so the next assertion only sees the call under test.  The
     * backend echoes the request with the certificate contents stripped.
     */
    const save = async (values: Record<string, unknown>) => {
        mocks.tlsConfigure.mockImplementation(async (v: any) => ({
            ...v,
            certificate_chain: '',
            private_key: '',
        }));
        await setTlsConfig({ certificate_chain: '', private_key: '', ...values });
        mocks.addSuccessToast.mockClear();
    };

    it('confirms a save that turns encryption on with its own toast', async () => {
        await save({ enabled: false });

        await setTlsConfig({ enabled: true });

        expect(mocks.addSuccessToast).toHaveBeenCalledTimes(1);
        expect(mocks.addSuccessToast).toHaveBeenCalledWith(copy('encryption_enabled_toast'));
    });

    it('keeps the generic toast when encryption was already on', async () => {
        await save({ enabled: true });

        await setTlsConfig({ port_https: 8443 });

        expect(mocks.addSuccessToast).toHaveBeenCalledTimes(1);
        expect(mocks.addSuccessToast).toHaveBeenCalledWith('Changes saved');
    });

    it('keeps the generic toast for saves that do not enable encryption', async () => {
        await save({ enabled: false });

        await setTlsConfig({ port_https: 8443 });

        expect(mocks.addSuccessToast).toHaveBeenCalledTimes(1);
        expect(mocks.addSuccessToast).toHaveBeenCalledWith('Changes saved');
    });

    it('stays silent about the enable when the caller asks for it', async () => {
        await save({ enabled: false });

        await setTlsConfig({ enabled: true }, { silent: true });

        expect(mocks.addSuccessToast).not.toHaveBeenCalled();
        expect(encryptionState.enabled).toBe(true);
    });
});

describe('getTlsStatus', () => {
    beforeEach(() => vi.clearAllMocks());

    it('empties the settings the backend leaves out of the status response', async () => {
        mocks.tlsConfigure.mockResolvedValue({
            ...RESPONSE_WITHOUT_EMPTIES,
            server_name: 'dns.example.com',
            port_https: 8443,
        });
        await setTlsConfig({ certificate_chain: '', private_key: '' });

        // The status response reports the same cleared settings the save did.
        mocks.tlsStatus.mockResolvedValue({ ...RESPONSE_WITHOUT_EMPTIES });
        await getTlsStatus();

        expect(encryptionState.server_name).toBe('');
        expect(encryptionState.port_https).toBe(0);
    });

    it('clears the warning the status response omits', async () => {
        mocks.tlsStatus.mockResolvedValue(RESPONSE_WITH_WARNING);
        await getTlsStatus();
        expect(encryptionState.warning_validation).toBe(SERVER_NAME_MISMATCH_WARNING);

        mocks.tlsStatus.mockResolvedValue(RESPONSE_AFTER_FIX);
        await getTlsStatus();

        expectValidationStatusCleared();
    });
});

describe('stale validation status', () => {
    beforeEach(() => vi.clearAllMocks());

    it('validateTlsConfig clears the warning a corrected response omits', async () => {
        mocks.tlsValidate.mockResolvedValue(RESPONSE_WITH_WARNING);
        await validateTlsConfig({ enabled: true, server_name: 'adguard.example.com' });

        expect(encryptionState.warning_validation).toBe(SERVER_NAME_MISMATCH_WARNING);
        expect(encryptionState.subject).toBe('CN=wrong.example.com');

        // The name matches the certificate again, so the backend drops the
        // warning and the metadata behind it from the response.
        mocks.tlsValidate.mockResolvedValue(RESPONSE_AFTER_FIX);
        await validateTlsConfig({ enabled: true, server_name: 'wrong.example.com' });

        expectValidationStatusCleared();
    });

    it('setTlsConfig clears the warning a corrected save omits', async () => {
        // The settings modal saves the corrected name, and the save echoes the
        // validation status back.
        mocks.tlsConfigure.mockResolvedValue(RESPONSE_WITH_WARNING);
        await setTlsConfig({ server_name: 'adguard.example.com' });

        expect(encryptionState.warning_validation).toBe(SERVER_NAME_MISMATCH_WARNING);

        mocks.tlsConfigure.mockResolvedValue(RESPONSE_AFTER_FIX);
        await setTlsConfig({ server_name: 'wrong.example.com' });

        expectValidationStatusCleared();
    });
});

describe('validateTlsConfig', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // The store is shared by the whole file, and this describe asserts on
        // what `persist: false` leaves behind, so the starting point has to be
        // stated rather than inherited from an earlier test.
        resetValidationStatus();
    });

    it('persist:false returns the decoded config without touching the store or toasts', async () => {
        mocks.tlsValidate.mockResolvedValue({
            valid_chain: true,
            valid_cert: true,
            valid_key: true,
            valid_pair: true,
            subject: 'CN=example.com',
            warning_validation: '',
            certificate_chain: '',
            private_key: '',
        });

        const res = await validateTlsConfig({ enabled: true }, { persist: false });

        expect('error' in res).toBe(false);
        if ('error' in res) return;
        expect(res.valid_cert).toBe(true);
        expect(res.valid_pair).toBe(true);
        expect(encryptionState.valid_cert).toBe(false);
        expect(encryptionState.processingValidate).toBe(false);
        expect(mocks.addErrorToast).not.toHaveBeenCalled();
        expect(mocks.addSuccessToast).not.toHaveBeenCalled();
    });

    it('persist:false returns { error } with the body text on 400 without toasts', async () => {
        mocks.tlsValidate.mockRejectedValue(
            new Error(
                'http://127.0.0.1/control/tls/validate | port 443 for HTTPS is not available | 400',
            ),
        );

        const res = await validateTlsConfig({ enabled: true }, { persist: false });

        expect(res).toEqual({ error: 'port 443 for HTTPS is not available' });
        expect(encryptionState.processingValidate).toBe(false);
        expect(mocks.addErrorToast).not.toHaveBeenCalled();
    });

    it('persist:false keeps newline-joined bodies from errors.Join', async () => {
        mocks.tlsValidate.mockRejectedValue(
            new Error(
                'http://127.0.0.1/control/tls/validate | port 443 for HTTPS is not available\nport 853 for DNS-over-TLS is not available | 400',
            ),
        );

        const res = await validateTlsConfig({ enabled: true }, { persist: false });

        expect(res).toEqual({
            error: 'port 443 for HTTPS is not available\nport 853 for DNS-over-TLS is not available',
        });
    });

    it('persist:true (default) shows an error toast on failure and clears processingValidate', async () => {
        mocks.tlsValidate.mockRejectedValue(
            new Error('http://127.0.0.1/control/tls/validate | plain DNS is required | 400'),
        );

        await validateTlsConfig({ enabled: true });

        expect(mocks.addErrorToast).toHaveBeenCalledTimes(1);
        expect(encryptionState.processingValidate).toBe(false);
    });
});
