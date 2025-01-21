export type ClientForm = {
    name: string;
    tags: { value: string; label: string }[];
    ids: { name: string }[];
    use_global_settings: boolean;
    use_global_blocked_services: boolean;
    blocked_services_schedule: {
        time_zone: string;
    };
    safe_search: {
        enabled: boolean;
        [key: string]: boolean;
    };
    upstreams: string;
    upstreams_cache_enabled: boolean;
    upstreams_cache_size: number;
    blocked_services: Record<string, boolean>;
    filtering_enabled: boolean;
    safebrowsing_enabled: boolean;
    parental_enabled: boolean;
    ignore_querylog: boolean;
    ignore_statistics: boolean;
};

export type SubmitClientForm = Omit<ClientForm, 'ids' | 'tags'> & {
    ids: string[];
    tags: string[];
};
