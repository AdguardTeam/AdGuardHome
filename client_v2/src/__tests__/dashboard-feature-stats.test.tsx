import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@solidjs/testing-library';

const mocks = vi.hoisted(() => ({
    initSettings: vi.fn(),
    getStats: vi.fn(),
    getStatsConfig: vi.fn(),
    getClients: vi.fn(),
    getAccessList: vi.fn(),
    toggleProtection: vi.fn(),
    setSettingsState: vi.fn(),
    statsState: {
        processingStats: false,
        processingGetConfig: false,
        interval: 24 * 60 * 60 * 1000,
        enabled: true,
        numDnsQueries: 10,
        numBlockedFiltering: 2,
        numReplacedSafebrowsing: 0,
        numReplacedParental: 0,
        numReplacedSafesearch: 0,
        avgProcessingTime: 1,
        dnsQueries: [10],
        blockedFiltering: [2],
        replacedSafebrowsing: [] as number[],
        replacedParental: [] as number[],
        topClients: [] as never[],
        topQueriedDomains: [] as never[],
        topBlockedDomains: [] as never[],
        topUpstreamsResponses: [] as never[],
        topUpstreamsAvgTime: [] as never[],
    },
}));

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string) => key,
        getPlural: (key: string) => key,
        getUILanguage: () => 'en',
    },
}));

vi.mock('panel/lib/theme', () => ({
    default: {
        layout: { container: '', containerIn: '' },
        title: { h5: '' },
        text: { t3: '' },
    },
}));

vi.mock('panel/common/ui/Loader', () => ({
    PageLoader: () => <div data-testid="page-loader" />,
}));

vi.mock('panel/stores/dashboard', () => ({
    dashboardState: {
        protectionEnabled: true,
        protectionDisabledDuration: null,
        processingProtection: false,
    },
    toggleProtection: mocks.toggleProtection,
    getClients: mocks.getClients,
}));

vi.mock('panel/stores/stats', () => ({
    get statsState() {
        return mocks.statsState;
    },
    getStats: mocks.getStats,
    getStatsConfig: mocks.getStatsConfig,
}));

vi.mock('panel/stores/access', () => ({
    accessState: { processing: false },
    getAccessList: mocks.getAccessList,
}));

vi.mock('panel/stores/settings', async () => {
    const { createStore } = await import('solid-js/store');
    const [settingsState, setSettingsState] = createStore({
        loadStatus: 'idle' as FeatureLoadStatus,
        processing: true,
        settingsList: {
            safebrowsing: { enabled: false as boolean | undefined },
            parental: { enabled: false as boolean | undefined },
            safesearch: { enabled: false as boolean | undefined },
        },
    });

    mocks.setSettingsState.mockImplementation((state: FeatureState) => {
        setSettingsState(state);
    });

    return {
        settingsState,
        initSettings: mocks.initSettings,
    };
});

vi.mock('panel/components/Dashboard/blocks/Header/Header', () => ({
    Header: (props: { onPeriodChange: (period: number) => void }) => (
        <button type="button" onClick={() => props.onPeriodChange(2 * 24 * 60 * 60 * 1000)}>
            change period
        </button>
    ),
    getPeriodLabel: (period: number) => String(period),
}));

vi.mock('panel/components/Dashboard/blocks/StatCard', () => ({
    CARDS_THEME: {
        QUERIES: 'queries',
        ADS: 'ads',
        THREATS: 'threats',
        ADULT: 'adult',
    },
    CARDS_COLORS: {
        QUERIES: '',
        ADS: '',
        THREATS: '',
        ADULT: '',
    },
    StatCard: (props: { cardTheme: string }) => (
        <div data-testid={`stat-card-${props.cardTheme}`} />
    ),
}));

vi.mock('panel/components/Dashboard/blocks/StatRow', () => ({
    StatRow: (props: { rowTheme: string }) => <div data-testid={`stat-row-${props.rowTheme}`} />,
}));

vi.mock('panel/components/Dashboard/blocks/EmptyState', () => ({
    EmptyState: () => <div data-testid="empty-state" />,
}));

vi.mock('panel/components/Dashboard/blocks/TopClients', () => ({
    TopClients: (): null => null,
}));
vi.mock('panel/components/Dashboard/blocks/TopQueriedDomains', () => ({
    TopQueriedDomains: (): null => null,
}));
vi.mock('panel/components/Dashboard/blocks/TopBlockedDomains', () => ({
    TopBlockedDomains: (): null => null,
}));
vi.mock('panel/components/Dashboard/blocks/TopUpstreams', () => ({
    TopUpstreams: (): null => null,
}));
vi.mock('panel/components/Dashboard/blocks/UpstreamAvgTime', () => ({
    UpstreamAvgTime: (): null => null,
}));

import { Dashboard } from 'panel/components/Dashboard/Dashboard';

type FeatureLoadStatus = 'idle' | 'loading' | 'loaded' | 'failed';

type FeatureState = {
    loadStatus: FeatureLoadStatus;
    processing: boolean;
    settingsList: {
        safebrowsing: { enabled: boolean | undefined };
        parental: { enabled: boolean | undefined };
        safesearch: { enabled: boolean | undefined };
    };
};

const setFeatureState = (
    loadStatus: FeatureLoadStatus,
    enabled: {
        safebrowsing: boolean | undefined;
        parental: boolean | undefined;
        safesearch: boolean | undefined;
    },
) => {
    mocks.setSettingsState({
        loadStatus,
        processing: loadStatus === 'idle' || loadStatus === 'loading',
        settingsList: {
            safebrowsing: { enabled: enabled.safebrowsing },
            parental: { enabled: enabled.parental },
            safesearch: { enabled: enabled.safesearch },
        },
    });
};

const setFeatureMetrics = ({
    safebrowsingScalar = 0,
    safebrowsingSeries = [],
    parentalScalar = 0,
    parentalSeries = [],
    safesearchScalar = 0,
}: {
    safebrowsingScalar?: number;
    safebrowsingSeries?: number[];
    parentalScalar?: number;
    parentalSeries?: number[];
    safesearchScalar?: number;
} = {}) => {
    mocks.statsState.numReplacedSafebrowsing = safebrowsingScalar;
    mocks.statsState.replacedSafebrowsing = safebrowsingSeries;
    mocks.statsState.numReplacedParental = parentalScalar;
    mocks.statsState.replacedParental = parentalSeries;
    mocks.statsState.numReplacedSafesearch = safesearchScalar;
};

const deferred = <T,>() => {
    let resolve!: (value: T | PromiseLike<T>) => void;
    let reject!: (reason?: unknown) => void;
    const promise = new Promise<T>((res, rej) => {
        resolve = res;
        reject = rej;
    });

    return { promise, reject, resolve };
};

const expectCoreStats = () => {
    expect(screen.getByTestId('stat-card-queries')).toBeInTheDocument();
    expect(screen.getByTestId('stat-card-ads')).toBeInTheDocument();
    expect(screen.getByTestId('stat-row-dnsQueries')).toBeInTheDocument();
    expect(screen.getByTestId('stat-row-adsBlocked')).toBeInTheDocument();
    expect(screen.getByTestId('stat-row-averageProcessingTime')).toBeInTheDocument();
};

const expectAllFeatureStats = () => {
    expect(screen.getByTestId('stat-card-threats')).toBeInTheDocument();
    expect(screen.getByTestId('stat-card-adult')).toBeInTheDocument();
    expect(screen.getByTestId('stat-row-threatsBlocked')).toBeInTheDocument();
    expect(screen.getByTestId('stat-row-adultWebsitesBlocked')).toBeInTheDocument();
    expect(screen.getByTestId('stat-row-safeSearchUsed')).toBeInTheDocument();
};

describe('Dashboard feature statistics', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mocks.initSettings.mockReset();
        setFeatureMetrics();
        setFeatureState('idle', {
            safebrowsing: false,
            parental: false,
            safesearch: false,
        });
    });

    it.each(['idle', 'loading', 'failed'] as const)(
        'keeps feature statistics visible while status is %s',
        (loadStatus) => {
            setFeatureState(loadStatus, {
                safebrowsing: false,
                parental: false,
                safesearch: false,
            });

            render(() => <Dashboard />);

            expectCoreStats();
            expectAllFeatureStats();
        },
    );

    it('keeps enabled feature statistics visible after status loads', () => {
        setFeatureState('loaded', {
            safebrowsing: true,
            parental: true,
            safesearch: true,
        });

        render(() => <Dashboard />);

        expectCoreStats();
        expectAllFeatureStats();
    });

    it('keeps feature statistics visible when a loaded response omits a status', () => {
        setFeatureState('loaded', {
            safebrowsing: undefined,
            parental: undefined,
            safesearch: undefined,
        });

        render(() => <Dashboard />);

        expectCoreStats();
        expectAllFeatureStats();
    });

    it('hides disabled feature statistics after status loads', () => {
        setFeatureState('loaded', {
            safebrowsing: false,
            parental: false,
            safesearch: false,
        });

        render(() => <Dashboard />);

        expectCoreStats();
        expect(screen.queryByTestId('stat-card-threats')).not.toBeInTheDocument();
        expect(screen.queryByTestId('stat-card-adult')).not.toBeInTheDocument();
        expect(screen.queryByTestId('stat-row-threatsBlocked')).not.toBeInTheDocument();
        expect(screen.queryByTestId('stat-row-adultWebsitesBlocked')).not.toBeInTheDocument();
        expect(screen.queryByTestId('stat-row-safeSearchUsed')).not.toBeInTheDocument();
    });

    it('keeps disabled feature statistics visible when their scalar count is nonzero', () => {
        setFeatureMetrics({
            safebrowsingScalar: 1,
            parentalScalar: 1,
            safesearchScalar: 1,
        });
        setFeatureState('loaded', {
            safebrowsing: false,
            parental: false,
            safesearch: false,
        });

        render(() => <Dashboard />);

        expectCoreStats();
        expectAllFeatureStats();
    });

    it('keeps disabled feature statistics visible when their series contains data', () => {
        setFeatureMetrics({
            safebrowsingSeries: [0, 1],
            parentalSeries: [1, 0],
        });
        setFeatureState('loaded', {
            safebrowsing: false,
            parental: false,
            safesearch: false,
        });

        render(() => <Dashboard />);

        expectCoreStats();
        expect(screen.getByTestId('stat-card-threats')).toBeInTheDocument();
        expect(screen.getByTestId('stat-card-adult')).toBeInTheDocument();
        expect(screen.getByTestId('stat-row-threatsBlocked')).toBeInTheDocument();
        expect(screen.getByTestId('stat-row-adultWebsitesBlocked')).toBeInTheDocument();
        expect(screen.queryByTestId('stat-row-safeSearchUsed')).not.toBeInTheDocument();
    });

    it('hides only disabled feature statistics in a mixed state', () => {
        setFeatureState('loaded', {
            safebrowsing: true,
            parental: false,
            safesearch: true,
        });

        render(() => <Dashboard />);

        expectCoreStats();
        expect(screen.getByTestId('stat-card-threats')).toBeInTheDocument();
        expect(screen.queryByTestId('stat-card-adult')).not.toBeInTheDocument();
        expect(screen.getByTestId('stat-row-threatsBlocked')).toBeInTheDocument();
        expect(screen.queryByTestId('stat-row-adultWebsitesBlocked')).not.toBeInTheDocument();
        expect(screen.getByTestId('stat-row-safeSearchUsed')).toBeInTheDocument();
    });

    it('initializes feature status once without refetching it for a period change', () => {
        render(() => <Dashboard />);

        expect(mocks.initSettings).toHaveBeenCalledOnce();
        fireEvent.click(screen.getByRole('button', { name: 'change period' }));
        expect(mocks.initSettings).toHaveBeenCalledOnce();
    });

    it.each(['loading', 'loaded'] as const)(
        'does not duplicate feature status initialization while status is %s',
        (loadStatus) => {
            setFeatureState(loadStatus, {
                safebrowsing: false,
                parental: false,
                safesearch: false,
            });

            render(() => <Dashboard />);

            expect(mocks.initSettings).not.toHaveBeenCalled();
        },
    );

    it('reactively hides zero-valued feature statistics after disabled status loads', async () => {
        const status = deferred<void>();
        mocks.initSettings.mockImplementation(async () => {
            setFeatureState('loading', {
                safebrowsing: false,
                parental: false,
                safesearch: false,
            });
            await status.promise;
            setFeatureState('loaded', {
                safebrowsing: false,
                parental: false,
                safesearch: false,
            });
        });

        render(() => <Dashboard />);

        expectAllFeatureStats();
        const request = mocks.initSettings.mock.results[0].value as Promise<void>;
        status.resolve();
        await request;

        await waitFor(() => {
            expect(screen.queryByTestId('stat-card-threats')).not.toBeInTheDocument();
            expect(screen.queryByTestId('stat-card-adult')).not.toBeInTheDocument();
            expect(screen.queryByTestId('stat-row-threatsBlocked')).not.toBeInTheDocument();
            expect(screen.queryByTestId('stat-row-adultWebsitesBlocked')).not.toBeInTheDocument();
            expect(screen.queryByTestId('stat-row-safeSearchUsed')).not.toBeInTheDocument();
        });
    });

    it('keeps zero-valued feature statistics visible when status loading fails', async () => {
        const status = deferred<void>();
        mocks.initSettings.mockImplementation(async () => {
            setFeatureState('loading', {
                safebrowsing: false,
                parental: false,
                safesearch: false,
            });
            await status.promise;
            setFeatureState('failed', {
                safebrowsing: false,
                parental: false,
                safesearch: false,
            });
        });

        render(() => <Dashboard />);

        expectAllFeatureStats();
        const request = mocks.initSettings.mock.results[0].value as Promise<void>;
        status.resolve();
        await request;

        await waitFor(() => {
            expectAllFeatureStats();
        });
    });
});
