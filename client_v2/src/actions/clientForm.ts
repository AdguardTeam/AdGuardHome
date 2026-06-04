import { createAction } from 'redux-actions';
import intl from 'panel/common/intl';
import { apiClient } from '../api/Api';
import { Client, ClientFormState, RootState } from '../initialState';
import { validateIdentifier } from '../helpers/validators';
import { addErrorToast, addSuccessToast } from './toasts';
import { getClients } from './index';

type AppDispatch = (action: unknown) => unknown;
type GetState = () => RootState;

export const initClientForm = createAction('INIT_CLIENT_FORM');
export const updateClientFormField = createAction('UPDATE_CLIENT_FORM_FIELD');
export const clearClientForm = createAction('CLEAR_CLIENT_FORM');
export const setFormErrors = createAction('SET_FORM_ERRORS');
export const clearFormErrors = createAction('CLEAR_FORM_ERRORS');

export const addClientRequest = createAction('ADD_CLIENT_FORM_REQUEST');
export const addClientFailure = createAction('ADD_CLIENT_FORM_FAILURE');
export const addClientSuccess = createAction('ADD_CLIENT_FORM_SUCCESS');

export const updateClientRequest = createAction('UPDATE_CLIENT_FORM_REQUEST');
export const updateClientFailure = createAction('UPDATE_CLIENT_FORM_FAILURE');
export const updateClientSuccess = createAction('UPDATE_CLIENT_FORM_SUCCESS');

export const buildFormPayload = (client: Client) => ({
    name: client.name,
    ids: client.ids || [''],
    tags: client.tags || [],
    use_global_settings: client.use_global_settings || false,
    filtering_enabled: client.filtering_enabled || false,
    safebrowsing_enabled: client.safebrowsing_enabled || false,
    parental_enabled: client.parental_enabled || false,
    safe_search: client.safe_search || {
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
    upstreams_cache_size: client.upstreams_cache_size || 0,
});

export const buildClientConfig = (form: ClientFormState) => {
    return {
        name: form.name,
        ids: form.ids.filter((id: string) => id.trim() !== ''),
        tags: form.tags,
        use_global_settings: form.use_global_settings,
        filtering_enabled: form.filtering_enabled,
        safebrowsing_enabled: form.safebrowsing_enabled,
        parental_enabled: form.parental_enabled,
        safe_search: form.safe_search,
        ignore_querylog: form.ignore_querylog,
        ignore_statistics: form.ignore_statistics,
        blocked_services: form.blocked_services,
        use_global_blocked_services: form.use_global_blocked_services,
        blocked_services_schedule: form.blocked_services_schedule,
        upstreams: form.upstreams
            ? form.upstreams.split('\n').filter((line: string) => line.trim() !== '')
            : [],
        upstreams_cache_enabled: form.upstreams_cache_enabled,
        upstreams_cache_size: form.upstreams_cache_size,
    };
};

export const saveClient =
    () =>
    async (dispatch: AppDispatch, getState: GetState): Promise<boolean> => {
        const { clientForm } = getState();
        const errors: Record<string, string | string[]> = {};

        // Validate client name
        if (!clientForm.name.trim()) {
            errors.name = intl.getMessage('form_error_required');
        }

        // Validate identifiers
        const idErrors: (string | undefined)[] = clientForm.ids.map((id: string, index: number) => {
            if (!id.trim()) {
                return intl.getMessage('form_error_required');
            }
            return validateIdentifier(id, clientForm.ids, index);
        });
        if (idErrors.some((e: string | undefined) => e !== undefined)) {
            errors.ids = idErrors;
        }

        if (Object.keys(errors).length > 0) {
            dispatch(setFormErrors(errors));
            return false;
        }

        dispatch(clearFormErrors());
        const config = buildClientConfig(clientForm);

        if (clientForm.mode === 'edit') {
            dispatch(updateClientRequest());
            try {
                await apiClient.updateClient({
                    name: clientForm.originalName,
                    data: config,
                });
                dispatch(updateClientSuccess());
                dispatch(clearClientForm());
                dispatch(
                    addSuccessToast(intl.getMessage('client_updated', { key: clientForm.name })),
                );
                dispatch(getClients());
                return true;
            } catch (error) {
                dispatch(addErrorToast({ error }));
                dispatch(updateClientFailure());
                return false;
            }
        } else {
            dispatch(addClientRequest());
            try {
                await apiClient.addClient(config);
                dispatch(addClientSuccess());
                dispatch(clearClientForm());
                dispatch(
                    addSuccessToast(intl.getMessage('client_added', { key: clientForm.name })),
                );
                dispatch(getClients());
                return true;
            } catch (error) {
                dispatch(addErrorToast({ error }));
                dispatch(addClientFailure());
                return false;
            }
        }
    };
