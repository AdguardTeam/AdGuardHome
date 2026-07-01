export type BannerSpec =
    | { type: 'tls_expired' }
    | { type: 'tls_expiring' }
    | { type: 'update_auto'; version: string; announcementUrl: string }
    | { type: 'update_manual'; version: string; announcementUrl: string };

/** Keys matching the query param values for each banner type. */
export type BannerType = 'tlsExpired' | 'tlsExpiring' | 'updateAuto' | 'updateManual';

/** Pre-built banner specs for visual QA. Use via `<Banners forceBanner={...}>` or `?forceBanner=updateAuto`. */
export const BANNER_TEST_VALUES: Record<BannerType, BannerSpec> = {
    tlsExpired: { type: 'tls_expired' },
    tlsExpiring: { type: 'tls_expiring' },
    updateAuto: {
        type: 'update_auto',
        version: '1.0',
        announcementUrl: 'https://github.com/AdguardTeam/AdGuardHome/releases',
    },
    updateManual: {
        type: 'update_manual',
        version: '1.0',
        announcementUrl: 'https://github.com/AdguardTeam/AdGuardHome/releases',
    },
};

/**
 * Reads `?forceBanner=<key>` from the current URL query string.
 * Returns the matching banner spec, or null if the param is absent or invalid.
 *
 * Example: `http://localhost:3000/?forceBanner=tlsExpired` → `{ type: 'tls_expired' }`
 */
export const getForceBannerFromQuery = (): BannerSpec | null => {
    // Check both query sources: before hash and inside hash fragment.
    // HashRouter: http://localhost/#/dashboard?forceBanner=tlsExpired
    const sources = [window.location.search, window.location.hash.split('?')[1]].filter(
        Boolean,
    ) as string[];

    for (const src of sources) {
        const key = new URLSearchParams(src).get('forceBanner');
        if (key && key in BANNER_TEST_VALUES) {
            return BANNER_TEST_VALUES[key as BannerType];
        }
    }

    return null;
};

/**
 * Checks whether two banner specs represent the same logical banner.
 * For TLS banners, only the type matters.
 * For update banners, the version and announcement URL must also match.
 */
export const bannerSpecsEqual = (a: BannerSpec, b: BannerSpec | null): b is BannerSpec => {
    if (!b) return false;
    if (a.type !== b.type) return false;
    if (a.type === 'update_auto' || a.type === 'update_manual') {
        const update = b as typeof a;
        return a.version === update.version && a.announcementUrl === update.announcementUrl;
    }
    return true;
};
