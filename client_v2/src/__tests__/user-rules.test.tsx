import React from 'react';
import { render, screen, waitFor, within, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { UserRules } from 'panel/components/UserRules/UserRules';
import {
    BLOCK_ACTIONS,
    FILTERED_STATUS,
    MODAL_TYPE,
    SETTINGS_NAMES,
} from 'panel/helpers/constants';
import { initialState, RootState } from 'panel/initialState';

const mocks = vi.hoisted(() => ({
    state: null as unknown as RootState,
    dispatch: vi.fn((action) => action),
    getFilteringStatus: vi.fn(() => ({ type: 'getFilteringStatus' })),
    getClients: vi.fn(() => ({ type: 'getClients' })),
    setRules: vi.fn((rules) => ({ type: 'setRules', payload: rules })),
    checkHost: vi.fn((payload) => ({ type: 'checkHost', payload })),
    toggleFilterStatus: vi.fn((url, data, whitelist) =>
        Promise.resolve({ type: 'toggleFilterStatus', payload: { url, data, whitelist } }),
    ),
    initSettings: vi.fn(() => ({ type: 'initSettings' })),
    toggleBlocking: vi.fn((type, domain) => ({
        type: 'toggleBlocking',
        payload: { type, domain },
    })),
    toggleBlockingForClient: vi.fn((type, domain, client) => ({
        type: 'toggleBlockingForClient',
        payload: { type, domain, client },
    })),
    toggleSetting: vi.fn(() => Promise.resolve(true)),
    getRewritesList: vi.fn(() => ({ type: 'getRewritesList' })),
    updateRewrite: vi.fn((payload, options) =>
        Promise.resolve({ type: 'updateRewrite', payload, options }),
    ),
    deleteRewrite: vi.fn((payload, options) =>
        Promise.resolve({ type: 'deleteRewrite', payload, options }),
    ),
    updateClient: vi.fn((payload, name, options) =>
        Promise.resolve({ type: 'updateClient', payload, name, options }),
    ),
    getBlockedServices: vi.fn(() => ({ type: 'getBlockedServices' })),
    getAllBlockedServices: vi.fn(() => ({ type: 'getAllBlockedServices' })),
    updateBlockedServices: vi.fn((payload, options) =>
        Promise.resolve({ type: 'updateBlockedServices', payload, options }),
    ),
    addSuccessToast: vi.fn((message) => ({ type: 'addSuccessToast', payload: message })),
}));

vi.mock('react-redux', () => ({
    batch: (fn: () => void) => fn(),
    useDispatch: () => mocks.dispatch,
    useSelector: (selector: (state: RootState) => unknown) => selector(mocks.state),
}));

vi.mock('panel/actions/filtering', () => ({
    getFilteringStatus: mocks.getFilteringStatus,
    setRules: mocks.setRules,
    checkHost: mocks.checkHost,
    toggleFilterStatus: mocks.toggleFilterStatus,
}));

vi.mock('panel/actions', () => ({
    getClients: mocks.getClients,
    initSettings: mocks.initSettings,
    toggleBlocking: mocks.toggleBlocking,
    toggleBlockingForClient: mocks.toggleBlockingForClient,
    toggleSetting: mocks.toggleSetting,
}));

vi.mock('panel/actions/clients', () => ({
    updateClient: mocks.updateClient,
}));

vi.mock('panel/actions/rewrites', () => ({
    getRewritesList: mocks.getRewritesList,
    updateRewrite: mocks.updateRewrite,
    deleteRewrite: mocks.deleteRewrite,
}));

vi.mock('panel/actions/services', () => ({
    getBlockedServices: mocks.getBlockedServices,
    getAllBlockedServices: mocks.getAllBlockedServices,
    updateBlockedServices: mocks.updateBlockedServices,
}));

vi.mock('panel/actions/toasts', () => ({
    addSuccessToast: mocks.addSuccessToast,
    createUndoToast: (message: any, actionLabel: any) => ({
        message,
        actionLabel,
        undoId: 'mock-undo-id',
    }),
}));

vi.mock('panel/helpers/helpers', async (importOriginal) => {
    const actual = await importOriginal<typeof import('panel/helpers/helpers')>();

    return {
        ...actual,
        delay: () => Promise.resolve(),
    };
});

vi.mock(
    'panel/components/FilterLists/blocks/ConfigureRewritesModal/ConfigureRewritesModal',
    async () => {
        const React = await import('react');

        return {
            ConfigureRewritesModal: ({ modalId, rewriteToEdit, onSubmit }: any) => {
                const [domain, setDomain] = React.useState(rewriteToEdit?.domain ?? '');
                const [answer, setAnswer] = React.useState(rewriteToEdit?.answer ?? '');

                React.useEffect(() => {
                    setDomain(rewriteToEdit?.domain ?? '');
                    setAnswer(rewriteToEdit?.answer ?? '');
                }, [rewriteToEdit]);

                if (mocks.state.modals.modalId !== modalId) {
                    return null;
                }

                return (
                    <div role="dialog">
                        <input
                            data-testid="rewrite-domain-input"
                            value={domain}
                            onChange={(event) => setDomain(event.currentTarget.value)}
                        />
                        <input
                            data-testid="rewrite-answer-input"
                            value={answer}
                            onChange={(event) => setAnswer(event.currentTarget.value)}
                        />
                        <button
                            type="button"
                            data-testid="rewrite-save-button"
                            onClick={() =>
                                onSubmit?.({
                                    domain,
                                    answer,
                                    enabled: rewriteToEdit?.enabled ?? false,
                                })
                            }
                        >
                            Save
                        </button>
                    </div>
                );
            },
        };
    },
);

vi.mock('panel/components/FilterLists/blocks/DeleteRewriteModal', () => ({
    DeleteRewriteModal: ({ rewriteToDelete, onConfirm }: any) => {
        if (mocks.state.modals.modalId !== MODAL_TYPE.DELETE_REWRITE || !rewriteToDelete?.domain) {
            return null;
        }

        return (
            <button
                type="button"
                data-testid="rewrite-delete-confirm"
                onClick={() => onConfirm?.()}
            >
                Remove
            </button>
        );
    },
}));

type RenderOptions = {
    dashboard?: Partial<RootState['dashboard']>;
    filtering?: Partial<RootState['filtering']>;
    settings?: Partial<RootState['settings']>;
    rewrites?: Partial<RootState['rewrites']>;
    services?: Partial<RootState['services']>;
};

type CheckResult = NonNullable<RootState['filtering']['check']>;
type PersistentClient = NonNullable<RootState['dashboard']>['clients'][number];

const EXAMPLE_FILTER = {
    id: 101,
    name: 'Example Blocklist',
    url: 'https://filters.example/blocklist.txt',
    enabled: true,
    lastUpdated: '',
    rulesCount: 12,
};

const EXAMPLE_ALLOWLIST = {
    id: 201,
    name: 'Example Allowlist',
    url: 'https://filters.example/allowlist.txt',
    enabled: true,
    lastUpdated: '',
    rulesCount: 7,
};

const MATCHED_REWRITE = {
    domain: 'rewrite.example',
    answer: 'target.example',
    enabled: true,
};

const createPersistentClient = (overrides: Partial<PersistentClient> = {}): PersistentClient => ({
    blocked_services: [],
    blocked_services_schedule: { time_zone: 'UTC' },
    filtering_enabled: false,
    ids: ['office-laptop'],
    ignore_querylog: false,
    ignore_statistics: false,
    name: 'office-laptop',
    parental_enabled: false,
    safe_search: { enabled: false },
    safebrowsing_enabled: false,
    safesearch_enabled: false,
    tags: [],
    upstreams: [],
    upstreams_cache_enabled: false,
    upstreams_cache_size: 0,
    use_global_blocked_services: true,
    use_global_settings: true,
    ...overrides,
});

const createState = (overrides: RenderOptions = {}): RootState => ({
    ...initialState,
    dashboard: {
        ...initialState.dashboard,
        ...overrides.dashboard,
    },
    filtering: {
        ...initialState.filtering,
        ...overrides.filtering,
    },
    settings: {
        ...initialState.settings,
        settingsList: {
            parental: { enabled: true },
            safebrowsing: { enabled: true },
            safesearch: { enabled: true },
            ...(overrides.settings?.settingsList || {}),
        },
        ...overrides.settings,
    },
    rewrites: {
        ...initialState.rewrites,
        ...overrides.rewrites,
    },
    services: {
        ...initialState.services,
        list: {
            ids: [],
            ...(overrides.services?.list || {}),
        },
        ...overrides.services,
    },
});

const renderUserRules = (overrides: RenderOptions = {}) => {
    mocks.state = createState(overrides);

    return render(<UserRules />);
};

const createCheckResult = (overrides: Partial<CheckResult> = {}): CheckResult =>
    ({
        hostname: 'example.test',
        reason: FILTERED_STATUS.NOT_FILTERED_NOT_FOUND,
        rules: [],
        ...overrides,
    }) as CheckResult;

const renderCheckResult = (check: Partial<CheckResult>, overrides: RenderOptions = {}) =>
    renderUserRules({
        ...overrides,
        filtering: {
            ...overrides.filtering,
            check: createCheckResult(check),
        },
    });

const renderMatchedFilterResult = (hostname = 'filtered.example') =>
    renderCheckResult(
        {
            hostname,
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules: [{ filter_list_id: EXAMPLE_FILTER.id, text: `||${hostname}^` }],
        },
        {
            filtering: {
                filters: [EXAMPLE_FILTER],
            },
        },
    );

const renderMatchedAllowlistResult = (hostname = 'allowed.example') =>
    renderCheckResult(
        {
            hostname,
            reason: FILTERED_STATUS.NOT_FILTERED_WHITE_LIST,
            rules: [{ filter_list_id: EXAMPLE_ALLOWLIST.id, text: `@@||${hostname}^$important` }],
        },
        {
            filtering: {
                whitelistFilters: [EXAMPLE_ALLOWLIST],
            },
        },
    );

const renderMatchedRewriteResult = () =>
    renderCheckResult(
        {
            hostname: MATCHED_REWRITE.domain,
            reason: FILTERED_STATUS.REWRITE,
            cname: MATCHED_REWRITE.answer,
        },
        {
            rewrites: {
                list: [MATCHED_REWRITE],
            },
        },
    );

const getBootstrapDispatchTypes = () => mocks.dispatch.mock.calls.map(([action]) => action?.type);

const submitCheckForm = async (
    user: ReturnType<typeof userEvent.setup>,
    options: { hostname: string; client?: string; qtype?: string },
) => {
    const qtype = options.qtype ?? 'A';

    fireEvent.change(screen.getByTestId('user-rules-check-hostname'), {
        target: { value: options.hostname },
    });

    if (options.client) {
        fireEvent.change(screen.getByTestId('user-rules-check-client'), {
            target: { value: options.client },
        });
    }

    if (qtype !== 'A') {
        await user.click(screen.getByLabelText('DNS record type'));
        await user.click(within(screen.getByRole('listbox')).getByText(qtype));
    }

    await user.click(screen.getByTestId('user-rules-check-submit'));
};

const expectRecheck = (hostname: string, client?: string, qtype = 'A') => {
    expect(mocks.checkHost).toHaveBeenCalledWith({
        name: hostname,
        client,
        qtype,
    });
};

const resultActionScenarios = [
    {
        name: 'custom filtering blocks',
        renderScenario: () =>
            renderCheckResult({
                hostname: 'blocked.example',
                reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
                rules: [{ filter_list_id: 0, text: '||blocked.example^' }],
            }),
        title: 'Domain is blocked',
        actions: [['allow', 'Add to allowlist']],
    },
    {
        name: 'parental results',
        renderScenario: () =>
            renderCheckResult({
                hostname: 'adult.example',
                reason: FILTERED_STATUS.FILTERED_PARENTAL,
            }),
        actions: [
            ['allow', 'Add to allowlist'],
            ['disable-parental', 'Disable Parental control'],
        ],
    },
    {
        name: 'non-custom filter blocks',
        renderScenario: () => renderMatchedFilterResult(),
        actions: [
            ['allow', 'Add to allowlist'],
            ['disable-filter', 'Disable filter'],
        ],
    },
    {
        name: 'processed results',
        renderScenario: () =>
            renderCheckResult({
                hostname: 'plain.example',
                reason: FILTERED_STATUS.NOT_FILTERED_NOT_FOUND,
            }),
        title: 'Domain is processed',
        description: 'No rules matched',
        rejectsObjectObject: true,
        actions: [
            ['allow', 'Add to allowlist'],
            ['block', 'Block'],
        ],
    },
    {
        name: 'custom allowed results',
        renderScenario: () =>
            renderCheckResult({
                hostname: 'allowed.example',
                reason: FILTERED_STATUS.NOT_FILTERED_WHITE_LIST,
                rules: [{ filter_list_id: 0, text: '@@||allowed.example^$important' }],
            }),
        title: 'Domain is allowed',
        actions: [['block', 'Block']],
    },
    {
        name: 'allowlist filter allowed results',
        renderScenario: () => renderMatchedAllowlistResult(),
        title: 'Domain is allowed',
        actions: [['disable-filter', 'Disable filter']],
    },
];

const settingToggleScenarios = [
    {
        name: 'safe browsing',
        actionKind: 'disable-safebrowsing',
        hostname: 'malware.example',
        reason: FILTERED_STATUS.FILTERED_SAFE_BROWSING,
        settingKey: SETTINGS_NAMES.safebrowsing,
        expectedSettingValue: true,
        toast: expect.objectContaining({
            message: 'Browsing security disabled',
            actionLabel: 'Undo',
        }),
    },
    {
        name: 'parental control',
        actionKind: 'disable-parental',
        hostname: 'adult.example',
        reason: FILTERED_STATUS.FILTERED_PARENTAL,
        settingKey: SETTINGS_NAMES.parental,
        expectedSettingValue: true,
        toast: expect.objectContaining({
            message: 'Parental control disabled',
            actionLabel: 'Undo',
        }),
    },
    {
        name: 'safe search',
        actionKind: 'disable-safesearch',
        hostname: 'search.example',
        reason: FILTERED_STATUS.FILTERED_SAFE_SEARCH,
        settingKey: SETTINGS_NAMES.safesearch,
        expectedSettingValue: expect.objectContaining({ enabled: false }),
        toast: expect.objectContaining({ message: 'Safe search disabled', actionLabel: 'Undo' }),
    },
];

beforeEach(() => {
    mocks.state = createState();
    mocks.dispatch.mockReset();
    mocks.dispatch.mockImplementation((action) => {
        if (action?.type === 'OPEN_MODAL') {
            mocks.state = {
                ...mocks.state,
                modals: {
                    ...mocks.state.modals,
                    modalId: action.payload.modalId,
                },
            };
        }

        if (action?.type === 'CLOSE_MODAL') {
            mocks.state = {
                ...mocks.state,
                modals: {
                    ...mocks.state.modals,
                    modalId: null,
                },
            };
        }

        return action;
    });

    [
        mocks.getFilteringStatus,
        mocks.getClients,
        mocks.setRules,
        mocks.checkHost,
        mocks.toggleFilterStatus,
        mocks.initSettings,
        mocks.toggleBlocking,
        mocks.toggleBlockingForClient,
        mocks.toggleSetting,
        mocks.getRewritesList,
        mocks.updateRewrite,
        mocks.deleteRewrite,
        mocks.updateClient,
        mocks.getBlockedServices,
        mocks.getAllBlockedServices,
        mocks.updateBlockedServices,
        mocks.addSuccessToast,
    ].forEach((mock) => mock.mockClear());

    mocks.toggleFilterStatus.mockImplementation((url, data, whitelist) =>
        Promise.resolve({ type: 'toggleFilterStatus', payload: { url, data, whitelist } }),
    );
    mocks.toggleSetting.mockResolvedValue(true);
    mocks.updateRewrite.mockImplementation((payload, options) =>
        Promise.resolve({ type: 'updateRewrite', payload, options }),
    );
    mocks.deleteRewrite.mockImplementation((payload, options) =>
        Promise.resolve({ type: 'deleteRewrite', payload, options }),
    );
    mocks.updateClient.mockImplementation((payload, name, options) =>
        Promise.resolve({ type: 'updateClient', payload, name, options }),
    );
    mocks.updateBlockedServices.mockImplementation((payload, options) =>
        Promise.resolve({ type: 'updateBlockedServices', payload, options }),
    );
});

describe('UserRules harness', () => {
    it('submits hostname, client, and qtype from the check form', async () => {
        const user = userEvent.setup();

        renderUserRules();

        await user.type(screen.getByLabelText('Hostname or domain name'), 'qtype.example');
        await user.type(
            screen.getByLabelText('Client identifier (name, ClientID, or IP address)'),
            'office-laptop',
        );
        await user.click(screen.getByLabelText('DNS record type'));
        await user.click(screen.getByText('CNAME'));
        await user.click(screen.getByRole('button', { name: 'Check' }));

        expect(mocks.checkHost).toHaveBeenCalledWith({
            name: 'qtype.example',
            client: 'office-laptop',
            qtype: 'CNAME',
        });
    });

    it('uses the default qtype when submitting the check form', async () => {
        const user = userEvent.setup();

        renderUserRules();

        await submitCheckForm(user, { hostname: 'required.example' });

        expect(screen.getByTestId('user-rules-check-submit')).toBeEnabled();
        expect(mocks.checkHost).toHaveBeenCalledWith({
            name: 'required.example',
            client: undefined,
            qtype: 'A',
        });
    });

    it('shows a result loader instead of stale result content while a new check is pending', async () => {
        const user = userEvent.setup();
        let resolveCheck: (() => void) | undefined;
        const pendingCheck = new Promise<void>((resolve) => {
            resolveCheck = resolve;
        });

        mocks.dispatch.mockImplementation((action) => {
            if (action?.type === 'checkHost') {
                return pendingCheck;
            }

            return action;
        });

        renderCheckResult({ hostname: 'old.example' });

        await submitCheckForm(user, { hostname: 'new.example' });

        expect(screen.getByTestId('user-rules-result-loader')).toBeInTheDocument();
        expect(screen.queryByTestId('user-rules-result-card')).not.toBeInTheDocument();

        resolveCheck?.();
    });

    it('bootstraps supporting state and hides the result card after dismiss', async () => {
        const user = userEvent.setup();

        renderCheckResult({ hostname: 'example.com' });

        await waitFor(() => {
            expect(getBootstrapDispatchTypes()).toEqual([
                'getFilteringStatus',
                'initSettings',
                'getClients',
                'getRewritesList',
                'getBlockedServices',
                'getAllBlockedServices',
            ]);
        });

        await user.click(screen.getByRole('button', { name: 'Close result' }));

        expect(screen.queryByText('Domain is processed')).not.toBeInTheDocument();
    });

    resultActionScenarios.forEach(
        ({ name, renderScenario, title, description, rejectsObjectObject, actions }) => {
            it(`renders expected result actions for ${name}`, () => {
                renderScenario();

                if (title) {
                    expect(screen.getByText(title)).toBeInTheDocument();
                }

                if (description) {
                    expect(screen.getByText(description)).toBeInTheDocument();
                }

                if (rejectsObjectObject) {
                    expect(screen.queryByText(/\[object Object\]/)).not.toBeInTheDocument();
                }

                actions.forEach(([actionKind, label]) => {
                    expect(
                        screen.getByTestId(`user-rules-result-action-${actionKind}`),
                    ).toHaveTextContent(label);
                });
            });
        },
    );

    it('does not show qtype or source rows in the result details', async () => {
        const user = userEvent.setup();

        renderMatchedFilterResult();

        await submitCheckForm(user, { hostname: 'filtered.example', qtype: 'A' });

        const resultCard = screen.getByTestId('user-rules-result-card');

        expect(within(resultCard).queryByText(/DNS record type:/)).not.toBeInTheDocument();
        expect(within(resultCard).queryByText(/Source:/)).not.toBeInTheDocument();
    });

    it('does not show a reason row when a matched filter has no source name', () => {
        renderCheckResult({
            hostname: 'filtered.example',
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules: [{ filter_list_id: 999, text: '||filtered.example^' }],
        });

        const resultCard = screen.getByTestId('user-rules-result-card');

        expect(
            within(resultCard).queryByText('Reason:', { selector: 'strong' }),
        ).not.toBeInTheDocument();
    });

    it('uses the client-specific toggle when allowlisting a client-specific block', async () => {
        const user = userEvent.setup();

        renderCheckResult({
            hostname: 'blocked.example',
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules: [{ filter_list_id: 0, text: '||blocked.example^' }],
        });

        await submitCheckForm(user, {
            hostname: 'blocked.example',
            client: 'office-laptop',
        });

        mocks.checkHost.mockClear();
        await user.click(screen.getByRole('button', { name: 'Add to allowlist' }));

        expect(mocks.toggleBlockingForClient).toHaveBeenCalledWith(
            BLOCK_ACTIONS.UNBLOCK,
            'blocked.example',
            'office-laptop',
        );
        expect(mocks.toggleBlocking).not.toHaveBeenCalled();
        expectRecheck('blocked.example', 'office-laptop');
    });

    it('updates the resolved persistent client when disabling Safe Browsing for a client-scoped result', async () => {
        const user = userEvent.setup();

        renderCheckResult(
            {
                hostname: 'malware.example',
                reason: FILTERED_STATUS.FILTERED_SAFE_BROWSING,
            },
            {
                dashboard: {
                    clients: [createPersistentClient()],
                },
                filtering: {
                    enabled: true,
                },
                settings: {
                    settingsList: {
                        parental: { enabled: true },
                        safebrowsing: { enabled: true },
                        safesearch: { enabled: true, google: true },
                    },
                },
            },
        );

        await submitCheckForm(user, {
            hostname: 'malware.example',
            client: 'office-laptop',
        });

        mocks.checkHost.mockClear();
        await user.click(screen.getByTestId('user-rules-result-action-disable-safebrowsing'));

        expect(mocks.updateClient).toHaveBeenCalledWith(
            expect.objectContaining({
                name: 'office-laptop',
                use_global_settings: false,
                filtering_enabled: true,
                parental_enabled: true,
                safebrowsing_enabled: false,
                safe_search: expect.objectContaining({ enabled: true, google: true }),
                safesearch_enabled: true,
            }),
            'office-laptop',
            { showToast: false, toggleModal: false },
        );
        expect(mocks.toggleSetting).not.toHaveBeenCalled();
        expect(mocks.addSuccessToast).toHaveBeenCalledWith(
            expect.objectContaining({ message: 'Browsing security disabled', actionLabel: 'Undo' }),
        );
        expectRecheck('malware.example', 'office-laptop');
    });

    it('hides client-scoped settings actions when the checked client cannot be resolved', async () => {
        const user = userEvent.setup();

        renderCheckResult({ hostname: 'adult.example', reason: FILTERED_STATUS.FILTERED_PARENTAL });

        await submitCheckForm(user, {
            hostname: 'adult.example',
            client: 'unknown-client',
        });

        expect(
            screen.queryByTestId('user-rules-result-action-disable-parental'),
        ).not.toBeInTheDocument();
        expect(screen.getByTestId('user-rules-result-action-allow')).toBeInTheDocument();
    });

    it('passes the matched custom allow rule when blocking a custom allowed result', async () => {
        const user = userEvent.setup();

        renderCheckResult({
            hostname: 'allowed.example',
            reason: FILTERED_STATUS.NOT_FILTERED_WHITE_LIST,
            rules: [{ filter_list_id: 0, text: '@@allowed.example^$important' }],
        });

        await submitCheckForm(user, { hostname: 'allowed.example' });

        mocks.checkHost.mockClear();
        await user.click(screen.getByTestId('user-rules-result-action-block'));

        expect(mocks.toggleBlocking).toHaveBeenCalledWith(
            BLOCK_ACTIONS.BLOCK,
            'allowed.example',
            undefined,
            undefined,
            '@@allowed.example^$important',
        );
        expectRecheck('allowed.example');
    });

    it('renders Safe Search as a rewritten result with redirected target details', () => {
        renderCheckResult({
            hostname: 'google.com',
            reason: FILTERED_STATUS.FILTERED_SAFE_SEARCH,
            cname: 'forcesafesearch.google.com',
        });

        const resultCard = screen.getByTestId('user-rules-result-card');

        expect(within(resultCard).getByText('Rewrite rule is applied')).toBeInTheDocument();
        expect(within(resultCard).getByText('Status:', { selector: 'strong' })).toBeInTheDocument();
        expect(within(resultCard).getByText('Safe search')).toBeInTheDocument();
        expect(
            within(resultCard).getByText('Redirected to:', { selector: 'strong' }),
        ).toBeInTheDocument();
        expect(within(resultCard).getByText('forcesafesearch.google.com')).toBeInTheDocument();
        expect(
            within(resultCard).queryByText('CNAME:', { selector: 'strong' }),
        ).not.toBeInTheDocument();
        expect(screen.getByTestId('user-rules-result-action-allow')).toHaveTextContent(
            'Add to allowlist',
        );
        expect(screen.getByTestId('user-rules-result-action-disable-safesearch')).toHaveTextContent(
            'Disable Safe Search',
        );
    });

    it('renders Safe Browsing with blocked threats reason, source, and rule details', () => {
        renderCheckResult({
            hostname: 'malware.example',
            reason: FILTERED_STATUS.FILTERED_SAFE_BROWSING,
            rules: [{ filter_list_id: -4, text: 'adguard-malware-shavar' }],
        });

        const resultCard = screen.getByTestId('user-rules-result-card');

        expect(within(resultCard).getByText('Domain:', { selector: 'strong' })).toBeInTheDocument();
        expect(within(resultCard).getByText('malware.example')).toBeInTheDocument();
        expect(within(resultCard).getByText('Reason:', { selector: 'strong' })).toBeInTheDocument();
        expect(within(resultCard).getByText('Blocked threats')).toBeInTheDocument();
        expect(within(resultCard).getByText('Source:', { selector: 'strong' })).toBeInTheDocument();
        expect(within(resultCard).getByText('Safe Browsing')).toBeInTheDocument();
        expect(within(resultCard).getByText('Rule:', { selector: 'strong' })).toBeInTheDocument();
        expect(within(resultCard).getByText('adguard-malware-shavar')).toBeInTheDocument();
    });

    it('closes the result while a result action is pending', async () => {
        const user = userEvent.setup();
        let resolveToggle: (() => void) | undefined;
        const pendingToggle = new Promise<void>((resolve) => {
            resolveToggle = resolve;
        });

        renderCheckResult({
            hostname: 'plain.example',
            reason: FILTERED_STATUS.NOT_FILTERED_NOT_FOUND,
        });

        await submitCheckForm(user, { hostname: 'plain.example' });

        mocks.dispatch.mockImplementation((action) => {
            if (action?.type === 'toggleBlocking') {
                return pendingToggle;
            }

            return action;
        });

        await user.click(screen.getByTestId('user-rules-result-action-block'));

        expect(screen.queryByTestId('user-rules-result-loader')).not.toBeInTheDocument();
        expect(screen.queryByTestId('user-rules-result-card')).not.toBeInTheDocument();

        resolveToggle?.();
    });

    it('shows only Disable filter for allowlist filter results', () => {
        renderMatchedAllowlistResult();

        const actionOrder = Array.from(
            screen
                .getByTestId('user-rules-result-card')
                .querySelectorAll('[data-testid^="user-rules-result-action-"]'),
        ).map((element) => element.getAttribute('data-testid'));

        expect(actionOrder).toEqual(['user-rules-result-action-disable-filter']);
    });

    settingToggleScenarios.forEach(
        ({ name, actionKind, hostname, reason, settingKey, expectedSettingValue, toast }) => {
            it(`disables ${name} from result actions and rechecks the target`, async () => {
                const user = userEvent.setup();

                renderCheckResult({ hostname, reason });

                await submitCheckForm(user, { hostname });
                mocks.checkHost.mockClear();

                await user.click(screen.getByTestId(`user-rules-result-action-${actionKind}`));

                expect(mocks.toggleSetting).toHaveBeenCalledWith(settingKey, expectedSettingValue);
                expect(mocks.addSuccessToast).toHaveBeenCalledWith(toast);
                expectRecheck(hostname);
            });
        },
    );

    it('disables the matched filter and rechecks the host', async () => {
        const user = userEvent.setup();

        renderMatchedFilterResult();

        await submitCheckForm(user, { hostname: 'filtered.example', qtype: 'A' });
        mocks.checkHost.mockClear();

        await user.click(screen.getByTestId('user-rules-result-action-disable-filter'));

        expect(mocks.toggleFilterStatus).toHaveBeenCalledWith(
            'https://filters.example/blocklist.txt',
            {
                name: EXAMPLE_FILTER.name,
                url: EXAMPLE_FILTER.url,
                enabled: false,
            },
            false,
        );
        const toastPayload = mocks.addSuccessToast.mock.calls.at(-1)?.[0];
        render(<>{toastPayload.message}</>);
        expect(screen.getByText('Example Blocklist', { selector: 'strong' })).toBeInTheDocument();
        expectRecheck('filtered.example');
    });

    it('does not toast or recheck when disabling a matched filter fails', async () => {
        const user = userEvent.setup();

        mocks.toggleFilterStatus.mockResolvedValue(false as never);
        renderMatchedFilterResult();

        await submitCheckForm(user, { hostname: 'filtered.example', qtype: 'A' });
        mocks.checkHost.mockClear();

        await user.click(screen.getByTestId('user-rules-result-action-disable-filter'));

        expect(mocks.addSuccessToast).not.toHaveBeenCalled();
        expect(mocks.checkHost).not.toHaveBeenCalled();
    });

    it('disables the matched allowlist and rechecks the host', async () => {
        const user = userEvent.setup();

        renderMatchedAllowlistResult();

        await submitCheckForm(user, { hostname: 'allowed.example', qtype: 'A' });
        mocks.checkHost.mockClear();

        await user.click(screen.getByTestId('user-rules-result-action-disable-filter'));

        expect(mocks.toggleFilterStatus).toHaveBeenCalledWith(
            'https://filters.example/allowlist.txt',
            {
                name: EXAMPLE_ALLOWLIST.name,
                url: EXAMPLE_ALLOWLIST.url,
                enabled: false,
            },
            true,
        );
        expectRecheck('allowed.example');
    });

    it('allows the blocked service with the user-rules toast copy', async () => {
        const user = userEvent.setup();

        renderCheckResult(
            {
                hostname: 'video.example',
                reason: FILTERED_STATUS.FILTERED_BLOCKED_SERVICE,
                service_name: 'youtube',
                rules: [{ filter_list_id: 0, text: '||amemv.com^' }],
            },
            {
                services: {
                    list: {
                        ids: ['youtube'],
                    },
                    allServices: [{ id: 'youtube', name: 'YouTube', rules: ['||amemv.com^'] }],
                },
            },
        );

        await submitCheckForm(user, {
            hostname: 'video.example',
        });

        expect(screen.getByText('Reason:', { selector: 'strong' })).toBeInTheDocument();
        expect(screen.getByText('Blocked services')).toBeInTheDocument();
        expect(screen.getByText('Service:', { selector: 'strong' })).toBeInTheDocument();
        expect(screen.getByText('YouTube')).toBeInTheDocument();
        expect(screen.getByText('Rule:', { selector: 'strong' })).toBeInTheDocument();
        expect(screen.getByText('||amemv.com^')).toBeInTheDocument();

        mocks.checkHost.mockClear();
        await user.click(screen.getByTestId('user-rules-result-action-disable-blocked-service'));

        expect(mocks.updateBlockedServices).toHaveBeenCalledWith({ ids: [] });
        expectRecheck('video.example');
    });

    it('updates the resolved persistent client when allowing a client-scoped blocked service', async () => {
        const user = userEvent.setup();

        renderCheckResult(
            {
                hostname: 'video.example',
                reason: FILTERED_STATUS.FILTERED_BLOCKED_SERVICE,
                service_name: 'youtube',
                rules: [{ filter_list_id: 0, text: '||amemv.com^' }],
            },
            {
                dashboard: {
                    clients: [
                        createPersistentClient({
                            ids: ['10.0.0.2'],
                        }),
                    ],
                },
                services: {
                    list: {
                        ids: ['youtube', 'netflix'],
                        schedule: { time_zone: 'UTC' },
                    },
                    allServices: [{ id: 'youtube', name: 'YouTube', rules: ['||amemv.com^'] }],
                },
            },
        );

        await submitCheckForm(user, {
            hostname: 'video.example',
            client: '10.0.0.2',
        });

        mocks.checkHost.mockClear();
        await user.click(screen.getByTestId('user-rules-result-action-disable-blocked-service'));

        expect(mocks.updateClient).toHaveBeenCalledWith(
            expect.objectContaining({
                name: 'office-laptop',
                use_global_blocked_services: false,
                blocked_services: ['netflix'],
                blocked_services_schedule: { time_zone: 'UTC' },
            }),
            'office-laptop',
            { showToast: false, toggleModal: false },
        );
        expect(mocks.updateBlockedServices).not.toHaveBeenCalled();
        expectRecheck('video.example', '10.0.0.2');
    });

    it('does not toast or recheck when allowing a blocked service fails', async () => {
        const user = userEvent.setup();

        mocks.updateBlockedServices.mockResolvedValue(false as never);
        renderCheckResult(
            {
                hostname: 'video.example',
                reason: FILTERED_STATUS.FILTERED_BLOCKED_SERVICE,
                service_name: 'youtube',
                rules: [{ filter_list_id: 0, text: '||amemv.com^' }],
            },
            {
                services: {
                    list: {
                        ids: ['youtube'],
                    },
                    allServices: [{ id: 'youtube', name: 'YouTube', rules: ['||amemv.com^'] }],
                },
            },
        );

        await submitCheckForm(user, {
            hostname: 'video.example',
        });

        mocks.checkHost.mockClear();
        await user.click(screen.getByTestId('user-rules-result-action-disable-blocked-service'));

        expect(mocks.addSuccessToast).not.toHaveBeenCalled();
        expect(mocks.checkHost).not.toHaveBeenCalled();
    });

    it('allows the blocked service even when the result service name does not exactly match the catalog entry', async () => {
        const user = userEvent.setup();

        renderCheckResult(
            {
                hostname: 'video.example',
                reason: FILTERED_STATUS.FILTERED_BLOCKED_SERVICE,
                service_name: 'YouTube (restricted)',
                rules: [{ filter_list_id: 0, text: '||amemv.com^' }],
            },
            {
                services: {
                    list: {
                        ids: ['youtube'],
                    },
                    allServices: [{ id: 'youtube', name: 'YouTube', rules: ['||amemv.com^'] }],
                },
            },
        );

        await submitCheckForm(user, {
            hostname: 'video.example',
        });

        mocks.checkHost.mockClear();
        await user.click(screen.getByTestId('user-rules-result-action-disable-blocked-service'));

        expect(mocks.updateBlockedServices).toHaveBeenCalledWith({ ids: [] });
        expect(mocks.addSuccessToast).toHaveBeenCalled();
        expectRecheck('video.example');
    });

    it('does not show rewrite edit actions for hosts-file rewrites', () => {
        renderCheckResult({
            hostname: 'hosts.example',
            reason: FILTERED_STATUS.REWRITE_HOSTS,
            ip_addrs: ['127.0.0.1'],
        });

        expect(
            screen.queryByTestId('user-rules-result-action-edit-rewrite'),
        ).not.toBeInTheDocument();
    });

    it('does not show rewrite edit actions for dnsrewrite filter rules', () => {
        renderCheckResult({
            hostname: 'rule.example',
            reason: FILTERED_STATUS.REWRITE_RULE,
            cname: 'target.example',
        });

        expect(
            screen.queryByTestId('user-rules-result-action-edit-rewrite'),
        ).not.toBeInTheDocument();
    });

    it('uses the User Rules delete toast when removing a matched rewrite', async () => {
        const user = userEvent.setup();

        renderMatchedRewriteResult();

        expect(screen.getByTestId('user-rules-result-action-edit-rewrite')).toHaveTextContent(
            'Edit DNS rewrite',
        );
        expect(screen.getByTestId('user-rules-result-action-delete-rewrite')).toHaveTextContent(
            'Remove DNS rewrite',
        );

        await user.click(screen.getByTestId('user-rules-result-action-delete-rewrite'));

        expect(mocks.deleteRewrite).toHaveBeenCalledWith(MATCHED_REWRITE, { showToast: false });
        expect(mocks.addSuccessToast).toHaveBeenCalledWith(
            expect.objectContaining({
                message: 'Rule removed from DNS rewrite',
                actionLabel: 'Undo',
            }),
        );
    });

    it('does not toast or recheck when removing a matched rewrite fails', async () => {
        const user = userEvent.setup();

        mocks.deleteRewrite.mockResolvedValue(false as never);
        renderMatchedRewriteResult();

        await user.click(screen.getByTestId('user-rules-result-action-delete-rewrite'));

        expect(mocks.addSuccessToast).not.toHaveBeenCalled();
        expect(mocks.checkHost).not.toHaveBeenCalled();
    });

    it('uses the generic changes-saved toast when editing a matched rewrite', async () => {
        const user = userEvent.setup();

        renderMatchedRewriteResult();

        await user.click(screen.getByTestId('user-rules-result-action-edit-rewrite'));
        const domainInput = await screen.findByTestId('rewrite-domain-input');
        const answerInput = await screen.findByTestId('rewrite-answer-input');

        await user.clear(domainInput);
        await user.type(domainInput, MATCHED_REWRITE.domain);
        await user.clear(answerInput);
        await user.type(answerInput, 'new-target.example');
        await user.click(screen.getByTestId('rewrite-save-button'));

        await waitFor(() => {
            expect(mocks.updateRewrite).toHaveBeenCalledWith(
                {
                    target: MATCHED_REWRITE,
                    update: { ...MATCHED_REWRITE, answer: 'new-target.example' },
                },
                { showToast: false, closeModal: false },
            );
        });
        expect(mocks.addSuccessToast).toHaveBeenCalledWith('Changes saved');
    });

    it('does not toast or recheck when updating a matched rewrite fails', async () => {
        const user = userEvent.setup();

        mocks.updateRewrite.mockResolvedValue(false as never);
        renderMatchedRewriteResult();

        await user.click(screen.getByTestId('user-rules-result-action-edit-rewrite'));
        await user.click(screen.getByTestId('rewrite-save-button'));

        expect(mocks.addSuccessToast).not.toHaveBeenCalled();
        expect(mocks.checkHost).not.toHaveBeenCalled();
    });
});
