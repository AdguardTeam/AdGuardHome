export const LOCAL_STORAGE_KEYS = {
    THEME: 'account_theme',
    BLOCKLIST_PAGE_SIZE: 'blocklist_page_size',
    ALLOWLIST_PAGE_SIZE: 'allowlist_page_size',
    CLIENTS_PAGE_SIZE: 'clients_page_size',
    REWRITES_PAGE_SIZE: 'rewrites_page_size',
    AUTO_CLIENTS_PAGE_SIZE: 'auto_clients_page_size',
};

export const LocalStorageHelper = {
    setItem(key: any, value: any) {
        try {
            localStorage.setItem(key, JSON.stringify(value));
        } catch (error) {
            console.error(`Error setting ${key} in local storage: ${error.message}`);
        }
    },

    getItem(key: any) {
        try {
            const item = localStorage.getItem(key);
            return item ? JSON.parse(item) : null;
        } catch (error) {
            console.error(`Error getting ${key} from local storage: ${error.message}`);
            return null;
        }
    },

    removeItem(key: any) {
        try {
            localStorage.removeItem(key);
        } catch (error) {
            console.error(`Error removing ${key} from local storage: ${error.message}`);
        }
    },

    clear() {
        try {
            localStorage.clear();
        } catch (error) {
            console.error(`Error clearing local storage: ${error.message}`);
        }
    },
};
