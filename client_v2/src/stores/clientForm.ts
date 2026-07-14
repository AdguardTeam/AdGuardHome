import { createStore, reconcile } from 'solid-js/store';
import { untrack } from 'solid-js';
import type { ClientFormState, Client } from 'panel/initialState';
import { apiClient } from 'panel/api/Api';
import intl from 'panel/common/intl';
import { validateIdentifier, validateCacheSize } from 'panel/helpers/validators';
import { DEFAULT_DNS_CACHE_SIZE } from 'panel/helpers/constants';
import { addErrorToast, addSuccessToast } from './toasts';
import { getClients, dashboardState } from './dashboard';

const getInitialClientFormState = (): ClientFormState => ({
    mode: 'add',
    originalName: '',
    name: '',
    ids: [''],
    tags: [],
    use_global_settings: false,
    filtering_enabled: false,
    safebrowsing_enabled: false,
    parental_enabled: false,
    safe_search: {
        enabled: false,
        google: false,
        youtube: false,
        bing: false,
        duckduckgo: false,
        yandex: false,
        pixabay: false,
    },
    ignore_querylog: false,
    ignore_statistics: false,
    blocked_services: [],
    use_global_blocked_services: false,
    blocked_services_schedule: {
        time_zone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    },
    upstreams: '',
    upstreams_cache_enabled: false,
    upstreams_cache_size: DEFAULT_DNS_CACHE_SIZE,
    processingSave: false,
    formErrors: {},
});

const [state, setState] = createStore<ClientFormState>(getInitialClientFormState());

export const initClientForm = (client?: Partial<ClientFormState> | null) => {
    if (client) {
        setState({
            ...getInitialClientFormState(),
            ...client,
            mode: 'edit' as const,
            originalName: client.name || '',
        });
    } else {
        setState(getInitialClientFormState());
    }
};

export const updateClientFormField = (
    fieldOrObj: keyof ClientFormState | { field: keyof ClientFormState; value: any },
    maybeValue?: any,
    replace?: boolean,
) => {
    const field = typeof fieldOrObj === 'string' ? fieldOrObj : fieldOrObj.field;
    const value = typeof fieldOrObj === 'string' ? maybeValue : fieldOrObj.value;
    setState(field as any, replace ? reconcile(value) : value);
    // Clear the error for this field
    if (state.formErrors[field as string]) {
        setState('formErrors', (prev) => {
            const next = { ...prev };
            delete next[field as string];
            return next;
        });
    }
};

export const clearClientForm = () => {
    setState(getInitialClientFormState());
};

export const setFormErrors = (errors: Record<string, string | string[]>) => {
    setState('formErrors', errors);
};

export const clearFormErrors = () => {
    setState('formErrors', {});
};

export const setProcessingSave = (value: boolean) => {
    setState('processingSave', value);
};

export const buildFormPayload = (client: Client): Partial<ClientFormState> => ({
    name: client.name,
    ids: client.ids || [''],
    tags: client.tags || [],
    use_global_settings: client.use_global_settings || false,
    filtering_enabled: client.filtering_enabled || false,
    safebrowsing_enabled: client.safebrowsing_enabled || false,
    parental_enabled: client.parental_enabled || false,
    safe_search: (client.safe_search as any) || {
        enabled: false,
        google: false,
        youtube: false,
        bing: false,
        duckduckgo: false,
        yandex: false,
        pixabay: false,
    },
    ignore_querylog: client.ignore_querylog || false,
    ignore_statistics: client.ignore_statistics || false,
    blocked_services: client.blocked_services || [],
    use_global_blocked_services: client.use_global_blocked_services || false,
    blocked_services_schedule: client.blocked_services_schedule || {
        time_zone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    },
    upstreams: (client.upstreams || []).join('\n'),
    upstreams_cache_enabled: client.upstreams_cache_enabled || false,
    upstreams_cache_size: client.upstreams_cache_size || DEFAULT_DNS_CACHE_SIZE,
});

export const buildClientConfig = (form: ClientFormState) => ({
    name: form.name,
    ids: form.ids.filter((id: string) => id.trim() !== ''),
    tags: form.tags,
    use_global_settings: form.use_global_settings,
    use_global_blocked_services: form.use_global_settings,
    filtering_enabled: form.filtering_enabled,
    safebrowsing_enabled: form.safebrowsing_enabled,
    parental_enabled: form.parental_enabled,
    safe_search: form.safe_search,
    ignore_querylog: form.ignore_querylog,
    ignore_statistics: form.ignore_statistics,
    blocked_services: form.blocked_services,
    blocked_services_schedule: form.blocked_services_schedule,
    upstreams: form.upstreams
        ? form.upstreams.split('\n').filter((line: string) => line.trim() !== '')
        : [],
    upstreams_cache_enabled: form.upstreams_cache_enabled,
    upstreams_cache_size: form.upstreams_cache_size,
});

/**
 * Returns all identifier strings from other persistent clients, excluding the
 * client currently being edited.
 */
export const computeExistingClientIds = (): string[] =>
    (dashboardState.clients || [])
        .filter((c: Client) => state.mode !== 'edit' || c.name !== state.originalName)
        .flatMap((c: Client) => c.ids);

export const computeExistingClientNames = (): string[] =>
    (dashboardState.clients || [])
        .filter((c: Client) => state.mode !== 'edit' || c.name !== state.originalName)
        .map((c: Client) => c.name);

export const saveClient = async (): Promise<boolean> => {
    const errors: Record<string, string | string[]> = {};

    if (!state.name.trim()) {
        errors.name = intl.getMessage('form_error_required');
    }

    if (!errors.name) {
        const existingClientNames = computeExistingClientNames();
        if (existingClientNames.includes(state.name.trim())) {
            errors.name = intl.getMessage('client_name_already_exists');
        }
    }

    const existingClientIds = computeExistingClientIds();

    const idErrors = state.ids.map((id: string, index: number) => {
        if (!id.trim()) {
            return intl.getMessage('form_error_required');
        }
        return validateIdentifier(id, state.ids, index, existingClientIds);
    });
    if (idErrors.some((e: string | undefined) => e !== undefined)) {
        errors.ids = idErrors as string[];
    }

    // Validate cache size when per-client cache is enabled (and not using global settings).
    // When use_global_settings is true, upstream settings are inherited; the cache size
    // value is still sent to the API but the backend ignores it. We skip validation to
    // avoid false positives on stale/inherited values.
    if (!state.use_global_settings && state.upstreams_cache_enabled) {
        const cacheErr = validateCacheSize(state.upstreams_cache_size, true);
        if (cacheErr) {
            errors.upstreams_cache_size = cacheErr;
        }
    }

    if (Object.keys(errors).length > 0) {
        setFormErrors(errors);
        return false;
    }

    clearFormErrors();
    const config = buildClientConfig(state);

    if (state.mode === 'edit') {
        setProcessingSave(true);
        try {
            await apiClient.updateClient({ name: state.originalName, data: config });
            clearClientForm();
            await getClients();
            return true;
        } catch (error) {
            addErrorToast({ error });
            return false;
        } finally {
            setProcessingSave(false);
        }
    } else {
        setProcessingSave(true);
        try {
            await apiClient.addClient(config);
            clearClientForm();
            await getClients();
            addSuccessToast({ message: intl.getMessage('client_added') });
            return true;
        } catch (error) {
            addErrorToast({ error });
            return false;
        } finally {
            setProcessingSave(false);
        }
    }
};

export const clientFormState = untrack(() => state);
