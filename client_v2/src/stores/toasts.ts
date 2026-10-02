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
type SuccessToastPayload =
    | string
    | {
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
    noIcon?: boolean;
};

export type ToastNotice = {
    id: string;
    message: string;
    type: 'error' | 'success' | 'notice' | 'warning';
    actionLabel?: string;
    undoId?: string;
    action?: ToastAction;
    options?: Record<string, unknown>;
    code?: string;
    noIcon?: boolean;
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

/**
 * Adds a problem notice, collapsing a repeat of the same problem instead of
 * stacking a copy — a request that keeps failing must not flood the screen with
 * identical toasts.  Notices carrying an action always get their own entry
 * because the action belongs to that particular occurrence.
 */
const pushProblemToast = (type: 'error' | 'warning', payload: ErrorToastPayload) => {
    const { error, options, action, noIcon } = payload;
    const message = error instanceof Error ? error.message : String(error);

    if (type === 'error') {
        console.error(message); // eslint-disable-line no-console
    }

    const notice: ToastNotice = {
        id: nanoid(),
        message,
        options,
        type,
        noIcon,
    };
    if (action) {
        notice.action = action;
    }

    setState('notices', (prev) => {
        const isDuplicate = (n: ToastNotice) =>
            !action && n.type === type && n.message === message && !n.action && !n.undoId;

        if (prev.some(isDuplicate)) {
            // The fresh id makes the toast render as a new notice, which
            // restarts its dismiss timer so the user gets the full time to read
            // the repeated message.
            return prev.map((n) => (isDuplicate(n) ? notice : n));
        }

        return [...prev, notice];
    });
};

export const addErrorToast = (payload: ErrorToastPayload) => pushProblemToast('error', payload);

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

export const addWarningToast = (payload: ErrorToastPayload) => pushProblemToast('warning', payload);

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
