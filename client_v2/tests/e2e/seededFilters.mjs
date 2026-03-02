/**
 * Filter lists seeded on disk by prepareConfig, so that the tests don't depend
 * on the network.  The far-future last_updated keeps AdGuard Home from
 * refreshing them.
 *
 * One blocklist is left disabled on purpose, to cover assigning a globally
 * disabled list to a single client.
 */
export const SEEDED_FILTERS = [
    { id: 101, name: 'E2E blocklist one', enabled: true, rule: '||e2e-list-one.example^' },
    { id: 102, name: 'E2E blocklist two', enabled: true, rule: '||e2e-list-two.example^' },
    { id: 103, name: 'E2E blocklist disabled', enabled: false, rule: '||e2e-list-three.example^' },
];

export const SEEDED_ALLOW_FILTERS = [
    { id: 110, name: 'E2E allowlist', enabled: true, rule: '@@||e2e-allowed.example^' },
];
