import { handleActions, Action } from 'redux-actions';

import { ClientFormState, getInitialClientFormState } from '../initialState';
import {
    initClientForm,
    updateClientFormField,
    clearClientForm,
    setFormErrors,
    clearFormErrors,
    addClientRequest,
    addClientFailure,
    addClientSuccess,
    updateClientRequest,
    updateClientFailure,
    updateClientSuccess,
} from '../actions/clientForm';

type FieldUpdate = {
    field: keyof ClientFormState;
    value: ClientFormState[keyof ClientFormState];
};

type FormErrors = Record<string, string | string[]>;

const clientForm = handleActions<ClientFormState, any>(
    {
        [initClientForm.toString()]: (
            _state: ClientFormState,
            { payload }: Action<Partial<ClientFormState> | null>,
        ) => {
            if (payload) {
                return {
                    ...getInitialClientFormState(),
                    ...payload,
                    mode: 'edit' as const,
                    originalName: payload.name || '',
                };
            }
            return getInitialClientFormState();
        },

        [updateClientFormField.toString()]: (
            state: ClientFormState,
            { payload }: Action<FieldUpdate>,
        ) => {
            const { field, value } = payload;
            const errors = { ...state.formErrors };
            if (errors[field]) {
                delete errors[field];
            }
            return {
                ...state,
                [field]: value,
                formErrors: errors,
            };
        },

        [clearClientForm.toString()]: () => getInitialClientFormState(),

        [setFormErrors.toString()]: (state: ClientFormState, { payload }: Action<FormErrors>) => ({
            ...state,
            formErrors: payload,
        }),

        [clearFormErrors.toString()]: (state: ClientFormState) => ({
            ...state,
            formErrors: {},
        }),

        [addClientRequest.toString()]: (state: ClientFormState) => ({
            ...state,
            processingSave: true,
        }),
        [addClientFailure.toString()]: (state: ClientFormState) => ({
            ...state,
            processingSave: false,
        }),
        [addClientSuccess.toString()]: (state: ClientFormState) => ({
            ...state,
            processingSave: false,
        }),

        [updateClientRequest.toString()]: (state: ClientFormState) => ({
            ...state,
            processingSave: true,
        }),
        [updateClientFailure.toString()]: (state: ClientFormState) => ({
            ...state,
            processingSave: false,
        }),
        [updateClientSuccess.toString()]: (state: ClientFormState) => ({
            ...state,
            processingSave: false,
        }),
    },
    getInitialClientFormState(),
);

export default clientForm;
