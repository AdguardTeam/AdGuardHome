import { createStore } from 'solid-js/store';
import { untrack } from 'solid-js';
import { nanoid } from 'nanoid';

type ToastAction = {
    text: string;
    actionType?: string;
    actionPayload?: unknown;
    callback?: () => void;
};

/** Payload accepted by addSuccessToast. */
type SuccessToastPayload = string | {
    message: string;
    code?: string;
    actionLabel?: string;
    undoId?: string;
};

/** Payload accepted by addErrorToast / addWarningToast. */
type ErrorToastPayload = {
    error: unknown;
    options?: Record<string, unknown>;
    action?: ToastAction;
};

type ToastNotice = {
    id: string;
    message: string;
    type: 'error' | 'success' | 'notice' | 'warning';
    actionLabel?: string;
    undoId?: string;
    action?: ToastAction;
    options?: Record<string, unknown>;
    code?: string;
};

type ToastsState = {
    notices: ToastNotice[];
};

const initialState: ToastsState = {
    notices: [],
};

const [state, setState] = createStore<ToastsState>(initialState);

type UndoCallback = () => void | Promise<void>;

const undoRegistry = new Map<string, UndoCallback>();

export const getUndoCallback = (id: string): UndoCallback | undefined => undoRegistry.get(id);

export const clearUndoCallback = (id: string): void => {
    undoRegistry.delete(id);
};

export const createUndoToast = (
    message: string,
    actionLabel: string,
    onUndo: UndoCallback,
): { message: string; actionLabel: string; undoId: string } => {
    const undoId = nanoid();
    undoRegistry.set(undoId, onUndo);
    return { message, actionLabel, undoId };
};

export const addErrorToast = (payload: ErrorToastPayload) => {
    const { error, options, action } = payload;
    const message = error instanceof Error ? error.message : String(error);
    console.error(message); // eslint-disable-line no-console
    const notice: ToastNotice = {
        id: nanoid(),
        message,
        options,
        type: 'error' as const,
    };
    if (action) {
        notice.action = action;
    }
    setState('notices', (prev) => [...prev, notice]);
};

export const addSuccessToast = (message: SuccessToastPayload) => {
    const notice: ToastNotice = {
        id: nanoid(),
        type: 'success',
        message: typeof message === 'string' ? message : message.message,
    };
    if (typeof message === 'object') {
        notice.actionLabel = message.actionLabel;
        notice.undoId = message.undoId;
        notice.code = message.code;
    }
    setState('notices', (prev) => [...prev, notice]);
};

export const addWarningToast = (payload: ErrorToastPayload) => {
    const { error, options, action } = payload;
    const message = error instanceof Error ? error.message : String(error);
    const notice: ToastNotice = {
        id: nanoid(),
        message,
        options,
        type: 'warning' as const,
    };
    if (action) {
        notice.action = action;
    }
    setState('notices', (prev) => [...prev, notice]);
};

export const addNoticeToast = (message: string) => {
    setState('notices', (prev) => [
        ...prev,
        {
            id: nanoid(),
            message,
            type: 'notice' as const,
        },
    ]);
};

export const removeToast = (id: string) => {
    setState('notices', (prev) => prev.filter((n) => n.id !== id));
};

export const toastsState = untrack(() => state);
