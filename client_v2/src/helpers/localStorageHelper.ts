export const LOCAL_STORAGE_KEYS = {
    THEME: 'account_theme',
    BLOCKLIST_PAGE_SIZE: 'blocklist_page_size',
    ALLOWLIST_PAGE_SIZE: 'allowlist_page_size',
    CLIENTS_PAGE_SIZE: 'clients_page_size',
    REWRITES_PAGE_SIZE: 'rewrites_page_size',
    AUTO_CLIENTS_PAGE_SIZE: 'auto_clients_page_size',
    LANGUAGE: 'language',
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
