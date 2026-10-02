export const LOCAL_STORAGE_KEYS = {
    THEME: 'account_theme',
    BLOCKLIST_PAGE_SIZE: 'blocklist_page_size',
    ALLOWLIST_PAGE_SIZE: 'allowlist_page_size',
    CLIENTS_PAGE_SIZE: 'clients_page_size',
    REWRITES_PAGE_SIZE: 'rewrites_page_size',
    AUTO_CLIENTS_PAGE_SIZE: 'auto_clients_page_size',
    LANGUAGE: 'language',
    STATS_PERIOD: 'stats_period',
    TOP_CLIENTS_PAGE_SIZE: 'top_clients_page_size',
    TOP_QUERIED_DOMAINS_PAGE_SIZE: 'top_queried_domains_page_size',
    TOP_BLOCKED_DOMAINS_PAGE_SIZE: 'top_blocked_domains_page_size',
    TOP_UPSTREAMS_PAGE_SIZE: 'top_upstreams_page_size',
    UPSTREAM_AVG_TIME_PAGE_SIZE: 'upstream_avg_time_page_size',
    TOP_CLIENTS_SORT: 'top_clients_sort',
    TOP_QUERIED_DOMAINS_SORT: 'top_queried_domains_sort',
    TOP_BLOCKED_DOMAINS_SORT: 'top_blocked_domains_sort',
    TOP_UPSTREAMS_SORT: 'top_upstreams_sort',
    UPSTREAM_AVG_TIME_SORT: 'upstream_avg_time_sort',
};

export const LocalStorageHelper = {
    setItem(key: string, value: unknown) {
        try {
            localStorage.setItem(key, JSON.stringify(value));
        } catch (error) {
            console.error(`Error setting ${key} in local storage: ${(error as Error).message}`);
        }
    },

    getItem<T = unknown>(key: string): T | null {
        try {
            const item = localStorage.getItem(key);
            return item ? (JSON.parse(item) as T) : null;
        } catch (error) {
            console.error(`Error getting ${key} from local storage: ${(error as Error).message}`);
            return null;
        }
    },

    removeItem(key: string) {
        try {
            localStorage.removeItem(key);
        } catch (error) {
            console.error(`Error removing ${key} from local storage: ${(error as Error).message}`);
        }
    },

    clear() {
        try {
            localStorage.clear();
        } catch (error) {
            console.error(`Error clearing local storage: ${(error as Error).message}`);
        }
    },
};
