import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { apiClient } from 'panel/api/Api';
import { addErrorToast } from './toasts';
import { HTML_PAGES } from 'panel/helpers/constants';

type LoginState = {
    processingLogin: boolean;
    email: string;
    password: string;
    error: unknown;
};

const initialState: LoginState = {
    processingLogin: false,
    email: '',
    password: '',
    error: null,
};

const [state, setState] = createStore<LoginState>(initialState);

export const processLogin = async (values: { name: string; password: string }) => {
    setState({ processingLogin: true, error: null });
    try {
        await apiClient.login(values);
        const dashboardUrl =
            window.location.origin +
            window.location.pathname.replace(HTML_PAGES.LOGIN, HTML_PAGES.MAIN);
        window.location.replace(dashboardUrl);
        setState({ processingLogin: false, error: null });
    } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        const isInvalidCredentials = message.includes(' | 403');
        if (!isInvalidCredentials) {
            addErrorToast({ error });
        }
        setState({ processingLogin: false, error: isInvalidCredentials ? true : error });
    }
};

export const loginState = untrack(() => state);
